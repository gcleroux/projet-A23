package server

import (
	"context"
	"net"
	"os"
	"testing"

	api "github.com/gcleroux/projet-ift605/api/v1"
	"github.com/gcleroux/projet-ift605/src/log"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestServer(t *testing.T) {
	scenarios := make(map[string]func(t *testing.T, client api.LogClient, config *Config))

	scenarios["Write/Read a message to/from the log succeeds"] = testWriteRead
	scenarios["Write/Read stream succeeds"] = testWriteReadStream
	scenarios["Read past log boundary fails"] = testReadPastBoundary

	for scenario, fn := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			client, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, client, config)
		})
	}
}

func setupTest(t *testing.T, fn func(*Config)) (client api.LogClient, cfg *Config, teardown func()) {
	t.Helper()

	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	clientOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	cc, err := grpc.Dial(l.Addr().String(), clientOptions...)
	require.NoError(t, err)

	dir, err := os.MkdirTemp(os.TempDir(), "server-test")
	require.NoError(t, err)

	clog, err := log.NewLog(dir, log.Config{
		Segment: struct {
			MaxStoreBytes uint64
			MaxIndexBytes uint64
			InitialOffset uint64
		}{
			MaxStoreBytes: 1024,
			MaxIndexBytes: 1024,
			InitialOffset: 0,
		},
	})
	require.NoError(t, err)

	cfg = &Config{
		CommitLog: clog,
	}
	if fn != nil {
		fn(cfg)
	}
	server, err := NewGRPCServer(cfg)
	require.NoError(t, err)

	go func() {
		err := server.Serve(l)
		require.NoError(t, err)
	}()

	client = api.NewLogClient(cc)

	return client, cfg, func() {
		server.Stop()
		cc.Close()
		l.Close()
		err := clog.Remove()
		require.NoError(t, err)
	}
}

func testWriteRead(t *testing.T, client api.LogClient, config *Config) {
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

func testReadPastBoundary(t *testing.T, client api.LogClient, config *Config) {
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

func testWriteReadStream(t *testing.T, client api.LogClient, config *Config) {
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