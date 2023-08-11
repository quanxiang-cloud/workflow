package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	wl "git.yunify.com/quanxiang/workflow/pkg/log"
	"git.yunify.com/quanxiang/workflow/pkg/node/nodes"
	"git.yunify.com/quanxiang/workflow/pkg/thirdparty/quanxiang"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/kelseyhightower/envconfig"

	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
	"git.yunify.com/quanxiang/workflow/pkg/node"
)

type From struct {
	logger log.Logger

	qx quanxiang.QuanXiang
}

func (f *From) Do(_ context.Context, in *node.Request) (*node.Result, error) {
	ctx := context.Background()
	level.Info(f.logger).Log("message", "try to update form data")
	rule, err := genRule(in.Params)
	if err != nil {
		level.Error(f.logger).Log("message", err)
		return &node.Result{}, err
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

	// query
	qs := make([]interface{}, 0)

	for _, cd := range rule.filterRule.Conditions {
		if cd.Operator == "eq" {
			term := map[string]interface{}{
				"term": map[string]interface{}{
					cd.FieldName: sourceData[cd.Value],
				},
			}
			qs = append(qs, term)
		}
	}

	query := make(map[string]interface{})
	if rule.filterRule.Tag == "and" {
		query = map[string]interface{}{
			"must": qs,
		}
	} else {
		query = map[string]interface{}{
			"should": query,
		}
	}
	if len(rule.filterRule.Conditions) == 0 {
		term := map[string]interface{}{
			"term": map[string]interface{}{
				"_id": sourceData["_id"],
			},
		}
		query = term
	} else {
		query = map[string]interface{}{
			"bool": query,
		}
	}

	// entity
	entity := make(map[string]interface{})
	for _, value := range rule.updateRule {
		var v interface{}
		switch value.ValueFrom {
		case "fixedValue":

			v = value.ValueOf
		case "currentFormValue":
			v = sourceData[value.ValueOf.(string)]
		case "formula":
			expr := value.ValueOf.(string)
			expr = strings.ReplaceAll(expr, "$flowVar_", "[$variable.flowVar_")
			expr = strings.ReplaceAll(expr, "$field_", "[$formData.field_")

			buf := bytes.Buffer{}
			var flag bool
			for _, c := range expr {
				if c == '$' {
					flag = true
				}
				if flag && (c == ' ') { /*|| c == '!' || c == '>' || c == '<' || c == '=' ||
					c == '+' || c == '-' || c == '*' || c == '/' */
					buf.WriteRune(']')
					flag = false
				}

				buf.WriteRune(c)
			}

			v, _ = nodes.Evaluate(buf.String(), sourceData, rule.communal)
		case "processVariable":
			v = rule.communal[value.ValueOf.(string)]
		}

		entity[value.FieldName] = v
	}

	_, err = f.qx.UpdateFormData(ctx, &quanxiang.UpdateFormDataRequest{
		AppID:  rule.appID,
		FormID: rule.targetTableID,
		Data: quanxiang.UpdateEntity{
			Query:  query,
			Entity: entity,
		},
	})

	if err != nil {
		level.Error(f.logger).Log("message", err, "appID", rule.appID, "tableID", rule.tableID, "dataID", rule.dataID)
		return &node.Result{}, errors.Wrap(err, "fail update from data")
	}

	level.Info(f.logger).Log("message", "update from data", "appID", rule.appID, "tableID", rule.tableID, "query", query)
	return &node.Result{
		Status: v1alpha1.Finish,
	}, nil
}

type rule struct {
	appID         string
	tableID       string
	dataID        string
	targetTableID string
	filterRule    struct {
		Tag        string `json:"tag,omitempty"`
		Conditions []struct {
			FieldName string `json:"fieldName,omitempty"`
			Operator  string `json:"operator,omitempty"`
			Value     string `json:"value,omitempty"`
		} `json:"conditions,omitempty"`
	}
	updateRule []updateRule
	communal   map[string]interface{}
}

type updateRule struct {
	FieldName string      `json:"fieldName,omitempty"`
	ValueFrom string      `json:"valueFrom,omitempty"`
	ValueOf   interface{} `json:"valueOf,omitempty"`
}

func genRule(in []*v1alpha1.KeyAndValue) (*rule, error) {
	rule := &rule{
		communal:   map[string]interface{}{},
		updateRule: make([]updateRule, 0),
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
		case "filterRule":
			err := json.Unmarshal([]byte(elem.Value), &rule.filterRule)
			if err != nil {
				return nil, errors.Wrap(err, "fail unmashal filter rule")
			}
		case "updateRule":
			err := json.Unmarshal([]byte(elem.Value), &rule.updateRule)
			if err != nil {
				return nil, errors.Wrap(err, "fail unmashal update rule")
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
