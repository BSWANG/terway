package eni

import (
	"context"
	"net/netip"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/AliyunContainerService/terway/pkg/factory"
	"github.com/AliyunContainerService/terway/types"
	"github.com/AliyunContainerService/terway/types/daemon"
)

func NewLocalTest(eni *daemon.ENI, factory factory.Factory, poolConfig *types.PoolConfig) *Local {
	l := &Local{
		eni:        eni,
		batchSize:  poolConfig.BatchSize,
		cap:        poolConfig.MaxIPPerENI,
		cond:       sync.NewCond(&sync.Mutex{}),
		ipv4:       make(Set),
		ipv6:       make(Set),
		enableIPv4: poolConfig.EnableIPv4,
		enableIPv6: poolConfig.EnableIPv6,
		factory:    factory,

		rateLimitEni: rate.NewLimiter(100, 100),
		rateLimitv4:  rate.NewLimiter(100, 100),
		rateLimitv6:  rate.NewLimiter(100, 100),
	}

	return l
}

func TestLocal_Release_ValidIPv4(t *testing.T) {
	local := NewLocalTest(&daemon.ENI{ID: "eni-1"}, nil, &types.PoolConfig{})
	request := &LocalIPResource{
		ENI: daemon.ENI{ID: "eni-1"},
		IP:  types.IPSet2{IPv4: netip.MustParseAddr("192.0.2.1")},
	}
	cni := &daemon.CNI{PodID: "pod-1"}

	local.ipv4.Add(NewValidIP(netip.MustParseAddr("192.0.2.1"), false))
	local.ipv4[netip.MustParseAddr("192.0.2.1")].Allocate("pod-1")

	assert.True(t, local.Release(context.Background(), cni, request))
}

func TestLocal_Release_ValidIPv6(t *testing.T) {
	local := NewLocalTest(&daemon.ENI{ID: "eni-1"}, nil, &types.PoolConfig{})
	request := &LocalIPResource{
		ENI: daemon.ENI{ID: "eni-1"},
		IP:  types.IPSet2{IPv6: netip.MustParseAddr("fd00:46dd:e::1")},
	}
	cni := &daemon.CNI{PodID: "pod-1"}

	local.ipv6.Add(NewValidIP(netip.MustParseAddr("fd00:46dd:e::1"), false))
	local.ipv6[netip.MustParseAddr("fd00:46dd:e::1")].Allocate("pod-1")

	assert.True(t, local.Release(context.Background(), cni, request))
}

func TestLocal_Release_NilENI(t *testing.T) {
	local := NewLocalTest(nil, nil, &types.PoolConfig{})
	request := &LocalIPResource{
		ENI: daemon.ENI{ID: "eni-1"},
		IP:  types.IPSet2{IPv4: netip.MustParseAddr("192.0.2.1")},
	}
	cni := &daemon.CNI{PodID: "pod-1"}

	assert.False(t, local.Release(context.Background(), cni, request))
}

func TestLocal_Release_DifferentENIID(t *testing.T) {
	local := NewLocalTest(&daemon.ENI{ID: "eni-1"}, nil, &types.PoolConfig{})
	request := &LocalIPResource{
		ENI: daemon.ENI{ID: "eni-2"},
		IP:  types.IPSet2{IPv4: netip.MustParseAddr("192.0.2.1")},
	}
	cni := &daemon.CNI{PodID: "pod-1"}

	assert.False(t, local.Release(context.Background(), cni, request))
}

func TestLocal_Release_ValidIPv4_ReleaseIPv6(t *testing.T) {
	local := NewLocalTest(&daemon.ENI{ID: "eni-1"}, nil, &types.PoolConfig{})
	request := &LocalIPResource{
		ENI: daemon.ENI{ID: "eni-1"},
		IP:  types.IPSet2{IPv4: netip.MustParseAddr("192.0.2.1"), IPv6: netip.MustParseAddr("fd00:46dd:e::1")},
	}
	cni := &daemon.CNI{PodID: "pod-1"}

	local.ipv4.Add(NewValidIP(netip.MustParseAddr("192.0.2.1"), false))
	local.ipv4[netip.MustParseAddr("192.0.2.1")].Allocate("pod-1")

	local.ipv6.Add(NewValidIP(netip.MustParseAddr("fd00:46dd:e::1"), false))
	local.ipv6[netip.MustParseAddr("fd00:46dd:e::1")].Allocate("pod-1")

	assert.True(t, local.Release(context.Background(), cni, request))

	assert.Equal(t, ipStatusValid, local.ipv4[netip.MustParseAddr("192.0.2.1")].status)
	assert.Equal(t, "", local.ipv4[netip.MustParseAddr("192.0.2.1")].podID)

	assert.Equal(t, ipStatusValid, local.ipv6[netip.MustParseAddr("fd00:46dd:e::1")].status)
	assert.Equal(t, "", local.ipv6[netip.MustParseAddr("fd00:46dd:e::1")].podID)
}

func TestLocal_AllocWorker_EnableIPv4(t *testing.T) {
	local := NewLocalTest(&daemon.ENI{ID: "eni-1"}, nil, &types.PoolConfig{
		EnableIPv4: true,
	})
	cni := &daemon.CNI{PodID: "pod-1"}

	respCh := make(chan *AllocResp)
	go local.allocWorker(context.Background(), cni, nil, respCh, func() {})

	go func() {
		local.cond.L.Lock()
		local.ipv4.Add(NewValidIP(netip.MustParseAddr("192.0.2.1"), false))
		local.cond.Broadcast()
		local.cond.L.Unlock()
	}()

	resp := <-respCh
	assert.Len(t, resp.NetworkConfigs, 1)

	lo := resp.NetworkConfigs[0].(*LocalIPResource)
	assert.Equal(t, "192.0.2.1", lo.IP.IPv4.String())
	assert.False(t, lo.IP.IPv6.IsValid())
	assert.Equal(t, "eni-1", lo.ENI.ID)
}

func TestLocal_AllocWorker_EnableIPv6(t *testing.T) {
	local := NewLocalTest(&daemon.ENI{ID: "eni-1"}, nil, &types.PoolConfig{
		EnableIPv6: true,
	})
	cni := &daemon.CNI{PodID: "pod-1"}

	respCh := make(chan *AllocResp)
	go local.allocWorker(context.Background(), cni, nil, respCh, func() {})

	go func() {
		local.cond.L.Lock()
		local.ipv6.Add(NewValidIP(netip.MustParseAddr("fd00:46dd:e::1"), false))
		local.cond.Broadcast()
		local.cond.L.Unlock()
	}()

	resp := <-respCh
	assert.Len(t, resp.NetworkConfigs, 1)

	lo := resp.NetworkConfigs[0].(*LocalIPResource)
	assert.Equal(t, netip.MustParseAddr("fd00:46dd:e::1"), lo.IP.IPv6)
	assert.False(t, lo.IP.IPv4.IsValid())
	assert.Equal(t, "eni-1", lo.ENI.ID)
}

func TestLocal_AllocWorker_ParentCancelContext(t *testing.T) {
	local := NewLocalTest(&daemon.ENI{ID: "eni-1"}, nil, &types.PoolConfig{
		EnableIPv4: true,
	})
	cni := &daemon.CNI{PodID: "pod-1"}

	ctx, cancel := context.WithCancel(context.Background())
	respCh := make(chan *AllocResp)
	go local.allocWorker(ctx, cni, nil, respCh, func() {})

	cancel()

	_, ok := <-respCh
	assert.False(t, ok)
}

func TestLocal_Dispose(t *testing.T) {
	local := NewLocalTest(&daemon.ENI{ID: "eni-1"}, nil, &types.PoolConfig{})
	local.status = statusInUse
	local.ipv4.Add(NewValidIP(netip.MustParseAddr("192.0.2.1"), false))
	local.ipv4[netip.MustParseAddr("192.0.2.1")].Allocate("pod-1")
	local.ipv6.Add(NewValidIP(netip.MustParseAddr("fd00:46dd:e::1"), false))
	local.ipv6[netip.MustParseAddr("fd00:46dd:e::1")].Allocate("pod-1")

	n := local.Dispose(10)

	assert.Equal(t, 0, n)
	assert.Equal(t, statusInUse, local.status)
	assert.Equal(t, 1, len(local.ipv4))
	assert.Equal(t, 1, len(local.ipv6))
}

func TestLocal_DisposeWholeENI(t *testing.T) {
	local := NewLocalTest(&daemon.ENI{ID: "eni-1"}, nil, &types.PoolConfig{})
	local.status = statusInUse
	local.ipv4.Add(NewValidIP(netip.MustParseAddr("192.0.2.1"), true))
	local.ipv6.Add(NewValidIP(netip.MustParseAddr("fd00:46dd:e::1"), false))

	n := local.Dispose(1)

	assert.Equal(t, 1, n)
	assert.Equal(t, statusDeleting, local.status)
}
