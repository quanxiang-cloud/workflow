package main

import (
	"context"
	"database/sql"
	stderr "errors"
	"net/http"
	"time"

	"git.yunify.com/quanxiang/trigger/internal/database"
	"git.yunify.com/quanxiang/trigger/internal/database/mysql"
	"git.yunify.com/quanxiang/trigger/internal/service"
	"git.yunify.com/quanxiang/trigger/pkg/trigger"
	"git.yunify.com/quanxiang/workflow/apis"
	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
	"git.yunify.com/quanxiang/workflow/pkg/client/clientset/versioned"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/groupcache/lru"
)

type form struct {
	conf   *Config
	logger log.Logger

	formRepo database.FormRepo

	plr versioned.Client
	lru *lru.Cache
}

type FromService interface {
	trigger.Interface

	Exec(ctx context.Context, in *Data) error
}

func New(ctx context.Context, conf *Config, opts ...service.Option) (FromService, error) {
	form := &form{
		conf: conf,
	}

	for _, opt := range opts {
		opt(form)
	}

	form.plr = versioned.New(form.conf.WorkflowInstance, form.logger)
	if conf.NVl.Enable {
		form.lru = lru.New(conf.NVl.BufferLength)
	}
	return form, nil
}

func (f *form) SetLogger(logger log.Logger) {
	f.logger = logger
}

func (f *form) SetDB(db *sql.DB) {
	f.formRepo = mysql.NewForm(db)
}

type FormData struct {
	AppID   string `json:"app_id,omitempty"`
	TableID string `json:"table_id,omitempty"`

	// Type is form data action, creade or update
	Type    string   `json:"type,omitempty"`
	Filters []string `json:"filters,omitempty"`
}

func (f *form) Add(ctx context.Context, name, pipelineName string, data trigger.Data) error {
	formData := data.(*FormData)
	if formData.Type != "CREATE" && formData.Type != "UPDATE" {
		return errors.NewErr(http.StatusBadRequest, &errors.CodeError{
			Code:    http.StatusBadRequest,
			Message: "form type must be CREATE or UPDATED",
		})
	}

	fl, err := f.formRepo.Get(ctx, name)
	if err != nil {
		level.Error(f.logger).Log("message", err.Error(), "name", name, "app_id", formData.AppID, "table_id", formData.TableID, "type", formData.Type)
		return errors.Wrap(err, "fail get form trigger")
	}

	if fl != nil {
		return errors.NewErr(http.StatusConflict)
	}

	fl = &database.FormTrigger{
		Name:         name,
		PipelineName: pipelineName,
		AppID:        formData.AppID,
		TableID:      formData.TableID,
		Type:         formData.Type,
		Filters:      formData.Filters,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}
	err = f.formRepo.Create(ctx, fl)
	if err != nil {
		return errors.Wrap(err, "fail insert form trigger to database")
	}

	return nil
}
func (f *form) Remove(ctx context.Context, name string) error {
	fl, err := f.formRepo.Get(ctx, name)
	if err != nil {
		level.Error(f.logger).Log("message", err.Error(), "name", name)
		return errors.Wrap(err, "fail get form trigger")
	}

	if fl == nil {
		return nil
	}

	err = f.formRepo.Delete(ctx, fl.ID)
	if err != nil {
		level.Error(f.logger).Log("message", err, "formLockID", fl.ID)
		return errors.Wrap(err, "fail delete form trigger from database")
	}

	return nil
}
func (f *form) Run(ctx context.Context) {
}

func (f *form) GetDataType(ctx context.Context) trigger.Data {
	return &FormData{}
}

type Data struct {
	TableID string      `json:"tableID"`
	Entity  interface{} `json:"entity"`
	Magic   string      `json:"magic"`
	Seq     string      `json:"seq"`
	Version string      `json:"version"`
	Method  string      `json:"method"`
}

func (f *form) Exec(ctx context.Context, in *Data) error {
	level.Info(f.logger).Log("message", "try to exec", "tableID", in.TableID)

	var _t string

	entity, ok := in.Entity.(map[string]interface{})
	if !ok {
		err := stderr.New("form data structures cannot be parsed correctly")
		level.Info(f.logger).Log("message", err.Error(), "tableID", in.TableID)
		return err
	}

	switch in.Method {
	case "post":
		_t = "CREATE"
		if id, ok := entity["creator_id"]; !ok || id == "" {
			level.Error(f.logger).Log("message", "can not get creator_id from from-data", "tableID", in.TableID)
			return nil
		}
	case "put":
		_t = "UPDATE"
		if id, ok := entity["modifier_id"]; !ok || id == "" {
			level.Error(f.logger).Log("message", "can not get modifier_id from from-data", "tableID", in.TableID)
			return nil
		}

	default:
		level.Info(f.logger).Log("message", "unexpected methods, discard", "method", in.Method, "tableID", in.TableID)
		return nil
	}

	if !f.isCompliance(in.TableID + in.Method) {
		level.Info(f.logger).Log("message", "compliance table", "tableID", in.TableID)
		return nil
	}

	fts, err := f.formRepo.GetByTableID(ctx, in.TableID, _t)
	if err != nil {
		level.Error(f.logger).Log("message", err, "tableID", in.TableID, "type", _t)
		return errors.Wrap(err, "fail get form trigger")
	}

	if len(fts) == 0 {
		f.addCache(in.TableID + in.Method)
	}

	err = f.processQuanxiangFrom(ctx, fts, in)
	if err != nil {
		level.Error(f.logger).Log("message", err, "tableID", in.TableID, "type", _t)
		return errors.Wrap(err, "fail process quanxaing form data")
	}

	return nil
}

func (f *form) processQuanxiangFrom(ctx context.Context, fts []*database.FormTrigger, in *Data) error {
	if len(fts) == 0 {
		return nil
	}
	entity := in.Entity.(map[string]interface{})

	_id, ok := entity["_id"]
	if !ok {
		return stderr.New("can not get form data id")
	}

	dataID, ok := _id.(string)
	if !ok {
		return stderr.New("can not change id type to string")
	}

	for _, ft := range fts {
		if ft.Type == "UPDATE" {
			var flag bool
			for _, filter := range ft.Filters {
				_, ok := entity[filter]
				if ok {
					flag = ok
					break
				}

			}
			if !flag {
				level.Info(f.logger).Log("message", "the interceptor cannot be satisfied", "tableID", in.TableID, "_id", _id)
				return nil
			}
		}

		level.Info(f.logger).Log("message", "try to exec", "appID", ft.AppID, "tableID", ft.TableID, "id", dataID, "pipelineName", ft.PipelineName)
		err := f.plr.Exec(ctx, &apis.ExecPipeline{
			Name: ft.PipelineName,
			Params: []*v1alpha1.KeyAndValue{
				{
					Key:   "appID",
					Value: ft.AppID,
				}, {
					Key:   "tableID",
					Value: ft.TableID,
				}, {
					Key:   "dataID",
					Value: dataID,
				},
			},
		})
		if err != nil {
			level.Error(f.logger).Log("message", err, "appID", ft.AppID, "tableID", ft.TableID, "dataID", dataID)
		}
	}

	return nil
}

func (f *form) isCompliance(id string) bool {
	if f.conf.NVl.Enable {
		value, ok := f.lru.Get(id)
		if !ok {
			return true
		}
		if time.Since(value.(time.Time)) >= f.conf.NVl.TTL {
			f.lru.RemoveOldest()
			f.lru.Remove(id)
			return true
		}
		return false
	}
	return true
}

func (f *form) addCache(id string) {
	if f.conf.NVl.Enable {
		f.lru.Add(id, time.Now())
	}
}

func oneOf(src string, dst []string) bool {
	for _, str := range dst {
		if str == src {
			return true
		}
	}
	return false
}
