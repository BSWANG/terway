package driver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	fileLockTimeOut = 11 * time.Second
)

func deleteRoutesForAddr(addr *net.IPNet, tableID int) error {
	routeList, err := netlink.RouteListFiltered(netlink.FAMILY_ALL, &netlink.Route{
		Dst:   addr,
		Table: tableID,
	}, netlink.RT_FILTER_DST|netlink.RT_FILTER_TABLE)
	if err != nil {
		return errors.Wrapf(err, "error get route list")
	}

	for _, route := range routeList {
		err = netlink.RouteDel(&route)
		if err != nil {
			return errors.Wrapf(err, "error cleanup route: %v", route)
		}
	}
	return nil
}

// add 1000 to link index to avoid route table conflict
func getRouteTableID(linkIndex int) int {
	return 1000 + linkIndex
}

// ipNetEqual returns true iff both IPNet are equal
func ipNetEqual(ipn1 *net.IPNet, ipn2 *net.IPNet) bool {
	if ipn1 == ipn2 {
		return true
	}
	if ipn1 == nil || ipn2 == nil {
		return false
	}
	m1, _ := ipn1.Mask.Size()
	m2, _ := ipn2.Mask.Size()
	return m1 == m2 && ipn1.IP.Equal(ipn2.IP)
}

const rpFilterSysctl = "net.ipv4.conf.%s.rp_filter"

// EnsureHostNsConfig setup host namespace configs
func EnsureHostNsConfig() error {
	existInterfaces, err := net.Interfaces()
	if err != nil {
		return errors.Wrapf(err, "error get exist interfaces on system")
	}

	for _, key := range []string{"default", "all"} {
		sysctlName := fmt.Sprintf(rpFilterSysctl, key)
		if _, err = sysctl.Sysctl(sysctlName, "0"); err != nil {
			return errors.Wrapf(err, "error set: %s sysctl value to 0", sysctlName)
		}
	}

	for _, existIf := range existInterfaces {
		sysctlName := fmt.Sprintf(rpFilterSysctl, existIf.Name)
		sysctlValue, err := sysctl.Sysctl(sysctlName)
		if err != nil {
			continue
		}
		if sysctlValue != "0" {
			if _, err = sysctl.Sysctl(sysctlName, "0"); err != nil {
				return errors.Wrapf(err, "error set: %s sysctl value to 0", sysctlName)
			}
		}
	}
	return nil
}

// EnsureLinkUp set link up,return changed and err
func EnsureLinkUp(link netlink.Link) (bool, error) {
	if link.Attrs().Flags&net.FlagUp != 0 {
		return false, nil
	}
	return true, netlink.LinkSetUp(link)
}

// EnsureDefaultRoute
func EnsureDefaultRoute(link netlink.Link, gw net.IP) (bool, error) {
	err := ip.ValidateExpectedRoute([]*types.Route{
		{
			Dst: *defaultRoute,
			GW:  gw,
		},
	})
	if err == nil {
		return false, nil
	}

	if !strings.Contains(err.Error(), "not found") {
		return false, err
	}

	err = netlink.RouteReplace(&netlink.Route{
		LinkIndex: link.Attrs().Index,
		Scope:     netlink.SCOPE_UNIVERSE,
		Flags:     int(netlink.FLAG_ONLINK),
		Dst:       defaultRoute,
		Gw:        gw,
	})
	return true, err
}

var Log = MyLog{
	l: log.New(ioutil.Discard, "", log.LstdFlags),
}

type MyLog struct {
	l     *log.Logger
	debug bool
}

// Debugf
func (m *MyLog) Debugf(format string, v ...interface{}) {
	if !m.debug {
		return
	}
	m.l.Printf(format, v...)
}

// Debug
func (m *MyLog) Debug(v ...interface{}) {
	if !m.debug {
		return
	}
	m.l.Print(v...)
}

// SetDebug
func (m *MyLog) SetDebug(d bool, fd *os.File) {
	if !d {
		m.l.SetOutput(ioutil.Discard)
		return
	}
	m.debug = true
	m.l.SetOutput(fd)
}

// JSONStr
func JSONStr(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

type Locker struct {
	FD *os.File
}

// Close
func (l *Locker) Close() error {
	if l.FD != nil {
		return l.FD.Close()
	}
	return nil
}

// GrabFileLock get file lock with timeout 11seconds
func GrabFileLock(lockfilePath string) (*Locker, error) {
	var success bool
	var err error
	l := &Locker{}
	defer func(l *Locker) {
		if !success {
			l.Close()
		}
	}(l)

	l.FD, err = os.OpenFile(lockfilePath, os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock %s: %v", lockfilePath, err)
	}
	if err := wait.PollImmediate(200*time.Millisecond, fileLockTimeOut, func() (bool, error) {
		if err := grabFileLock(l.FD); err != nil {
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to acquire new iptables lock: %v", err)
	}
	success = true
	return l, nil
}

func grabFileLock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}
