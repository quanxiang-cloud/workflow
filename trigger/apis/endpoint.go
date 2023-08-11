package apis

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

// Endpoints collects all of the endpoints that compose a workflow service.
type Endpoints struct {
	PostCreateTriggerEndpoint   endpoint.Endpoint
	DeleteRemoveTriggerEndpoint endpoint.Endpoint
}

// NewServerEndpoints returns an Endpoints struct where each endpoint invokes
// the corresponding method on the provided service. Useful in a all-svc
// server.
func NewServerEndpoints(s Service) Endpoints {
	return Endpoints{
		PostCreateTriggerEndpoint:   PostCreateTriggerEndpoint(s.GetTrigger()),
		DeleteRemoveTriggerEndpoint: DeleteRemoveTriggerEndpoint(s.GetTrigger()),
	}
}

func PostCreateTriggerEndpoint(s TriggerService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*CreateTrigger)
		err := s.Create(ctx, req)
		return universalResponse{Err: err}, nil
	}
}

func DeleteRemoveTriggerEndpoint(s TriggerService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*DeleteTrigger)
		err := s.Delete(ctx, req)
		return universalResponse{Err: err}, nil
	}
}

func (e Endpoints) Create(ctx context.Context, in *CreateTrigger) error {
	_, err := e.PostCreateTriggerEndpoint(ctx, in)
	return err
}
func (e Endpoints) Delete(ctx context.Context, in *DeleteTrigger) error {
	_, err := e.DeleteRemoveTriggerEndpoint(ctx, in)
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
