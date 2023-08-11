package versioned

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"git.yunify.com/quanxiang/workflow/apis"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/lb"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
)

type Client interface {
	Exec(ctx context.Context, in *apis.ExecPipeline) error

	Save(ctx context.Context, in *apis.SavePipeline) error

	ExecPipelineRun(ctx context.Context, in *apis.ExecPipelineRun) error
}

func New(instance string, logger log.Logger) Client {
	var endpoints apis.Endpoints
	var instancer sd.FixedInstancer = sd.FixedInstancer{instance}

	var (
		retryMax     = 3
		retryTimeout = 3 * time.Second
	)

	{
		{
			endpointer := sd.NewEndpointer(instancer, factoryFor(func(s Client) endpoint.Endpoint {
				return func(ctx context.Context, request interface{}) (interface{}, error) {
					req := request.(*apis.ExecPipeline)
					err := s.Exec(ctx, req)
					return nil, err
				}
			}), logger)
			balancer := lb.NewRoundRobin(endpointer)
			retry := lb.Retry(retryMax, retryTimeout, balancer)
			endpoints.PostExecPipelineEndpoint = retry
		}

		{
			endpointer := sd.NewEndpointer(instancer, factoryFor(func(s Client) endpoint.Endpoint {
				return func(ctx context.Context, request interface{}) (interface{}, error) {
					req := request.(*apis.SavePipeline)
					err := s.Save(ctx, req)
					return nil, err
				}
			}), logger)
			balancer := lb.NewRoundRobin(endpointer)
			retry := lb.Retry(retryMax, retryTimeout, balancer)
			endpoints.PostSavePipelineEndpoint = retry

		}
		{
			endpointer := sd.NewEndpointer(instancer, factoryFor(func(s Client) endpoint.Endpoint {
				return func(ctx context.Context, request interface{}) (interface{}, error) {
					req := request.(*apis.ExecPipelineRun)
					err := s.ExecPipelineRun(ctx, req)
					return nil, err
				}
			}), logger)
			balancer := lb.NewRoundRobin(endpointer)
			retry := lb.Retry(retryMax, retryTimeout, balancer)
			endpoints.PostExecpipelineRunEndpoint = retry

		}

	}

	return endpoints
}

func factoryFor(makeEndpoint func(Client) endpoint.Endpoint) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		service, err := NewClientEndPoints(instance)
		if err != nil {
			return nil, nil, err
		}
		return makeEndpoint(service), nil, nil
	}
}

func NewClientEndPoints(instance string) (apis.Endpoints, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	tgt, err := url.Parse(instance)
	if err != nil {
		return apis.Endpoints{}, err
	}
	tgt.Path = ""

	options := []httptransport.ClientOption{}

	return apis.Endpoints{
		PostExecPipelineEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			req := request.(*apis.ExecPipeline)

			r.URL.Path = fmt.Sprintf("/api/v1/pipeline/%s/exec", req.Name)

			return encodeRequest(ctx, r, req)
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return nil, errors.New(resp.Status)
			}
			return response, err
		}, options...).Endpoint(),
		PostSavePipelineEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			req := request.(*apis.SavePipeline)

			r.URL.Path = "/api/v1/pipeline"

			return encodeRequest(ctx, r, req)
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return nil, errors.New(resp.Status)
			}
			return response, err
		}, options...).Endpoint(),
		PostExecpipelineRunEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			execPipelineRun := request.(*apis.ExecPipelineRun)
			runID := strconv.FormatInt(execPipelineRun.ID, 10)
			r.URL.Path = "/api/v1/pipelineRun/" + runID + "/exec"

			return encodeRequest(ctx, r, request)
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return nil, errors.New(resp.Status)
			}
			return response, err
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
