package grpcserver

// this will be the server file for the grpc connection

import (
	"context"
	"net"
	"time"

	proto "go.keploy.io/server/grpc/regression"
	regression2 "go.keploy.io/server/pkg/service/regression"
	"go.keploy.io/server/pkg/service/run"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	logger *zap.Logger
	svc    regression2.Service
	run    run.Service
	proto.UnimplementedEndServiceServer
}

func New(logger *zap.Logger, svc regression2.Service, run run.Service) {
	listener, err := net.Listen("tcp", ":4040")
	if err != nil {
		panic(err)
	}

	srv := grpc.NewServer()
	proto.RegisterEndServiceServer(srv, &Server{logger: logger, svc: svc, run: run})
	reflection.Register(srv)

	if e := srv.Serve(listener); e != nil {
		panic(e)
	}

}

func (srv *Server) End(ctx context.Context, request *proto.EndRequest) (*proto.EndResponse, error) {
	stat := run.TestRunStatusFailed
	id := request.Id
	if request.Status == "true" {
		stat = run.TestRunStatusPassed
	}
	now := time.Now().Unix()

	err := srv.run.Put(ctx, run.TestRun{
		ID:      id,
		Updated: now,
		Status:  stat,
	})
	if err != nil {
		return &proto.EndResponse{Message: err.Error()}, nil
	}
	return &proto.EndResponse{Message: "OK"}, nil
}
