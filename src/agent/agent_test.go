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
	"github.com/stretchr/testify/require"
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

	var agents []*Agent
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
	// Waiting that all the servers are initialized
	for len(agents[0].membership.Members()) != 3 {
	}

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
	readResponse, err := leaderClient.Read(
		context.Background(),
		&api.ReadRequest{
			Offset: writeResponse.Offset,
		},
	)
	require.NoError(t, err)
	require.Equal(t, readResponse.Record.Value, []byte("foo"))

	// wait until replication has finished
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	replicated := make(chan struct{})
	go func() {
		for {
			a1, _ := agents[1].log.HighestOffset()
			a2, _ := agents[2].log.HighestOffset()

			if a1 != 0 && a2 != 0 {
				close(replicated)
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		t.Error("Replication took too long. Consider increasing the timer.")
	case <-replicated:
		// Logs done replicating
	}

	for i := 1; i < 3; i++ {
		followerClient := client(t, agents[i], peerTLSConfig)
		readResponse, err = followerClient.Read(
			context.Background(),
			&api.ReadRequest{
				Offset: writeResponse.Offset,
			},
		)
		require.NoError(t, err)
		require.Equal(t, readResponse.Record.Value, []byte("foo"))
	}
}

func client(t *testing.T, agent *Agent, tlsConfig *tls.Config) api.LogClient {
	tlsCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}

	rpcAddr, err := agent.Config.RPCAddr()
	require.NoError(t, err)

	conn, err := grpc.Dial(rpcAddr, opts...)
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
