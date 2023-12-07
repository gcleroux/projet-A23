package agent

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	api "github.com/gcleroux/projet-A23/api/v1"
	"github.com/gcleroux/projet-A23/src/config"
	"github.com/gcleroux/projet-A23/src/loadbalance"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestAgent(t *testing.T) {
	var agents []*Agent

	conf, err := config.LoadConfig()
	require.NoError(t, err)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      conf.Certs.ServerCertFile,
		KeyFile:       conf.Certs.ServerKeyFile,
		CAFile:        conf.Certs.CAFile,
		Server:        true,
		ServerAddress: conf.Servers[0].Address,
	})
	require.NoError(t, err)

	peerTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      conf.Certs.UserCertFile,
		KeyFile:       conf.Certs.UserKeyFile,
		CAFile:        conf.Certs.CAFile,
		Server:        false,
		ServerAddress: conf.Servers[0].Address,
	})
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		bindAddr, err := getAvailableAddr()
		fmt.Println(bindAddr)
		require.NoError(t, err)

		rpcPort, err := getAvailablePort()
		require.NoError(t, err)

		dir, err := os.MkdirTemp(os.TempDir(), "agent-test")
		require.NoError(t, err)

		// Subsequent clients join the cluster
		var startJoinAddrs []string
		if i != 0 {
			startJoinAddrs = append(
				startJoinAddrs,
				agents[0].Config.BindAddr,
			)
		}

		agent, err := New(Config{
			Bootstrap:       i == 0,
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
	writeResponse, err := leaderClient.Write(
		context.Background(),
		&api.WriteRequest{
			Record: &api.Record{
				Value: []byte("foo"),
			},
		},
	)
	require.NoError(t, err)
	// wait until replication has finished
	time.Sleep(3 * time.Second)

	readResponse, err := leaderClient.Read(
		context.Background(),
		&api.ReadRequest{
			Offset: writeResponse.Offset,
		},
	)
	require.NoError(t, err)
	require.Equal(t, readResponse.Record.Value, []byte("foo"))

	followerClient := client(t, agents[1], peerTLSConfig)
	readResponse, err = followerClient.Read(
		context.Background(),
		&api.ReadRequest{
			Offset: writeResponse.Offset,
		},
	)
	require.NoError(t, err)
	require.Equal(t, readResponse.Record.Value, []byte("foo"))
}

func client(t *testing.T, agent *Agent, tlsConfig *tls.Config) api.LogClient {
	tlsCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}

	rpcAddr, err := agent.Config.RPCAddr()
	require.NoError(t, err)

	conn, err := grpc.Dial(fmt.Sprintf(
		"%s:///%s",
		loadbalance.Name,
		rpcAddr,
	), opts...)
	require.NoError(t, err)

	client := api.NewLogClient(conn)
	return client
}

func getAvailableAddr() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	return l.Addr().String(), nil
}

func getAvailablePort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
