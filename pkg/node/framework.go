package node

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-kit/kit/endpoint"

	"github.com/go-kit/kit/transport"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
)

func NewHTTPHandler(ctx context.Context, s Interface, logger log.Logger, opts ...Option) http.Handler {
	r := mux.NewRouter()
	e := NewEndPoints(s)

	options := []httptransport.ServerOption{
		httptransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		httptransport.ServerErrorEncoder(func(_ context.Context, err error, w http.ResponseWriter) {
			if err == nil {
				panic("encodeError with nil error")
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": err.Error(),
			})
		}),
	}

	r.Methods("POST").Path("/api/v1/do").Handler(httptransport.NewServer(
		e.DoEndpoint,
		func(ctx context.Context, r *http.Request) (request interface{}, err error) {
			var req *Request
			if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
				return nil, e
			}

			return req, nil
		},
		func(ctx context.Context, w http.ResponseWriter, response interface{}) error {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			return json.NewEncoder(w).Encode(response)
		},
		options...,
	))

	return r
}

func Main(logger log.Logger, addr string) func(ctx context.Context, s Interface, opts ...Option) error {
	return func(ctx context.Context, s Interface, opts ...Option) error {
		h := NewHTTPHandler(ctx, s, logger, opts...)
		server := &http.Server{
			Addr:    addr,
			Handler: h,
		}

		ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
		defer stop()
		go func() {
			<-ctx.Done()
			level.Info(logger).Log("message", "Shutting down")
			shutdownCtx, cancel := context.WithTimeout(
				context.Background(),
				time.Second*5,
			)
			defer cancel()
			server.Shutdown(shutdownCtx) // nolint: errcheck
		}()

		level.Info(logger).Log("message", "Starting...", "addr", addr)

		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			level.Error(logger).Log("message", err.Error())
			return err
		}
		return nil
	}
}

type Option func(r *mux.Route, options ...httptransport.ServerOption)

func WithRouter(method string,
	path string,
	endpoint endpoint.Endpoint,
	dec httptransport.DecodeRequestFunc,
	enc httptransport.EncodeResponseFunc,
	opts ...httptransport.ServerOption) Option {

	return func(r *mux.Route, options ...httptransport.ServerOption) {
		r.Methods(method).Path(path).Handler(httptransport.NewServer(
			endpoint,
			dec,
			enc,
			append(options, opts...)...,
		))
	}
}
