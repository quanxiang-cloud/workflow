package apis

import (
	"context"
	"encoding/json"
	"net/http"

	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/transport"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
)

func NewHTTPHandler(s Service, logger log.Logger) http.Handler {
	r := gin.Default()
	e := NewServerEndpoints(s)

	options := []httptransport.ServerOption{
		httptransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		httptransport.ServerErrorEncoder(encodeError),
	}

	{
		group := r.Group("/api/v1")
		group.POST("/trigger", gin.WrapH(httptransport.NewServer(
			e.PostCreateTriggerEndpoint,
			func(ctx context.Context, r *http.Request) (request interface{}, err error) {
				var req CreateTrigger
				req.Data = s.GetTrigger().GetDataType(ctx)
				return reqJSON(&req)(ctx, r)
			},
			responseJSON,
			options...,
		)))

		group.DELETE("/trigger/:name", func(c *gin.Context) {
			name := c.Param("name")
			httptransport.NewServer(
				e.DeleteRemoveTriggerEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req DeleteTrigger
					req.Name = name
					return &req, nil
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
	}

	return r
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	if err == nil {
		panic("encodeError with nil error")
	}

	var ce *errors.Error
	if errors.As(err, &ce) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(ce.Code)
		if ce.Message != nil {
			w.Write([]byte(ce.Message.JSON()))
		}
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
}

func responseJSON(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if ur, ok := response.(ur); ok {
		if err := ur.GetErr(); err != nil {
			encodeError(ctx, err, w)
			return nil
		}
		return json.NewEncoder(w).Encode(ur.GetData())
	}
	return json.NewEncoder(w).Encode(response)
}

func reqJSON(v any) httptransport.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (request interface{}, err error) {
		if e := json.NewDecoder(r.Body).Decode(v); e != nil {
			return nil, e
		}

		return v, nil
	}
}
