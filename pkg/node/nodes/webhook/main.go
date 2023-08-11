package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	wl "git.yunify.com/quanxiang/workflow/pkg/log"
	"git.yunify.com/quanxiang/workflow/pkg/node"
	"git.yunify.com/quanxiang/workflow/pkg/thirdparty/quanxiang"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	API         string    `json:"sendUrl"`
	ContentType string    `json:"contentType"`
	Outputs     []Outputs `json:"outputs"`
	Method      string    `json:"sendMethod"`
	EditWay     string    `json:"editWay"`
	Inputs      []Inputs  `json:"inputs"`
}

type Outputs struct {
	Type  string      `json:"type"`
	Name  string      `json:"name"`
	Title string      `json:"title"`
	Data  interface{} `json:"data"`
}
type Data struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Title string `json:"title"`
	Data  string `json:"data"`
}
type Inputs struct {
	Type      string      `json:"type"`
	Name      string      `json:"name"`
	Data      interface{} `json:"data"`
	In        string      `json:"in"`
	FieldType string      `json:"fieldType"`
	FieldName string      `json:"fieldName"`
	TableID   string      `json:"tableID"`
}

type WebHook struct {
	logger log.Logger
	qx     quanxiang.QuanXiang
}

const (
	Title       = "title"
	Content     = "content"
	To          = "to"
	ContentType = "content_type"
)

func (w *WebHook) Do(ctx context.Context, in *node.Request) (*node.Result, error) {
	rule, err := genRule(in.Params)
	if err != nil {
		level.Error(w.logger).Log("message", err)
		return &node.Result{}, errors.Wrap(err, "fail get from data")
	}

	var sourceData map[string]interface{}
	if rule.dataID != "" && rule.tableID != "" {
		result, err := w.qx.GetFormData(ctx, &quanxiang.GetFormDataRequest{
			AppID:  rule.appID,
			FormID: rule.tableID,
			DataID: rule.dataID,
		})
		if err != nil {
			level.Error(w.logger).Log("message", err, "appID", rule.appID, "tableID", rule.tableID, "dataID", rule.dataID)
			return &node.Result{}, errors.Wrap(err, "fail get from data")
		}
		sourceData = result.Entity
	}

	var getValue = func(name interface{}, sourceData ...map[string]interface{}) interface{} {
		v, ok := name.(string)
		if !ok {
			return name
		}
		values := strings.Split(v, ".")
		if len(values) == 2 && (strings.HasPrefix(values[0], "$formData") || strings.HasPrefix(values[0], "$variable")) {
			for _, data := range sourceData {
				value, ok := data[strings.ReplaceAll(values[1], " ", "")]
				if ok {
					return value
				}
			}
		}
		return v
	}

	url, err := url.Parse(rule.rule.API)
	if err != nil {
		level.Error(w.logger).Log("message", err, "appID", rule.appID, "tableID", rule.tableID, "dataID", rule.dataID)
		return &node.Result{}, errors.Wrap(err, "fail get from data")
	}

	if url.Host == "localhost" {
		url.Host = "polyapi"
	}

	req := http.Request{
		Method: rule.rule.Method,
		URL:    url,
		Header: make(http.Header),
	}

	body := make(map[string]interface{})
	for _, input := range rule.rule.Inputs {
		if input.Name == "" {
			continue
		}
		value := getValue(input.Data, sourceData, rule.communal)
		switch input.In {
		case "body":
			body[input.Name] = value
		case "header":
			if v, ok := value.(string); ok {
				req.Header.Set(input.Name, v)
			} else {
				bb, err := json.Marshal(value)
				if err == nil {
					req.Header.Set(input.Name, string(bb))
				}
			}
		case "query":
			query := req.URL.Query()
			if v, ok := value.(string); ok {
				query.Set(input.Name, v)
			} else {
				bb, err := json.Marshal(value)
				if err == nil {
					query.Set(input.Name, string(bb))
				}
			}
			req.URL.RawQuery = query.Encode()
		}

	}

	paramByte, err := json.Marshal(body)
	if err != nil {
		return &node.Result{}, err
	}

	reader := bytes.NewReader(paramByte)
	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(reader)

	client := http.Client{
		Timeout: time.Second * 3,
	}
	resp, err := client.Do(&req)
	if err != nil {
		return &node.Result{
			Status: v1alpha1.Finish,
		}, err
	}

	if resp.StatusCode != http.StatusOK {
		return &node.Result{
			Status: v1alpha1.Finish,
		}, errors.NewErr(http.StatusBadRequest, &errors.CodeError{Code: resp.StatusCode})
	}

	defer resp.Body.Close()

	// var decodeBody func(prefix string, body map[string]interface{}) []*v1alpha1.KeyAndValue
	// decodeBody = func(prefix string, body map[string]interface{}) []*v1alpha1.KeyAndValue {
	// 	buf := make([]*v1alpha1.KeyAndValue, 0)

	// 	for name, value := range body {
	// 		if value, ok := value.(map[string]interface{}); ok {
	// 			buf = append(buf, decodeBody(prefix+name+".", value)...)
	// 		} else {
	// 			vb, err := json.Marshal(value)
	// 			if err != nil {
	// 				buf = append(buf, &v1alpha1.KeyAndValue{
	// 					Key:   prefix + name,
	// 					Value: string(vb),
	// 				})
	// 			}

	// 		}
	// 	}

	// 	return buf
	// }

	return &node.Result{
		Status: v1alpha1.Finish,
		// Out:    decodeBody("", body),
	}, nil
}

func genRule(in []*v1alpha1.KeyAndValue) (*rule, error) {
	rule := &rule{
		communal: map[string]interface{}{},
	}
	for _, elem := range in {
		switch elem.Key {
		case "appID":
			rule.appID = elem.Value
		case "tableID":
			rule.tableID = elem.Value
		case "dataID":
			rule.dataID = elem.Value
		case "config":
			err := json.Unmarshal([]byte(elem.Value), &rule.rule)
			if err != nil {
				return nil, errors.Wrap(err, "fail unmashal rule")
			}
		default:
			rule.communal[elem.Key] = elem.Value
		}
	}

	return rule, nil
}

type rule struct {
	appID    string
	tableID  string
	dataID   string
	rule     Config
	communal map[string]interface{}
}

type config struct {
	LogLevel           string   `envconfig:"LOG_LEVEL" default:"debug"`
	QuanxiangInstances []string `envconfig:"QUANXIANG_INSTANCES"`
	Port               string   `envconfig:"PORT" default:"80"`
}

func main() {
	conf := &config{}
	envconfig.MustProcess("", conf)

	logger := wl.NewLogger(conf.LogLevel)
	s := &WebHook{
		logger: logger,
	}
	ctx := context.Background()
	s.qx = quanxiang.New(conf.QuanxiangInstances, logger)
	node.Main(logger, fmt.Sprintf(":%s", conf.Port))(ctx, s)
}
