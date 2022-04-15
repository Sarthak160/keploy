package grpcserver

// this will be the server file for the grpc connection

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.keploy.io/server/graph"
	proto "go.keploy.io/server/grpc/regression"
	"go.keploy.io/server/pkg/models"
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

func (srv *Server) Start(ctx context.Context, request *proto.StartRequest) (*proto.StartResponse, error) {
	t := request.Total
	total, err := strconv.Atoi(t)
	if err != nil {
		return nil, err
	}
	app := request.App
	if app == "" {
		return nil, errors.New("app is required in request")
	}
	id := uuid.New().String()
	now := time.Now().Unix()
	err = srv.run.Put(ctx, run.TestRun{
		ID:      id,
		Created: now,
		Updated: now,
		Status:  run.TestRunStatusRunning,
		CID:     graph.DEFAULT_COMPANY,
		App:     app,
		User:    graph.DEFAULT_USER,
		Total:   total,
	})
	if err != nil {
		return nil, err
	}
	//check
	return &proto.StartResponse{Id: id}, nil
}

func helper(m map[string][]string) (res map[string]*proto.StrArr) {
	for k, v := range m {
		arr := &proto.StrArr{}
		arr.Value = append(arr.Value, v...)
		res[k] = arr
	}
	return
}
func toProtoTC(tcs models.TestCase) (*proto.TestCase, error) {
	reqHeader := helper(map[string][]string(tcs.HttpReq.Header))
	respHeader := helper(map[string][]string(tcs.HttpResp.Header))
	deps := []*proto.Dependency{}
	allKeys := helper(map[string][]string(tcs.AllKeys))
	anchors := helper(map[string][]string(tcs.Anchors))
	for _, j := range tcs.Deps {
		data := []*proto.DataBytes{}
		for _, k := range j.Data {
			data = append(data, &proto.DataBytes{
				Bin: k,
			})
		}
		deps = append(deps, &proto.Dependency{
			Name: j.Name,
			Type: string(j.Type),
			Meta: j.Meta,
			Data: data,
		})
	}
	ptcs := &proto.TestCase{
		Id:       tcs.ID,
		Created:  tcs.Created,
		Updated:  tcs.Updated,
		Captured: tcs.Captured,
		CID:      tcs.CID,
		AppID:    tcs.AppID,
		URI:      tcs.URI,
		HttpReq: &proto.HttpReq{
			Method:     string(tcs.HttpReq.Method),
			ProtoMajor: int64(tcs.HttpReq.ProtoMajor),
			ProtoMinor: int64(tcs.HttpReq.ProtoMinor),
			URL:        tcs.HttpReq.URL,
			URLParams:  tcs.HttpReq.URLParams,
			Header:     reqHeader,
			Body:       tcs.HttpReq.Body,
		},
		HttpResp: &proto.HttpResp{
			StatusCode: int64(tcs.HttpResp.StatusCode),
			Header:     respHeader,
			Body:       tcs.HttpResp.Body,
		},
		Deps:    deps,
		AllKeys: allKeys,
		Anchors: anchors,
		Noise:   tcs.Noise,
	}
	return ptcs, nil
}

func (srv *Server) GetTC(ctx context.Context, request *proto.GetTCRequest) (*proto.TestCase, error) {
	id := request.Id
	app := request.App
	tcs, err := srv.svc.Get(ctx, graph.DEFAULT_COMPANY, app, id)
	if err != nil {
		return nil, err
	}
	// print(tcs)
	tcs, err = srv.svc.Get(ctx, graph.DEFAULT_COMPANY, app, id)
	if err != nil {
		return nil, err
	}
	ptcs,err:= toProtoTC(tcs)
	if err != nil {
		return nil, err
	}
	return ptcs,nil
	// reqHeader := helper(map[string][]string(tcs.HttpReq.Header))
	// respHeader := helper(map[string][]string(tcs.HttpResp.Header))
	// deps := []*proto.Dependency{}
	// allKeys := helper(map[string][]string(tcs.AllKeys))
	// anchors := helper(map[string][]string(tcs.Anchors))
	// for _, j := range tcs.Deps {
	// 	data := []*proto.DataBytes{}
	// 	for _, k := range j.Data {
	// 		data = append(data, &proto.DataBytes{
	// 			Bin: k,
	// 		})
	// 	}
	// 	deps = append(deps, &proto.Dependency{
	// 		Name: j.Name,
	// 		Type: string(j.Type),
	// 		Meta: j.Meta,
	// 		Data: data,
	// 	})
	// }
	// return &proto.TestCase{
	// 	Id:       tcs.ID,
	// 	Created:  tcs.Created,
	// 	Updated:  tcs.Updated,
	// 	Captured: tcs.Captured,
	// 	CID:      tcs.CID,
	// 	AppID:    tcs.AppID,
	// 	URI:      tcs.URI,
	// 	HttpReq: &proto.HttpReq{
	// 		Method:     string(tcs.HttpReq.Method),
	// 		ProtoMajor: int64(tcs.HttpReq.ProtoMajor),
	// 		ProtoMinor: int64(tcs.HttpReq.ProtoMinor),
	// 		URL:        tcs.HttpReq.URL,
	// 		URLParams:  tcs.HttpReq.URLParams,
	// 		Header:     reqHeader,
	// 		Body:       tcs.HttpReq.Body,
	// 	},
	// 	HttpResp: &proto.HttpResp{
	// 		StatusCode: int64(tcs.HttpResp.StatusCode),
	// 		Header:     respHeader,
	// 		Body:       tcs.HttpResp.Body,
	// 	},
	// 	Deps:    deps,
	// 	AllKeys: allKeys,
	// 	Anchors: anchors,
	// 	Noise:   tcs.Noise,
	// }, nil

}

func (srv *Server) GetTCS(ctx context.Context, request *proto.GetTCSRequest) (*proto.GetTCSResponse, error) {
	app := request.App
	if app == "" {
		return nil, errors.New("app is required in request")
	}
	offsetStr := request.Offset
	limitStr := request.Limit
	var (
		offset int
		limit  int
		err    error
	)
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			srv.logger.Error("request for fetching testcases in converting offset to integer")
		}
	}
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			srv.logger.Error("request for fetching testcases in converting limit to integer")
		}
	}
	tcs, err := srv.svc.GetAll(ctx, graph.DEFAULT_COMPANY, app, &offset, &limit)
	if err != nil {
		return nil, err
	}
	var ptcs []*proto.TestCase
	for i := 0; i < len(tcs); i++ {
		ptc,err:= toProtoTC(tcs[i])
		if err != nil {
			return nil, err
		}
		ptcs = append(ptcs,ptc)
	}

	return &proto.GetTCSResponse{Tcs: ptcs}, nil
}

func (srv *Server) PostTC(ctx context.Context, request *proto.TestCaseReq) (*proto.PostTCResponse, error) {
	data := &proto.TestCaseReq{
		Captured:   request.Captured,
		AppID:      request.AppID,
		URI:        request.URI,
		HttpReq:    request.HttpReq,
		HttpResp:   request.HttpResp,
		Dependency: request.Dependency,
	}
	now := time.Now().UTC().Unix()
	var ptcs []*proto.TestCase
	// for i := 0; i < len(tcs); i++ {
	// 	ptc,err:= toProtoTC(tcs[i])
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	ptcs = append(ptcs,ptc)
	// }
	inserted, err := srv.svc.Put(ctx, graph.DEFAULT_COMPANY,[]*proto.TestCase )

}
