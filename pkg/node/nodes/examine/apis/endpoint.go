package apis

import (
	"context"
	"github.com/quanxiang-cloud/cabin/tailormade/resp"

	"git.yunify.com/quanxiang/workflow/pkg/node"
	"git.yunify.com/quanxiang/workflow/pkg/node/nodes/examine/service"
	"github.com/go-kit/kit/endpoint"
)

type Endpoints struct {
	node.Endpoints
	// portal
	AgreeEndpoint    endpoint.Endpoint
	RejectEndpoint   endpoint.Endpoint
	RecallEndpoint   endpoint.Endpoint
	TransferEndpoint endpoint.Endpoint
	UrgeEndpoint     endpoint.Endpoint
	ListEndpoint     endpoint.Endpoint
}

func NewEndPoints(s service.Task) Endpoints {
	end := Endpoints{}

	e := node.NewEndPoints(s)
	end.Endpoints = e

	end.AgreeEndpoint = AgreeEndpoint(s)
	end.RejectEndpoint = RejectEndpoint(s)
	end.RecallEndpoint = RecallEndpoint(s)
	end.UrgeEndpoint = UrgeEndpoint(s)
	end.TransferEndpoint = TransferEndpoint(s)
	end.UrgeEndpoint = UrgeEndpoint(s)
	end.ListEndpoint = ListEndpoint(s)
	return end
}

func AgreeEndpoint(s service.Task) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.AgreeRequest)
		response, err := s.Agree(ctx, req)
		r := resp.Format(response, err)
		return r, nil
	}
}
func RejectEndpoint(s service.Task) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.RejectRequest)
		response, err := s.Reject(ctx, req)
		r := resp.Format(response, err)
		return r, nil
	}
}

func RecallEndpoint(s service.Task) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.RecallRequest)
		response, err := s.Recall(ctx, req)
		r := resp.Format(response, err)
		return r, nil
	}
}
func UrgeEndpoint(s service.Task) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.UrgeRequest)
		response, err := s.Urge(ctx, req)
		r := resp.Format(response, err)
		return r, nil
	}
}

func TransferEndpoint(s service.Task) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*service.TransferRequest)
		response, err := s.Transfer(ctx, req)
		r := resp.Format(response, err)
		return r, err
	}
}

func ListEndpoint(s service.Task) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		//req := request.(*service.ListRequest)
		//response, err := s.List(ctx, req)
		//r := resp.Format(response, err)
		return nil, nil
	}
}
