package node

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/lb"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
)

type Endpoints struct {
	DoEndpoint endpoint.Endpoint
}

func NewEndPoints(s Interface) Endpoints {
	return Endpoints{
		DoEndpoint: DoEndpoint(s),
	}
}

func New(instance []string, logger log.Logger) Interface {
	var endpoints Endpoints
	var instancer sd.FixedInstancer = sd.FixedInstancer(instance)

	var (
		retryMax     = 3
		retryTimeout = 3 * time.Second
	)

	{
		factory := factoryFor(DoEndpoint)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.DoEndpoint = retry
	}

	return endpoints
}

func factoryFor(makeEndpoint func(Interface) endpoint.Endpoint) sd.Factory {
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
		DoEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			r.URL.Path = "/api/v1/do"
			req := request.(*Request)

			return encodeRequest(ctx, r, req)
		}, func(ctx context.Context, resp *http.Response) (interface{}, error) {
			var response *Result
			err := json.NewDecoder(resp.Body).Decode(&response)
			return response, err
		}, options...).Endpoint(),
	}, nil
}

// encodeRequest likewise JSON-encodes the request to the HTTP request body.
// Don't use it directly as a transport/http.Client EncodeRequestFunc:
// profilesvc endpoints require mutating the HTTP method and request path.
func encodeRequest(_ context.Context, req *http.Request, request interface{}) error {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(&buf)
	return nil
}

func DoEndpoint(s Interface) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*Request)
		return s.Do(ctx, req)
	}
}
func (e Endpoints) Do(ctx context.Context, in *Request) (*Result, error) {
	resp, err := e.DoEndpoint(ctx, in)
	if err != nil {
		return &Result{}, err
	}

	return resp.(*Result), nil
}
