package main

import (
	"bytes"
	"context"
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

type ProcessBranch struct {
	logger log.Logger

	qx quanxiang.QuanXiang
}

func (p *ProcessBranch) Do(_ context.Context, in *node.Request) (*node.Result, error) {
	ctx := context.Background()

	rule := genRule(in.Params)
	// exprs := quanxiangform.ParseExpr(rule.rule)

	var formData map[string]interface{}
	if strings.HasPrefix(rule.rule, "$field_") {
		result, err := p.qx.GetFormData(ctx, &quanxiang.GetFormDataRequest{
			AppID:  rule.appID,
			FormID: rule.tableID,
			DataID: rule.dataID,
		})
		if err != nil {
			level.Error(p.logger).Log("message", err, "appID", rule.appID, "tableID", rule.tableID, "dataID", rule.dataID)
			return &node.Result{}, errors.Wrap(err, "fail get from data")
		}

		formData = result.Entity
	}

	var fail = func() (*node.Result, error) {
		return &node.Result{
			Status: v1alpha1.Finish,
			Out: []*v1alpha1.KeyAndValue{
				{Key: "ok", Value: "false"},
			},
		}, nil
	}

	expr := rule.rule
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

	result, err := nodes.Evaluate(buf.String(), formData, nodes.GuessCommunal(rule.communal))
	if err != nil {
		level.Error(p.logger).Log("message", err, "rule", rule.rule)
		return fail()
	}

	if result, ok := result.(bool); !(ok && result == ok) {
		return fail()
	}

	return &node.Result{
		Status: v1alpha1.Finish,
		Out: []*v1alpha1.KeyAndValue{
			{Key: "ok", Value: "true"},
		},
	}, nil
}

type rule struct {
	appID    string
	tableID  string
	dataID   string
	rule     string
	communal map[string]string
}

func genRule(in []*v1alpha1.KeyAndValue) *rule {
	rule := &rule{
		communal: map[string]string{},
	}
	for _, elem := range in {
		switch elem.Key {
		case "appID":
			rule.appID = elem.Value
		case "tableID":
			rule.tableID = elem.Value
		case "dataID":
			rule.dataID = elem.Value
		case "rule":
			rule.rule = elem.Value
		default:
			rule.communal[elem.Key] = elem.Value
		}
	}

	return rule
}

type config struct {
	LogLevel           string   `envconfig:"LOG_LEVEL" default:"debug"`
	QuanxiangInstances []string `envconfig:"QUANXIANG_INSTANCES"`
	Port               string   `envconfig:"PORT" default:"8083"`
}

func main() {
	conf := &config{}
	envconfig.MustProcess("", conf)

	logger := wl.NewLogger(conf.LogLevel)
	s := &ProcessBranch{
		logger: logger,
	}
	ctx := context.Background()
	s.qx = quanxiang.New(conf.QuanxiangInstances, logger)
	node.Main(logger, fmt.Sprintf(":%s", conf.Port))(ctx, s)
}
