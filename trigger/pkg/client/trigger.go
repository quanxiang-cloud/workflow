package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.yunify.com/quanxiang/trigger/apis"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/lb"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
)

type Trigger interface {
	Create(ctx context.Context, in *apis.CreateTrigger) error
	Delete(ctx context.Context, in *apis.DeleteTrigger) error
}

type Endpoints struct {
	CreateEndpoints endpoint.Endpoint
	DeleteEndpoints endpoint.Endpoint
}

func (e Endpoints) Create(ctx context.Context, in *apis.CreateTrigger) error {
	_, err := e.CreateEndpoints(ctx, in)
	return err
}

func CreateEndpoint(e Trigger) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*apis.CreateTrigger)
		return nil, e.Create(ctx, req)
	}
}

func (e Endpoints) Delete(ctx context.Context, in *apis.DeleteTrigger) error {
	_, err := e.DeleteEndpoints(ctx, in)
	return err
}

func DeleteEndpoint(e Trigger) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*apis.DeleteTrigger)
		err := e.Delete(ctx, req)
		return nil, err
	}
}

func New(instance string, logger log.Logger) Trigger {
	var endpoints Endpoints
	var instancer sd.FixedInstancer = sd.FixedInstancer{instance}

	var (
		retryMax     = 3
		retryTimeout = 15 * time.Second
	)

	{
		factory := factoryFor(CreateEndpoint)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.CreateEndpoints = retry
	}
	{
		factory := factoryFor(DeleteEndpoint)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.DeleteEndpoints = retry
	}
	return endpoints
}

func factoryFor(makeEndpoint func(e Trigger) endpoint.Endpoint) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		service, err := NewClientEndPoints(instance)
		if err != nil {
			return nil, nil, err
		}
		return makeEndpoint(service), nil, nil
	}
}

func NewClientEndPoints(instance string) (Endpoints, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	tgt, err := url.Parse(instance)
	if err != nil {
		return Endpoints{}, err
	}
	tgt.Path = ""

	options := []httptransport.ClientOption{}

	return Endpoints{
		CreateEndpoints: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			r.URL.Path = "/api/v1/trigger"
			encodeRequest(ctx, r, request)
			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return nil, errors.NewErr(http.StatusInternalServerError, &errors.CodeError{
					Code:    resp.StatusCode,
					Message: "fail create trigger",
				})
			}

			return nil, err
		}, options...).Endpoint(),
		DeleteEndpoints: httptransport.NewClient("DELETE", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			req := request.(*apis.DeleteTrigger)
			r.URL.Path = "/api/v1/trigger/" + req.Name

			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return nil, errors.NewErr(http.StatusInternalServerError, &errors.CodeError{
					Code:    resp.StatusCode,
					Message: "fail delete trigger",
				})
			}

			return nil, err
		}, options...).Endpoint(),
	}, nil
}

func encodeRequest(_ context.Context, req *http.Request, request interface{}) error {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(&buf)
	return nil
}
