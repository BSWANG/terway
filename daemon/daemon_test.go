package daemon

import (
	"net/netip"
	"testing"

	"github.com/AliyunContainerService/terway/pkg/aliyun/client"
	factorymocks "github.com/AliyunContainerService/terway/pkg/factory/mocks"
	k8smocks "github.com/AliyunContainerService/terway/pkg/k8s/mocks"
	"github.com/AliyunContainerService/terway/pkg/utils/nodecap"
	"github.com/AliyunContainerService/terway/types"
	"github.com/AliyunContainerService/terway/types/daemon"

	"github.com/stretchr/testify/assert"
)

func Test_checkInstance1(t *testing.T) {
	nodecap.SetNodeCapabilities("erdma", "true")

	type args struct {
		limit      *client.Limits
		daemonMode string
		config     *daemon.Config
	}
	tests := []struct {
		name     string
		args     args
		v4       bool
		v6       bool
		trunking bool
		erdma    bool
	}{
		{
			name: "unsupported instance",
			args: args{
				limit:      &client.Limits{},
				daemonMode: "ENIMultiIP",
				config: &daemon.Config{
					IPStack:           "dual",
					EnableENITrunking: true,
					EnableERDMA:       true,
				},
			},
			v4:       true,
			v6:       false,
			trunking: false,
			erdma:    false,
		},
		{
			name: "supported instance",
			args: args{
				limit: &client.Limits{
					Adapters:              10,
					TotalAdapters:         15,
					IPv4PerAdapter:        10,
					IPv6PerAdapter:        10,
					MemberAdapterLimit:    10,
					MaxMemberAdapterLimit: 10,
					ERdmaAdapters:         2,
				},
				daemonMode: "ENIMultiIP",
				config: &daemon.Config{
					IPStack:           "dual",
					EnableENITrunking: true,
					EnableERDMA:       true,
				},
			},
			v4:       true,
			v6:       true,
			trunking: true,
			erdma:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := checkInstance(tt.args.limit, tt.args.daemonMode, tt.args.config)
			assert.Equalf(t, tt.v4, got, "v4(%v, %v, %v)", tt.args.limit, tt.args.daemonMode, tt.args.config)
			assert.Equalf(t, tt.v6, got1, "v6(%v, %v, %v)", tt.args.limit, tt.args.daemonMode, tt.args.config)
			assert.Equalf(t, tt.trunking, tt.args.config.EnableENITrunking, "trunking(%v, %v, %v)", tt.args.limit, tt.args.daemonMode, tt.args.config)
			assert.Equalf(t, tt.erdma, tt.args.config.EnableERDMA, "erdma(%v, %v, %v)", tt.args.limit, tt.args.daemonMode, tt.args.config)
		})
	}
}

func Test_initTrunk(t *testing.T) {
	type args struct {
		config     *daemon.Config
		poolConfig *types.PoolConfig
		k8sClient  *k8smocks.Kubernetes
		f          *factorymocks.Factory
	}
	tests := []struct {
		name     string
		args     args
		preStart func(args)
		want     string
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name: "empty trunk id should create new trunk",
			args: args{
				config: &daemon.Config{
					IPStack:           "dual",
					EnableENITrunking: true,
					EnableERDMA:       true,
				},
				poolConfig: &types.PoolConfig{
					MaxENI: 2,
				},
				k8sClient: k8smocks.NewKubernetes(t),
				f:         factorymocks.NewFactory(t),
			},
			preStart: func(args args) {
				args.k8sClient.On("GetTrunkID").Return("")
				args.f.On("CreateNetworkInterface", 1, 0, "trunk").Return(&daemon.ENI{
					ID:               "eni-1",
					MAC:              "",
					SecurityGroupIDs: nil,
					Trunk:            true,
					ERdma:            false,
					PrimaryIP:        types.IPSet{},
					GatewayIP:        types.IPSet{},
					VSwitchCIDR:      types.IPNetSet{},
					VSwitchID:        "",
				}, []netip.Addr{}, []netip.Addr{}, nil)
				args.f.On("GetAttachedNetworkInterface", "").Return([]*daemon.ENI{
					{
						ID:               "eni-1",
						MAC:              "",
						SecurityGroupIDs: nil,
						Trunk:            false,
						ERdma:            false,
						PrimaryIP:        types.IPSet{},
						GatewayIP:        types.IPSet{},
						VSwitchCIDR:      types.IPNetSet{},
						VSwitchID:        "",
					},
				}, nil)
			},
			want:    "eni-1",
			wantErr: assert.NoError,
		}, {
			name: "reuse exist trunk eni",
			args: args{
				config: &daemon.Config{
					IPStack:           "dual",
					EnableENITrunking: true,
					EnableERDMA:       true,
				},
				poolConfig: &types.PoolConfig{
					MaxENI: 2,
				},
				k8sClient: k8smocks.NewKubernetes(t),
				f:         factorymocks.NewFactory(t),
			},
			preStart: func(args args) {
				args.k8sClient.On("GetTrunkID").Return("")
				args.f.On("GetAttachedNetworkInterface", "").Return([]*daemon.ENI{
					{
						ID:               "eni-1",
						MAC:              "",
						SecurityGroupIDs: nil,
						Trunk:            true,
						ERdma:            false,
						PrimaryIP:        types.IPSet{},
						GatewayIP:        types.IPSet{},
						VSwitchCIDR:      types.IPNetSet{},
						VSwitchID:        "",
					},
				}, nil)
			},
			want:    "eni-1",
			wantErr: assert.NoError,
		}, {
			name: "disable trunk if can not create more",
			args: args{
				config: &daemon.Config{
					IPStack:           "dual",
					EnableENITrunking: true,
					EnableERDMA:       true,
				},
				poolConfig: &types.PoolConfig{
					MaxENI: 2,
				},
				k8sClient: k8smocks.NewKubernetes(t),
				f:         factorymocks.NewFactory(t),
			},
			preStart: func(args args) {
				args.k8sClient.On("GetTrunkID").Return("")
				args.f.On("GetAttachedNetworkInterface", "").Return([]*daemon.ENI{
					{
						ID: "eni-1",
					},
					{
						ID: "eni-2",
					},
				}, nil)
			},
			want:    "",
			wantErr: assert.NoError,
		}, {
			name: "enable wait trunk ready",
			args: args{
				config: &daemon.Config{
					IPStack:           "dual",
					EnableENITrunking: true,
					EnableERDMA:       true,
					WaitTrunkENI:      true,
				},
				poolConfig: &types.PoolConfig{
					MaxENI: 2,
				},
				k8sClient: k8smocks.NewKubernetes(t),
				f:         factorymocks.NewFactory(t),
			},
			preStart: func(args args) {
				args.k8sClient.On("GetTrunkID").Return("eni-1")
			},
			want:    "eni-1",
			wantErr: assert.NoError,
		}, {
			name: "enable wait trunk ready, get from remote",
			args: args{
				config: &daemon.Config{
					IPStack:           "dual",
					EnableENITrunking: true,
					EnableERDMA:       true,
					WaitTrunkENI:      true,
				},
				poolConfig: &types.PoolConfig{
					MaxENI: 2,
				},
				k8sClient: k8smocks.NewKubernetes(t),
				f:         factorymocks.NewFactory(t),
			},
			preStart: func(args args) {
				args.k8sClient.On("GetTrunkID").Return("")
				args.k8sClient.On("WaitTrunkReady").Return("eni-1", nil)
			},
			want:    "eni-1",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.preStart(tt.args)

			got, err := initTrunk(tt.args.config, tt.args.poolConfig, tt.args.k8sClient, tt.args.f)
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.want, got)
			if got == "" {
				assert.False(t, tt.args.config.EnableENITrunking)
			}
		})
	}
}
