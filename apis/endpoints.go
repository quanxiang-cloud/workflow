package apis

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

// Endpoints collects all of the endpoints that compose a workflow service.
type Endpoints struct {
	PostSavePipelineEndpoint    endpoint.Endpoint
	PostExecPipelineEndpoint    endpoint.Endpoint
	PostExecpipelineRunEndpoint endpoint.Endpoint
}

// NewServerEndpoints returns an Endpoints struct where each endpoint invokes
// the corresponding method on the provided service. Useful in a all-svc
// server.
func NewServerEndpoints(s Service) Endpoints {
	return Endpoints{
		PostSavePipelineEndpoint:    PostSavePipelineEndpoints(s.GetPipeline()),
		PostExecPipelineEndpoint:    PostExecPipelineEndpoints(s.GetPipeline()),
		PostExecpipelineRunEndpoint: PostExecpipelineRunEndpoint(s.GetPipelineRun()),
	}
}

func PostSavePipelineEndpoints(s PipelineService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*SavePipeline)
		err := s.Save(ctx, req)
		return universalResponse{Err: err}, nil
	}
}

func PostExecPipelineEndpoints(s PipelineService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*ExecPipeline)
		err := s.Exec(ctx, req)
		return universalResponse{Err: err}, nil
	}
}

func PostExecpipelineRunEndpoint(s PipelineRunService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*ExecPipelineRun)
		err := s.Exec(ctx, req)
		return universalResponse{Err: err}, nil
	}
}

func (e Endpoints) Save(ctx context.Context, in *SavePipeline) error {
	_, err := e.PostSavePipelineEndpoint(ctx, in)
	return err
}

func (e Endpoints) Exec(ctx context.Context, in *ExecPipeline) error {
	_, err := e.PostExecPipelineEndpoint(ctx, in)
	return err
}

func (e Endpoints) ExecPipelineRun(ctx context.Context, in *ExecPipelineRun) error {
	_, err := e.PostExecpipelineRunEndpoint(ctx, in)
	return err
}

type ur interface {
	GetErr() error
	GetData() interface{}
}

type universalResponse struct {
	Err  error       `json:"err,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

func (u universalResponse) GetErr() error {
	return u.Err
}

func (u universalResponse) GetData() interface{} {
	return u.Data
}
