package main

import (
	"context"
	"encoding/json"
	"fmt"

	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	wl "git.yunify.com/quanxiang/workflow/pkg/log"
	"git.yunify.com/quanxiang/workflow/pkg/node"
	"git.yunify.com/quanxiang/workflow/pkg/node/nodes"
	quanxiangform "git.yunify.com/quanxiang/workflow/pkg/node/nodes/quanxiang_form"
	"git.yunify.com/quanxiang/workflow/pkg/thirdparty/quanxiang"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/kelseyhightower/envconfig"
)

type From struct {
	logger log.Logger

	qx quanxiang.QuanXiang
}

func (f *From) Do(_ context.Context, in *node.Request) (*node.Result, error) {
	ctx := context.Background()
	level.Info(f.logger).Log("message", "exec creat form data start")
	rule, err := genRule(in.Params)
	if err != nil {
		level.Error(f.logger).Log("message", err)
		return &node.Result{}, errors.Wrap(err, "fail get from data")
	}

	var sourceData map[string]interface{}
	if rule.dataID != "" && rule.tableID != "" {
		result, err := f.qx.GetFormData(ctx, &quanxiang.GetFormDataRequest{
			AppID:  rule.appID,
			FormID: rule.tableID,
			DataID: rule.dataID,
		})
		if err != nil {
			level.Error(f.logger).Log("message", err, "appID", rule.appID, "tableID", rule.tableID, "dataID", rule.dataID)
			return &node.Result{}, errors.Wrap(err, "fail get from data")
		}
		sourceData = result.Entity
	}

	entity := make(map[string]interface{})
	for key, value := range rule.createRule {
		var v interface{}
		switch value.ValueFrom {
		case "fixedValue":
			v = value.ValueOf
		case "currentFormValue":
			v = sourceData[value.ValueOf.(string)]
		case "formula":
			v, err = nodes.Evaluate(value.ValueOf.(string), sourceData, rule.communal)
			_ = err
		case "processVariable":
			v = rule.communal[value.ValueOf.(string)]
		}

		entity[key] = v
	}

	_, err = f.qx.CreateFormData(ctx, &quanxiang.CreateFormDataRequest{
		AppID:  rule.appID,
		FormID: rule.targetTableID,
		Data: quanxiang.Data{
			Entity: entity,
		},
	})

	if err != nil {
		level.Error(f.logger).Log("message", err, "appID", rule.appID, "targetID", rule.targetTableID)
		return &node.Result{}, errors.Wrap(err, "fail insert from data")
	}

	level.Info(f.logger).Log("message", "success", "appID", rule.appID, "targetID", rule.targetTableID)
	return &node.Result{
		Status: v1alpha1.Finish,
	}, nil
}

type rule struct {
	appID         string
	tableID       string
	dataID        string
	targetTableID string
	createRule    map[string]quanxiangform.FlowFormValueRule
	communal      map[string]interface{}
}

func genRule(in []*v1alpha1.KeyAndValue) (*rule, error) {
	rule := &rule{
		communal: make(map[string]interface{}),
	}
	for _, elem := range in {
		switch elem.Key {
		case "appID":
			rule.appID = elem.Value
		case "tableID":
			rule.tableID = elem.Value
		case "dataID":
			rule.dataID = elem.Value
		case "targetTableID":
			rule.targetTableID = elem.Value
		case "createRule":
			err := json.Unmarshal([]byte(elem.Value), &rule.createRule)
			if err != nil {
				return nil, errors.Wrap(err, "fail unmashal create rule")
			}
		default:
			rule.communal[elem.Key] = elem.Value
		}
	}

	return rule, nil
}

type config struct {
	LogLevel           string   `envconfig:"LOG_LEVEL" default:"debug"`
	QuanxiangInstances []string `envconfig:"QUANXIANG_INSTANCES"`
	Port               string   `envconfig:"PORT" default:"8085"`
}

func main() {
	conf := &config{}
	envconfig.MustProcess("", conf)

	logger := wl.NewLogger(conf.LogLevel)
	s := &From{
		logger: logger,
	}
	ctx := context.Background()
	s.qx = quanxiang.New(conf.QuanxiangInstances, logger)
	node.Main(logger, fmt.Sprintf(":%s", conf.Port))(ctx, s)
}
