package server

import (
	"context"
	"flag"
	"net"
	"os"
	"testing"
	"time"

	api "github.com/gcleroux/projet-A23/api/v1"
	"github.com/gcleroux/projet-A23/src/auth"
	"github.com/gcleroux/projet-A23/src/config"
	"github.com/gcleroux/projet-A23/src/log"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"go.uber.org/zap"

	"go.opencensus.io/examples/exporter"
)

var debug = flag.Bool("debug", false, "Enable observability for debugging.")

type MockDLog struct {
	*log.Log
}

func (m *MockDLog) GetLeader() (raft.ServerAddress, raft.ServerID) {
	return "", ""
}

func TestMain(m *testing.M) {
	flag.Parse()
	if *debug {
		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		zap.ReplaceGlobals(logger)
	}
	os.Exit(m.Run())
}

func TestServer(t *testing.T) {
	scenarios := make(map[string]func(t *testing.T, userClient api.LogClient, nobodyClient api.LogClient, config *Config))

	scenarios["Write/Read a message to/from the log succeeds"] = testWriteRead
	scenarios["Write/Read stream succeeds"] = testWriteReadStream
	scenarios["Read past log boundary fails"] = testReadPastBoundary
	scenarios["Unauthorized user fails"] = testUnauthorized

	for scenario, fn := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			userClient,
				nobodyClient,
				config,
				teardown := setupTest(t, nil)
			defer teardown()
			fn(t, userClient, nobodyClient, config)
		})
	}
}

func setupTest(t *testing.T, fn func(*Config)) (userClient api.LogClient, nobodyClient api.LogClient, cfg *Config, teardown func()) {
	t.Helper()

	// Setup config
	conf, err := config.LoadConfig()
	require.NoError(t, err)

	l, err := net.Listen("tcp", conf.Servers[0].Address)
	require.NoError(t, err)

	newClient := func(crtPath, keyPath string) (
		*grpc.ClientConn,
		api.LogClient,
		[]grpc.DialOption,
	) {
		tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
			CertFile: crtPath,
			KeyFile:  keyPath,
			CAFile:   conf.Certs.CAFile,
			Server:   false,
		})
		require.NoError(t, err)
		tlsCreds := credentials.NewTLS(tlsConfig)
		opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
		conn, err := grpc.Dial(l.Addr().String(), opts...)
		require.NoError(t, err)
		client := api.NewLogClient(conn)
		return conn, client, opts
	}

	var userConn *grpc.ClientConn
	userConn, userClient, _ = newClient(
		conf.Certs.UserCertFile,
		conf.Certs.UserKeyFile,
	)

	var nobodyConn *grpc.ClientConn
	nobodyConn, nobodyClient, _ = newClient(
		conf.Certs.NobodyCertFile,
		conf.Certs.NobodyKeyFile,
	)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      conf.Certs.ServerCertFile,
		KeyFile:       conf.Certs.ServerKeyFile,
		CAFile:        conf.Certs.CAFile,
		ServerAddress: l.Addr().String(),
		Server:        true,
	})
	require.NoError(t, err)

	serverCreds := credentials.NewTLS(serverTLSConfig)

	dir, err := os.MkdirTemp(os.TempDir(), "server-test")
	require.NoError(t, err)

	clog, err := log.NewLog(dir, &log.Config{})
	require.NoError(t, err)

	mockLog := &MockDLog{Log: clog}

	authorizer := auth.New(conf.Certs.ACLModelFile, conf.Certs.ACLPolicyFile)

	var telemetryExporter *exporter.LogExporter
	if *debug {
		metricsLogFile, err := os.CreateTemp("", "metrics-*.log")
		require.NoError(t, err)
		t.Logf("metrics log file: %s", metricsLogFile.Name())

		tracesLogFile, err := os.CreateTemp("", "traces-*.log")
		require.NoError(t, err)
		t.Logf("traces log file: %s", tracesLogFile.Name())

		telemetryExporter, err = exporter.NewLogExporter(exporter.Options{
			MetricsLogFile:    metricsLogFile.Name(),
			TracesLogFile:     tracesLogFile.Name(),
			ReportingInterval: time.Second,
		})
		require.NoError(t, err)
		err = telemetryExporter.Start()
		require.NoError(t, err)
	}

	cfg = &Config{
		CommitLog:  mockLog,
		Authorizer: authorizer,
	}
	if fn != nil {
		fn(cfg)
	}
	server, err := NewGRPCServer(cfg, nil, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go func() {
		err := server.Serve(l)
		require.NoError(t, err)
	}()

	return userClient, nobodyClient, cfg, func() {
		server.Stop()
		userConn.Close()
		nobodyConn.Close()
		l.Close()
		if telemetryExporter != nil {
			time.Sleep(1500 * time.Millisecond)
			telemetryExporter.Stop()
			telemetryExporter.Close()
		}
	}
}

func testWriteRead(t *testing.T, client api.LogClient, _ api.LogClient, config *Config) {
	ctx := context.Background()

	want := &api.Record{
		Value: []byte("hello world"),
	}

	Write, err := client.Write(ctx, &api.WriteRequest{Record: want})
	require.NoError(t, err)

	Read, err := client.Read(ctx, &api.ReadRequest{Offset: Write.Offset})
	require.NoError(t, err)
	require.Equal(t, want.Value, Read.Record.Value)
	require.Equal(t, want.Offset, Read.Record.Offset)
}

func testReadPastBoundary(t *testing.T, client api.LogClient, _ api.LogClient, config *Config) {
	ctx := context.Background()

	Write, err := client.Write(ctx, &api.WriteRequest{Record: &api.Record{Value: []byte("hello world")}})
	require.NoError(t, err)

	Read, err := client.Read(ctx, &api.ReadRequest{Offset: Write.Offset + 1})
	if Read != nil {
		t.Fatal("Read not nil")
	}

	got := status.Code(err)
	want := status.Code(api.ErrOffsetOutOfRange{}.GRPCStatus().Err())
	if got != want {
		t.Fatalf("got err: %v, want: %v", got, want)
	}
}

func testWriteReadStream(t *testing.T, client api.LogClient, _ api.LogClient, config *Config) {
	ctx := context.Background()

	records := []*api.Record{}
	records = append(records, &api.Record{Value: []byte("first message"), Offset: 0})
	records = append(records, &api.Record{Value: []byte("second message"), Offset: 1})

	writeStream, err := client.WriteStream(ctx)
	require.NoError(t, err)

	for offset, record := range records {
		err = writeStream.Send(&api.WriteRequest{Record: record})
		require.NoError(t, err)

		res, err := writeStream.Recv()
		require.NoError(t, err)
		if res.Offset != uint64(offset) {
			t.Fatalf("got offset: %d, want: %d", res.Offset, offset)
		}
	}

	readStream, err := client.ReadStream(ctx, &api.ReadRequest{Offset: 0})
	require.NoError(t, err)
	for i, record := range records {
		res, err := readStream.Recv()
		require.NoError(t, err)
		require.Equal(t, res.Record, &api.Record{Value: record.Value, Offset: uint64(i)})
	}
}

func testUnauthorized(t *testing.T, _, client api.LogClient, config *Config) {
	ctx := context.Background()
	write, err := client.Write(ctx,
		&api.WriteRequest{
			Record: &api.Record{
				Value: []byte("hello world"),
			},
		},
	)
	if write != nil {
		t.Fatalf("Write response should be nil")
	}
	gotCode, wantCode := status.Code(err), codes.PermissionDenied
	if gotCode != wantCode {
		t.Fatalf("got code: %d, want: %d", gotCode, wantCode)
	}
	read, err := client.Read(ctx, &api.ReadRequest{
		Offset: 0,
	})
	if read != nil {
		t.Fatalf("Read response should be nil")
	}
	gotCode, wantCode = status.Code(err), codes.PermissionDenied
	if gotCode != wantCode {
		t.Fatalf("got code: %d, want: %d", gotCode, wantCode)
	}
}
