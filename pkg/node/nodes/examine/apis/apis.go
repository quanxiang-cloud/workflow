package apis

import (
	"context"
	"encoding/json"
	"net/http"

	"git.yunify.com/quanxiang/workflow/pkg/node"
	"git.yunify.com/quanxiang/workflow/pkg/node/nodes/examine/service"
)

func Router(endpoints Endpoints) []node.Option {
	return []node.Option{
		node.WithRouter("POST", "/api/v1/examine/agree", endpoints.AgreeEndpoint, func(ctx context.Context, r *http.Request) (request interface{}, err error) {
			var req service.AgreeRequest
			if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
				return nil, e
			}

			return &req, nil
		},
			func(ctx context.Context, w http.ResponseWriter, response interface{}) error {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				return json.NewEncoder(w).Encode(response)
			}),
		node.WithRouter("POST", "/api/v1/examine/reject", endpoints.RejectEndpoint, func(ctx context.Context, r *http.Request) (request interface{}, err error) {
			var req service.AgreeRequest
			if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
				return nil, e
			}

			return &req, nil
		},
			func(ctx context.Context, w http.ResponseWriter, response interface{}) error {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				return json.NewEncoder(w).Encode(response)
			}),
		node.WithRouter("POST", "/api/v1/examine/recall", endpoints.RecallEndpoint, func(ctx context.Context, r *http.Request) (request interface{}, err error) {
			var req service.AgreeRequest
			if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
				return nil, e
			}

			return &req, nil
		},
			func(ctx context.Context, w http.ResponseWriter, response interface{}) error {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				return json.NewEncoder(w).Encode(response)
			}),
		node.WithRouter("POST", "/api/v1/examine/transfer", endpoints.TransferEndpoint, func(ctx context.Context, r *http.Request) (request interface{}, err error) {
			var req service.AgreeRequest
			if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
				return nil, e
			}

			return &req, nil
		},
			func(ctx context.Context, w http.ResponseWriter, response interface{}) error {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				return json.NewEncoder(w).Encode(response)
			}),
		node.WithRouter("POST", "/api/v1/examine/urge", endpoints.UrgeEndpoint, func(ctx context.Context, r *http.Request) (request interface{}, err error) {
			var req service.AgreeRequest
			if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
				return nil, e
			}

			return &req, nil
		},
			func(ctx context.Context, w http.ResponseWriter, response interface{}) error {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				return json.NewEncoder(w).Encode(response)
			}),
		node.WithRouter("POST", "/api/v1/examine/list", endpoints.ListEndpoint, func(ctx context.Context, r *http.Request) (request interface{}, err error) {
			var req service.AgreeRequest
			if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
				return nil, e
			}

			return &req, nil
		},
			func(ctx context.Context, w http.ResponseWriter, response interface{}) error {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				return json.NewEncoder(w).Encode(response)
			}),
	}
}
