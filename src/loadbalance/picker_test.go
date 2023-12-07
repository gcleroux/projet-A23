package loadbalance

import (
	"testing"

	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"

	"github.com/stretchr/testify/require"
)

func TestPickerNoSubConnAvailable(t *testing.T) {
	picker := &Picker{}
	for _, method := range []string{
		"/log.vX.Log/Write",
		"/log.vX.Log/Read",
	} {
		info := balancer.PickInfo{
			FullMethodName: method,
		}
		result, err := picker.Pick(info)
		require.Equal(t, balancer.ErrNoSubConnAvailable, err)
		require.Nil(t, result.SubConn)
	}
}

func TestPickerWriteToLeader(t *testing.T) {
	picker, subConns := setupTest()
	info := balancer.PickInfo{
		FullMethodName: "/log.vX.Log/Write",
	}
	for i := 0; i < 5; i++ {
		gotPick, err := picker.Pick(info)
		require.NoError(t, err)
		require.Equal(t, subConns[0], gotPick.SubConn)
	}
}

func TestPickerReadFromFollowers(t *testing.T) {
	picker, subConns := setupTest()
	info := balancer.PickInfo{
		FullMethodName: "/log.vX.Log/Read",
	}
	for i := 0; i < 5; i++ {
		pick, err := picker.Pick(info)
		require.NoError(t, err)
		require.Equal(t, subConns[i%2+1], pick.SubConn)
	}
}

func setupTest() (*Picker, []*subConn) {
	var subConns []*subConn
	buildInfo := base.PickerBuildInfo{
		ReadySCs: make(map[balancer.SubConn]base.SubConnInfo),
	}
	for i := 0; i < 3; i++ {
		sc := &subConn{}
		addr := resolver.Address{
			Attributes: attributes.New("is_leader", i == 0),
		}
		// 0th sub conn is the leader
		sc.UpdateAddresses([]resolver.Address{addr})
		buildInfo.ReadySCs[sc] = base.SubConnInfo{Address: addr}
		subConns = append(subConns, sc)
	}
	picker := &Picker{}
	picker.Build(buildInfo)
	return picker, subConns
}

// subConn implements balancer.SubConn.
type subConn struct {
	addrs []resolver.Address
}

func (s *subConn) GetOrBuildProducer(builder balancer.ProducerBuilder) (p balancer.Producer, close func()) {
	// TODO implement me
	panic("implement me")
}

func (s *subConn) UpdateAddresses(addrs []resolver.Address) {
	s.addrs = addrs
}

func (s *subConn) Connect() {}

func (s *subConn) Shutdown() {}
