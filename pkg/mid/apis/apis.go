package apis

import (
	"context"
	"encoding/json"
	"git.yunify.com/quanxiang/workflow/internal/common"
	"net/http"

	triggerclinet "git.yunify.com/quanxiang/trigger/pkg/client"
	"git.yunify.com/quanxiang/workflow/pkg/client/clientset/versioned"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"git.yunify.com/quanxiang/workflow/pkg/mid/service"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/transport"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
)

func NewHTTPHandler(logger log.Logger, wl versioned.Client, trigger triggerclinet.Trigger, confMysql common.Mysql, workFlowInstance, homeHost string) http.Handler {
	r := gin.Default()
	e := NewEndPoints(logger, wl, trigger, confMysql, workFlowInstance, homeHost)

	options := []httptransport.ServerOption{
		httptransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		httptransport.ServerErrorEncoder(encodeError),
	}

	{
		group := r.Group("/api/v1/flow")
		group.POST("/flowList", func(c *gin.Context) {
			userID := c.GetHeader("User-Id")
			httptransport.NewServer(
				e.GetFlowListEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.GetFlowListRequest
					req.UserID = userID
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/flowInfo/:id", func(c *gin.Context) {
			id := c.Param("id")
			userID := c.GetHeader("User-Id")
			httptransport.NewServer(
				e.GetFlowInfoEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.GetFlowInfoRequest
					req.ID = id
					req.UserID = userID
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/deleteFlow/:id", func(c *gin.Context) {
			id := c.Param("id")
			httptransport.NewServer(
				e.DeleteFlowEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.DeleteFlowRequest
					req.ID = id
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/saveFlow", func(c *gin.Context) {
			userID := c.GetHeader("User-Id")
			userName := c.GetHeader("User-Name")
			httptransport.NewServer(
				e.SaveFlowEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.SaveFlowRequest
					req.UserID = userID
					req.UserName = userName
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/updateFlowStatus", func(c *gin.Context) {

			httptransport.NewServer(
				e.UpdateFlowStatusEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.UpdateFlowRequest
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/getVariableList", func(c *gin.Context) {
			id := c.Query("id")
			httptransport.NewServer(
				e.GetFlowVariableListEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.GetFlowVariableListRequest
					req.ID = id
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/saveFlowVariable", func(c *gin.Context) {

			httptransport.NewServer(
				e.SaveFlowVariableEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.SaveFlowVariableRequest
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/deleteFlowVariable/:id", func(c *gin.Context) {

			id := c.Param("id")

			httptransport.NewServer(
				e.DeleteFlowVariableEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.DeleteFlowVariableRequest
					req.ID = id
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		//------------------
		group.POST("/instance/myApplyList", func(c *gin.Context) {

			userID := c.GetHeader("User-Id")

			httptransport.NewServer(
				e.MyApplyListEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.MyApplyListRequest
					req.UserID = userID
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/instance/getFlowInstanceCount", func(c *gin.Context) {
			var req service.PendingExamineCountRequest
			userID := c.GetHeader("User-Id")
			req.UserID = userID
			httptransport.NewServer(
				e.PendingExamineCountEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/instance/waitReviewList", func(c *gin.Context) {
			userID := c.GetHeader("User-Id")
			httptransport.NewServer(
				e.PendingExamineListEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.PendingExamineListRequest
					req.UserID = userID
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/instance/reviewedList", func(c *gin.Context) {

			httptransport.NewServer(
				e.ExaminedListEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.ExaminedListRequest
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/instance/ccToMeList", func(c *gin.Context) {

			httptransport.NewServer(
				e.CCToMeListEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.CCToMeListRequest
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/instance/getFlowInstanceForm/:processInstanceID", func(c *gin.Context) {

			processInstanceID := c.Param("processInstanceID")
			userID := c.GetHeader("User-Id")
			httptransport.NewServer(
				e.GetFormFieldPermissionEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.GetFormFieldPermissionRequest
					req.ProcessInstanceID = processInstanceID
					req.UserID = userID
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/instance/processHistories/:processInstanceID", func(c *gin.Context) {

			processInstanceID := c.Param("processInstanceID")
			userID := c.GetHeader("User-Id")
			httptransport.NewServer(
				e.GetFlowProcessEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.GetFlowProcessRequest
					req.ProcessInstanceID = processInstanceID
					req.UserID = userID
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/instance/getFormData/:processInstanceID/:taskID", func(c *gin.Context) {

			processInstanceID := c.Param("processInstanceID")
			taskID := c.Param("taskID")

			httptransport.NewServer(
				e.GetFormDataEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.GetFormDataRequest
					req.ProcessInstanceID = processInstanceID
					req.TaskID = taskID
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/instance/reviewTask/:processInstanceID/:taskID", func(c *gin.Context) {

			processInstanceID := c.Param("processInstanceID")
			taskID := c.Param("taskID")
			userID := c.GetHeader("User-Id")

			httptransport.NewServer(
				e.ExamineEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.ExamineRequest
					req.UserID = userID
					req.TaskID = taskID
					req.ProcessInstanceID = processInstanceID
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/urge/taskUrge", func(c *gin.Context) {

			processInstanceID := c.Param("processInstanceID")
			userID := c.GetHeader("User-Id")

			httptransport.NewServer(
				e.UrgeEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.UrgeRequest
					req.ProcessInstanceID = processInstanceID
					req.UserID = userID
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/instance/cancel/:processInstanceId", func(c *gin.Context) {

			processInstanceID := c.Param("processInstanceID")

			httptransport.NewServer(
				e.RecallEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.RecallRequest
					req.ProcessInstanceID = processInstanceID
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/appReplicationExport", func(c *gin.Context) {
			userID := c.GetHeader("User-Id")
			userName := c.GetHeader("User-Name")
			httptransport.NewServer(
				e.AppReplicationExportEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {
					var req service.AppExportRequest
					req.UserID = userID
					req.UserName = userName
					return reqJSON(&req)(ctx, r)
				},
				responseJSON,
				options...,
			).ServeHTTP(c.Writer, c.Request)
		})
		group.POST("/appReplicationImport", func(c *gin.Context) {
			userID := c.GetHeader("User-Id")
			userName := c.GetHeader("User-Name")
			httptransport.NewServer(
				e.AppReplicationImportEndpoint,
				func(ctx context.Context, r *http.Request) (request interface{}, err error) {

					var req service.AppImportRequest
					req.UserID = userID
					req.UserName = userName
					return reqJSON(&req)(ctx, r)
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
