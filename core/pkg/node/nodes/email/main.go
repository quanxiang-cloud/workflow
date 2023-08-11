package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	wl "git.yunify.com/quanxiang/workflow/pkg/log"
	"git.yunify.com/quanxiang/workflow/pkg/thirdparty/quanxiang"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/log"
	"github.com/kelseyhightower/envconfig"

	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
	"git.yunify.com/quanxiang/workflow/pkg/node"
)

type Email struct {
	logger log.Logger

	qx quanxiang.QuanXiang
}

const (
	Title       = "title"
	Content     = "content"
	To          = "to"
	ContentType = "content_type"
)

func (e *Email) Do(ctx context.Context, in *node.Request) (*node.Result, error) {
	rule, err := genRule(in.Params)
	if err != nil {
		level.Error(e.logger).Log("message", err)
		return &node.Result{}, err
	}

	var getUser = func(ctx context.Context, userID string) (*quanxiang.UserData, error) {
		user, err := e.qx.GetUserInfo(ctx, &quanxiang.GetUsersInfoRequest{
			ID: userID,
		})
		if err != nil {
			return nil, errors.Wrap(err, "fail get user")
		}
		return user.Data, nil
	}

	var sourceData map[string]interface{}
	if rule.dataID != "" && rule.tableID != "" {
		result, err := e.qx.GetFormData(ctx, &quanxiang.GetFormDataRequest{
			AppID:  rule.appID,
			FormID: rule.tableID,
			DataID: rule.dataID,
		})
		if err != nil {
			level.Error(e.logger).Log("message", err, "appID", rule.appID, "tableID", rule.tableID, "dataID", rule.dataID)
			return &node.Result{}, errors.Wrap(err, "fail get from data")
		}
		sourceData = result.Entity
	}

	email := &quanxiang.Email{}
	for field, data := range sourceData {
		var value string
		if v, ok := data.(string); ok {
			value = v
		} else if vb, err := json.Marshal(data); err == nil {
			value = string(vb)
		} else {
			value = "NULL"
		}
		rule.Title = strings.ReplaceAll(rule.Title, fmt.Sprintf("${%s}", field), value)
		rule.Content = strings.ReplaceAll(rule.Content, fmt.Sprintf("${%s}", field), value)
	}

	email.Title = rule.Title
	email.Content = &quanxiang.Content{
		TemplateID: "quanliang",
		Content:    rule.Content,
	}

	for _, to := range strings.Split(rule.To, ",") {
		switch {
		case strings.HasPrefix(to, "email."):
			email.To = append(email.To, strings.TrimPrefix(to, "email."))
		case strings.HasPrefix(to, "fields."):
			data, ok := sourceData[strings.TrimPrefix(to, "fields.")]
			if ok {
				if v, ok := data.(string); ok {
					email.To = append(email.To, v)
				} else if v, ok := data.([]interface{}); ok {
					for _, elem := range v {
						if v, ok := elem.(map[string]interface{}); ok {
							if v, ok := v["value"]; ok {
								if userID, ok := v.(string); ok {
									user, err := getUser(ctx, userID)
									if err != nil {
										level.Error(e.logger).Log("message", err, "userID", userID)
										continue
									}
									email.To = append(email.To, user.Email)
								}
							}
						}
					}
				}
			}
		case to == "superior":
			userID, ok := sourceData["creator_id"]
			if ok {
				user, err := getUser(ctx, userID.(string))
				if err != nil {
					level.Error(e.logger).Log("message", err, "userID", userID)
				}
				if user != nil {
					for _, leaders := range user.Leader {
						for _, leader := range leaders {
							email.To = append(email.To, leader.Email)
						}
					}
				}
			}
		case to == "processInitiator":
			userID, ok := sourceData["creator_id"]
			if ok {
				user, err := getUser(ctx, userID.(string))
				if err != nil {
					level.Error(e.logger).Log("message", err, "userID", userID)
				}
				email.To = append(email.To, user.Email)
			}
		}
	}

	if len(email.To) != 0 {
		m := new(quanxiang.CreateReq)
		m.Email = email
		_, err = e.qx.SendMessage(ctx, []*quanxiang.CreateReq{m})
		if err != nil {
			return &node.Result{}, errors.Wrap(err, "fail send message")
		}
	}

	return &node.Result{
		Status: v1alpha1.Finish,
	}, nil
}

type rule struct {
	appID   string
	tableID string
	dataID  string

	To      string
	Title   string
	Content string
}

func genRule(in []*v1alpha1.KeyAndValue) (*rule, error) {
	rule := &rule{}
	for _, elem := range in {
		switch elem.Key {
		case "appID":
			rule.appID = elem.Value
		case "tableID":
			rule.tableID = elem.Value
		case "dataID":
			rule.dataID = elem.Value
		case "to":
			rule.To = elem.Value
		case "title":
			rule.Title = elem.Value
		case "content":
			rule.Content = elem.Value
		}
	}

	return rule, nil
}

type config struct {
	LogLevel           string   `envconfig:"LOG_LEVEL" default:"debug"`
	QuanxiangInstances []string `envconfig:"QUANXIANG_INSTANCES"`
	Port               string   `envconfig:"PORT" default:"8081"`
}

func main() {
	conf := &config{}
	envconfig.MustProcess("", conf)

	logger := wl.NewLogger(conf.LogLevel)
	s := &Email{
		logger: logger,
	}
	ctx := context.Background()
	s.qx = quanxiang.New(conf.QuanxiangInstances, logger)
	node.Main(logger, fmt.Sprintf(":%s", conf.Port))(ctx, s)
}
