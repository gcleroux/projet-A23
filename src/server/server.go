package server

import (
	"context"
	"time"

	api "github.com/gcleroux/projet-A23/api/v1"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/hashicorp/raft"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	api.UnimplementedLogServer
	*Config
}

type CommitLog interface {
	Append(*api.Record) (uint64, error)
	Read(uint64) (*api.Record, error)
	GetLeader() (raft.ServerAddress, raft.ServerID)
}

type Authorizer interface {
	Authorize(subject, object, action string) error
}

type ServerGetter interface {
	GetServers(*api.GetServersRequest) ([]*api.Server, error)
}

type Config struct {
	CommitLog    CommitLog
	Authorizer   Authorizer
	ServerGetter ServerGetter
	NodeName     string
	ServerAddr   string
	creds        credentials.TransportCredentials
}

const (
	objectWildcard = "*"
	writeAction    = "write"
	readAction     = "read"
)

var _ api.LogServer = (*grpcServer)(nil)

func newgrpcServer(config *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		Config: config,
	}
	return srv, nil
}

func NewGRPCServer(config *Config, creds credentials.TransportCredentials, grpcOpts ...grpc.ServerOption) (*grpc.Server, error) {
	logger := zap.L().Named(config.NodeName)
	zapOpts := []grpc_zap.Option{
		grpc_zap.WithDurationField(
			func(duration time.Duration) zapcore.Field {
				return zap.Int64(
					"grpc.time_ns",
					duration.Nanoseconds(),
				)
			},
		),
	}
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	err := view.Register(ocgrpc.DefaultServerViews...)
	if err != nil {
		return nil, err
	}

	grpcOpts = append(grpcOpts,
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				grpc_ctxtags.StreamServerInterceptor(),
				grpc_zap.StreamServerInterceptor(logger, zapOpts...),
				grpc_auth.StreamServerInterceptor(authenticate),
			)), grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(logger, zapOpts...),
			grpc_auth.UnaryServerInterceptor(authenticate),
		)),
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
	)
	gsrv := grpc.NewServer(grpcOpts...)
	srv, err := newgrpcServer(config)
	srv.creds = creds
	if err != nil {
		return nil, err
	}
	api.RegisterLogServer(gsrv, srv)
	return gsrv, nil
}

func (s *grpcServer) Write(ctx context.Context, req *api.WriteRequest) (*api.WriteResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		writeAction,
	); err != nil {
		return nil, err
	}
	record := req.Record
	leader, _ := s.CommitLog.GetLeader()
	if string(leader) != s.ServerAddr {
		record.Server = s.NodeName
		return s.forwardToLeader(ctx, record, string(leader))
	}
	if record.Server == "" {
		record.Server = s.NodeName
	}
	offset, err := s.CommitLog.Append(record)
	if err != nil {
		return nil, err
	}
	return &api.WriteResponse{Offset: offset}, nil
}

func (s *grpcServer) Read(ctx context.Context, req *api.ReadRequest) (*api.ReadResponse, error) {
	if err := s.Authorizer.Authorize(
		subject(ctx),
		objectWildcard,
		readAction,
	); err != nil {
		return nil, err
	}
	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}
	return &api.ReadResponse{Record: record}, nil
}

func (s *grpcServer) WriteStream(stream api.Log_WriteStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		res, err := s.Write(stream.Context(), req)
		if err != nil {
			return err
		}

		if err = stream.Send(res); err != nil {
			return err
		}
	}
}

func (s *grpcServer) ReadStream(req *api.ReadRequest, stream api.Log_ReadStreamServer) error {
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			res, err := s.Read(stream.Context(), req)
			switch err.(type) {
			case nil:
			case api.ErrOffsetOutOfRange:
				continue
			default:
				return err
			}
			if res.GetRecord().Server == s.NodeName {
				if err = stream.Send(res); err != nil {
					return err
				}
			}
			req.Offset++
		}
	}
}

func (s *grpcServer) GetServers(ctx context.Context, req *api.GetServersRequest) (*api.GetServersResponse, error) {
	servers, err := s.ServerGetter.GetServers(req)
	if err != nil {
		return nil, err
	}
	return &api.GetServersResponse{Servers: servers}, nil
}

func authenticate(ctx context.Context) (context.Context, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(
			codes.Unknown,
			"couldn't find peer info",
		).Err()
	}

	if peer.AuthInfo == nil {
		return context.WithValue(ctx, subjectContextKey{}, ""), nil
	}

	tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)

	return ctx, nil
}

func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

func (s *grpcServer) forwardToLeader(ctx context.Context, r *api.Record, addr string) (*api.WriteResponse, error) {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(s.creds)}
	leaderConn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	defer leaderConn.Close()
	client := api.NewLogClient(leaderConn)

	return client.Write(ctx, &api.WriteRequest{Record: r})
}

type subjectContextKey struct{}
