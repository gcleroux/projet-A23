package agent_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"testing"
	"time"

	api "github.com/gcleroux/projet-A23/api/v1"
	"github.com/gcleroux/projet-A23/src/agent"
	"github.com/gcleroux/projet-A23/src/config"
	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestAgent(t *testing.T) {
	conf, err := config.LoadConfig()
	require.NoError(t, err)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      conf.Certs.ServerCertFile,
		KeyFile:       conf.Certs.ServerKeyFile,
		CAFile:        conf.Certs.CAFile,
		Server:        true,
		ServerAddress: conf.Server.Address,
	})
	require.NoError(t, err)

	peerTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      conf.Certs.UserCertFile,
		KeyFile:       conf.Certs.UserKeyFile,
		CAFile:        conf.Certs.CAFile,
		Server:        false,
		ServerAddress: conf.Server.Address,
	})
	require.NoError(t, err)

	var agents []*agent.Agent
	for i := 0; i < 3; i++ {
		ports := dynaport.Get(2)
		bindAddr := fmt.Sprintf("%s:%d", "127.0.0.1", ports[0])
		rpcPort := ports[1]

		dir, err := os.MkdirTemp(os.TempDir(), "agent-test")
		require.NoError(t, err)

		var startJoinAddrs []string
		if i != 0 {
			startJoinAddrs = append(
				startJoinAddrs,
				agents[0].Config.BindAddr,
			)
		}

		agent, err := agent.New(agent.Config{
			NodeName:        fmt.Sprintf("%d", i),
			StartJoinAddrs:  startJoinAddrs,
			BindAddr:        bindAddr,
			RPCPort:         rpcPort,
			DataDir:         dir,
			ACLModelFile:    conf.Certs.ACLModelFile,
			ACLPolicyFile:   conf.Certs.ACLPolicyFile,
			ServerTLSConfig: serverTLSConfig,
			PeerTLSConfig:   peerTLSConfig,
		})
		require.NoError(t, err)

		agents = append(agents, agent)
	}
	defer func() {
		for _, agent := range agents {
			err := agent.Shutdown()
			require.NoError(t, err)
			require.NoError(t,
				os.RemoveAll(agent.Config.DataDir),
			)
		}
	}()
	time.Sleep(3 * time.Second)

	leaderClient := client(t, agents[0], peerTLSConfig)
	produceResponse, err := leaderClient.Write(
		context.Background(),
		&api.WriteRequest{
			Record: &api.Record{
				Value: []byte("foo"),
			},
		},
	)
	require.NoError(t, err)
	consumeResponse, err := leaderClient.Read(
		context.Background(),
		&api.ReadRequest{
			Offset: produceResponse.Offset,
		},
	)
	require.NoError(t, err)
	require.Equal(t, consumeResponse.Record.Value, []byte("foo"))

	// wait until replication has finished
	time.Sleep(3 * time.Second)

	followerClient := client(t, agents[1], peerTLSConfig)
	consumeResponse, err = followerClient.Read(
		context.Background(),
		&api.ReadRequest{
			Offset: produceResponse.Offset,
		},
	)
	require.NoError(t, err)
	require.Equal(t, consumeResponse.Record.Value, []byte("foo"))
}

func client(
	t *testing.T,
	agent *agent.Agent,
	tlsConfig *tls.Config,
) api.LogClient {
	tlsCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
	rpcAddr, err := agent.Config.RPCAddr()
	require.NoError(t, err)
	conn, err := grpc.Dial(fmt.Sprintf(
		"%s",
		rpcAddr,
	), opts...)
	require.NoError(t, err)
	client := api.NewLogClient(conn)
	return client
}
