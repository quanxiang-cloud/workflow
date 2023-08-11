package quanxiang

/*
Copyright 2022 QuanxiangCloud Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
     http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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

	"git.yunify.com/quanxiang/workflow/pkg/errors"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/lb"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/log"
	pkgerrors "github.com/pkg/errors"
)

const (
	sendMessageURI = "/api/v1/message/manager/create/batch"
)

// QuanXiang 消息服务提供
type QuanXiang interface {
	SendMessage(ctx context.Context, req []*CreateReq) (*Resp, error)
	GetFormData(ctx context.Context, req *GetFormDataRequest) (*GetFormDataResponse, error)
	SearchFormDataList(ctx context.Context, req *SearchFormDataListRequest) (*SearchFormDataListResponse, error)
	CreateFormData(ctx context.Context, req *CreateFormDataRequest) (*CreateFormDataResponse, error)
	UpdateFormData(ctx context.Context, req *UpdateFormDataRequest) (*UpdateFormDataResponse, error)
	GetAppInfo(ctx context.Context, req *GetAppInfoRequest) (*GetAppInfoResponse, error)
	GetUserInfo(ctx context.Context, req *GetUsersInfoRequest) (*GetUsersInfoResponse, error)
	GetFormSchema(c context.Context, req *GetFormSchemaRequest) (*GetFormSchemaResponse, error)
}

type message struct {
	client http.Client
}

// CreateReq CreateReq
type CreateReq struct {
	data `json:",omitempty"`
}

type data struct {
	Letter *Letter `json:"letter"`
	Email  *Email  `json:"email"`
	Phone  *Phone  `json:"phone"`
}

// Phone Phone
type Phone struct {
}

// Letter Letter
type Letter struct {
	ID      string   `json:"id,omitempty"`
	UUID    []string `json:"uuid,omitempty"`
	Content *Content `json:"contents"`
}

// Email Email
type Email struct {
	To          []string `json:"To"`
	Title       string   `json:"title"`
	Content     *Content `json:"contents"`
	ContentType string   `json:"content_type,omitempty"`
}

// Content Content
type Content struct {
	Content     string            `json:"content"`
	TemplateID  string            `json:"templateID"`
	KeyAndValue map[string]string `json:"keyAndValue"`
}

// Resp 未知返回体
type Resp struct {
	Code    int64       `json:"code"`
	Message string      `json:"msg,omitempty"`
	Data    interface{} `json:"data"`
}

type Endpoints struct {
	SendMessageEndpoint        endpoint.Endpoint
	GetFormDataEndpoint        endpoint.Endpoint
	SearchFormDataListEndpoint endpoint.Endpoint
	CreateFormDataEndpoint     endpoint.Endpoint
	UpdateFormDataEndpoint     endpoint.Endpoint
	GetAppInfoEndpoint         endpoint.Endpoint
	GetUsersInfoEndpoint       endpoint.Endpoint
	GetFormSchemaEndpoint      endpoint.Endpoint
}

func (e Endpoints) SendMessage(ctx context.Context, in []*CreateReq) (*Resp, error) {
	resp, err := e.SendMessageEndpoint(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp.(*Resp), err
}

func SendMessageEndpoint(e QuanXiang) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.([]*CreateReq)
		resp, err := e.SendMessage(ctx, req)
		return resp, err
	}
}

func New(instance []string, logger log.Logger) QuanXiang {
	var endpoints Endpoints

	var getInstancer = func(instance []string, host string) sd.FixedInstancer {
		if len(instance) != 0 {
			return sd.FixedInstancer(instance)
		}

		return sd.FixedInstancer([]string{host})
	}

	var (
		retryMax     = 30
		retryTimeout = 30 * time.Second
	)

	{
		factory := factoryFor(SendMessageEndpoint)
		endpointer := sd.NewEndpointer(getInstancer(instance, "http://message"), factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.SendMessageEndpoint = retry
	}
	{
		factory := factoryFor(GetFormDataEndpoint)
		endpointer := sd.NewEndpointer(getInstancer(instance, "http://form:8081"), factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.GetFormDataEndpoint = retry
	}
	{
		factory := factoryFor(SearchFormDataListEndpoint)
		endpointer := sd.NewEndpointer(getInstancer(instance, "http://form:8081"), factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.SearchFormDataListEndpoint = retry
	}
	{
		factory := factoryFor(CreateFormDataEndpoint)
		endpointer := sd.NewEndpointer(getInstancer(instance, "http://form:8081"), factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.CreateFormDataEndpoint = retry
	}
	{
		factory := factoryFor(UpdateFormDataEndpoint)
		endpointer := sd.NewEndpointer(getInstancer(instance, "http://form:8081"), factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.UpdateFormDataEndpoint = retry
	}
	{
		factory := factoryFor(GetFormSchemaEndpoint)
		endpointer := sd.NewEndpointer(getInstancer(instance, "http://form:80"), factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.GetFormSchemaEndpoint = retry
	}
	{
		factory := factoryFor(GetAppInfoEndpoint)
		endpointer := sd.NewEndpointer(getInstancer(instance, "http://app-center"), factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.GetAppInfoEndpoint = retry
	}
	{
		factory := factoryFor(GetUsersInfoEndpoint)
		endpointer := sd.NewEndpointer(getInstancer(instance, "http://org"), factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.GetUsersInfoEndpoint = retry
	}

	return endpoints
}

func factoryFor(makeEndpoint func(e QuanXiang) endpoint.Endpoint) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		service, err := NewClientEndPoints(instance)
		if err != nil {
			return nil, nil, err
		}
		return makeEndpoint(service), nil, nil
	}
}

func NewClientEndPoints(instance string) (Endpoints, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}

	var tgt *url.URL
	var err error

	tgt, err = url.Parse(instance)
	if err != nil {
		return Endpoints{}, err
	}
	tgt.Path = ""
	options := []httptransport.ClientOption{}

	return Endpoints{
		SendMessageEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			r.URL.Path = sendMessageURI

			req := request.([]*CreateReq)

			paramByte, err := json.Marshal(req)
			if err != nil {
				return err
			}
			fmt.Println(string(paramByte))
			reader := bytes.NewReader(paramByte)
			r.Header.Set("Content-Type", "application/json")
			r.Body = io.NopCloser(reader)

			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return &Resp{}, errors.NewErr(http.StatusInternalServerError, &errors.CodeError{
					Code:    resp.StatusCode,
					Message: "fail send email",
				})
			}

			r := new(Resp)

			err = json.NewDecoder(resp.Body).Decode(r)
			if err != nil {
				return &Resp{}, pkgerrors.Wrap(err, "fail decode email response")
			}
			return r, err
		}, options...).Endpoint(),
		GetFormDataEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {

			req := request.(*GetFormDataRequest)

			queryMap := map[string]interface{}{
				"query": map[string]interface{}{
					"term": map[string]interface{}{
						"_id": req.DataID,
					},
				},
				"ref": req.Ref,
			}
			r.URL.Path = fmt.Sprintf("%s%s%s%s%s", "/api/v1/form/", req.AppID, "/home/form/", req.FormID, "/get")
			paramByte, err := json.Marshal(queryMap)
			if err != nil {
				return err
			}

			reader := bytes.NewReader(paramByte)
			r.Header.Set("Content-Type", "application/json")
			r.Body = io.NopCloser(reader)

			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return &GetFormDataResponse{}, errors.NewErr(http.StatusInternalServerError, &errors.CodeError{
					Code:    resp.StatusCode,
					Message: "get form data err",
				})
			}

			r := &struct {
				Code int                 `json:"code,omitempty"`
				Data GetFormDataResponse `json:"data,omitempty"`
			}{}

			err = json.NewDecoder(resp.Body).Decode(r)
			if err != nil {
				return &GetFormDataResponse{}, pkgerrors.Wrap(err, "json decode err form data ")
			}
			if r.Code != 0 {
				return &GetFormDataResponse{}, errors.NewErr(http.StatusInternalServerError, &errors.CodeError{
					Code: r.Code,
				})
			}
			return &r.Data, err
		}, options...).Endpoint(),
		SearchFormDataListEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {

			req := request.(*SearchFormDataListRequest)
			req.Method = "find"
			r.URL.Path = fmt.Sprintf("%s%s%s%s%s", "/api/v1/form/", req.AppID, "/home/form/", req.FormID, "/search")
			req.AppID = ""
			req.FormID = ""
			paramByte, err := json.Marshal(req)
			if err != nil {
				return err
			}

			reader := bytes.NewReader(paramByte)
			r.Header.Set("Content-Type", "application/json")
			r.Body = io.NopCloser(reader)

			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return &SearchFormDataListResponse{}, errors.NewErr(http.StatusInternalServerError, &errors.CodeError{
					Code:    resp.StatusCode,
					Message: "search form data err",
				})
			}

			r := new(BaseResp)
			searchFormData := SearchFormData{}
			r.Data = searchFormData

			err = json.NewDecoder(resp.Body).Decode(r)
			if err != nil {
				return &SearchFormDataListResponse{}, pkgerrors.Wrap(err, "json decode err form search form data ")
			}
			err = makeData(r.Data, &searchFormData)
			if err != nil {
				return &SearchFormDataListResponse{}, pkgerrors.Wrap(err, "json decode err form search form data ")
			}
			return &SearchFormDataListResponse{
				Data: searchFormData,
			}, err
		}, options...).Endpoint(),
		CreateFormDataEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {

			req := request.(*CreateFormDataRequest)
			r.URL.Path = fmt.Sprintf("%s%s%s%s%s", "/api/v1/form/", req.AppID, "/home/form/", req.FormID, "/create")
			paramByte, err := json.Marshal(req.Data)
			if err != nil {
				return err
			}

			reader := bytes.NewReader(paramByte)
			r.Header.Set("Content-Type", "application/json")
			r.Body = io.NopCloser(reader)

			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return &CreateFormDataResponse{}, errors.NewErr(http.StatusInternalServerError, &errors.CodeError{
					Code:    resp.StatusCode,
					Message: "create form data err",
				})
			}

			r := new(CreateFormDataResponse)

			err = json.NewDecoder(resp.Body).Decode(r)
			if err != nil {
				return &CreateFormDataResponse{}, pkgerrors.Wrap(err, "json decode create form data response err ")
			}
			return r, err
		}, options...).Endpoint(),
		UpdateFormDataEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			req := request.(*UpdateFormDataRequest)

			if req.Data.Entity == nil {
				return nil
			}
			r.URL.Path = fmt.Sprintf("%s%s%s%s%s", "/api/v1/form/", req.AppID, "/home/form/", req.FormID, "/update")
			paramByte, err := json.Marshal(req.Data)
			if err != nil {
				return err
			}

			reader := bytes.NewReader(paramByte)
			r.Header.Set("Content-Type", "application/json")
			r.Body = io.NopCloser(reader)

			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return &UpdateFormDataResponse{}, errors.NewErr(http.StatusInternalServerError, &errors.CodeError{
					Code:    resp.StatusCode,
					Message: "update form data err",
				})
			}

			r := new(UpdateFormDataResponse)

			err = json.NewDecoder(resp.Body).Decode(r)
			if err != nil {
				return &UpdateFormDataResponse{}, pkgerrors.Wrap(err, "json decode update form data response err ")
			}
			return r, err
		}, options...).Endpoint(),
		GetAppInfoEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			req := request.(*GetAppInfoRequest)

			r.URL.Path = fmt.Sprintf("%s", "/api/v1/app-center/one")
			//todo 这里可能需要改动
			r.Header.Set("Role", "super")
			r.Header.Set("User-Id", req.UserID)
			paramByte, err := json.Marshal(req)
			if err != nil {
				return err
			}

			reader := bytes.NewReader(paramByte)
			r.Header.Set("Content-Type", "application/json")
			r.Body = io.NopCloser(reader)

			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return &GetAppInfoResponse{}, errors.NewErr(http.StatusInternalServerError, &errors.CodeError{
					Code:    resp.StatusCode,
					Message: "get data from appcenter err",
				})
			}

			r := new(Resp)
			appData := &AppData{}
			r.Data = appData
			err = json.NewDecoder(resp.Body).Decode(r)
			if err != nil {
				return &GetAppInfoResponse{}, pkgerrors.Wrap(err, "json decode data what is from appcenter err ")
			}
			err = makeData(r.Data, appData)
			if err != nil {
				return &GetAppInfoResponse{}, pkgerrors.Wrap(err, "json decode data what is from appcenter err ")
			}
			return &GetAppInfoResponse{Data: appData}, err
		}, options...).Endpoint(),
		GetUsersInfoEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			req := request.(*GetUsersInfoRequest)

			r.URL.Path = fmt.Sprintf("%s", "/api/v1/org/o/user/info")
			paramByte, err := json.Marshal(req)
			if err != nil {
				return err
			}

			reader := bytes.NewReader(paramByte)
			r.Header.Set("Content-Type", "application/json")
			r.Body = io.NopCloser(reader)

			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return &GetUsersInfoResponse{}, errors.NewErr(http.StatusInternalServerError, &errors.CodeError{
					Code:    resp.StatusCode,
					Message: "get data from org err",
				})
			}

			r := new(GetUsersInfoResponse)
			userData := UserData{}
			r.Data = &userData

			err = json.NewDecoder(resp.Body).Decode(r)
			if err != nil {
				return &GetUsersInfoResponse{}, pkgerrors.Wrap(err, "json decode data what is from org err ")
			}
			err = makeData(r.Data, &userData)
			if err != nil {
				return &GetUsersInfoResponse{}, pkgerrors.Wrap(err, "json decode data what is from org err ")
			}
			return &GetUsersInfoResponse{
				Data: &userData,
			}, err
		}, options...).Endpoint(),
		GetFormSchemaEndpoint: httptransport.NewClient("POST", tgt, func(ctx context.Context, r *http.Request, request interface{}) error {
			req := request.(*GetFormSchemaRequest)

			r.URL.Path = fmt.Sprintf("%s%s%s%s", "/api/v1/form/", req.AppID, "/home/schema/", req.FormID)

			paramByte, err := json.Marshal(struct {
				TableID string `json:"tableID"`
			}{
				TableID: req.FormID,
			})
			if err != nil {
				return err
			}

			reader := bytes.NewReader(paramByte)
			r.Header.Set("Content-Type", "application/json")
			r.Body = io.NopCloser(reader)

			return nil
		}, func(ctx context.Context, resp *http.Response) (response interface{}, err error) {
			if resp.StatusCode != http.StatusOK {
				return &GetFormSchemaResponse{}, errors.NewErr(http.StatusInternalServerError, &errors.CodeError{
					Code:    resp.StatusCode,
					Message: "get data from schema err",
				})
			}

			r := new(BaseResp)
			respData := GetFormSchemaResponse{}
			r.Data = respData

			err = json.NewDecoder(resp.Body).Decode(r)
			if err != nil {
				return &GetFormSchemaResponse{}, pkgerrors.Wrap(err, "json decode data what is from schema err ")
			}
			err = makeData(r.Data, &respData)
			if err != nil {
				return &GetUsersInfoResponse{}, pkgerrors.Wrap(err, "json decode data what is from schema err ")
			}
			return &respData, err
		}, options...).Endpoint(),
	}, nil
}

// form ----------
type GetFormDataRequest struct {
	AppID  string
	FormID string
	DataID string
	Ref    map[string]interface{}
}
type GetFormDataResponse struct {
	Entity     map[string]interface{} `json:"entity"`
	ErrorCount int64                  `json:"errorCount"`
}

func (e Endpoints) GetFormData(ctx context.Context, in *GetFormDataRequest) (*GetFormDataResponse, error) {
	resp, err := e.GetFormDataEndpoint(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp.(*GetFormDataResponse), err
}

func GetFormDataEndpoint(e QuanXiang) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*GetFormDataRequest)
		resp, err := e.GetFormData(ctx, req)
		return resp, err
	}
}

type BaseResp struct {
	Code    int         `json:"code"`
	Message string      `json:"msg"`
	Data    interface{} `json:"data"`
}
type SearchFormDataListRequest struct {
	AppID  string                 `json:"appID,omitempty"`
	FormID string                 `json:"formID,omitempty"`
	Method string                 `json:"method"`
	Page   int                    `json:"page"`
	Size   int                    `json:"size"`
	Query  map[string]interface{} `json:"query"`
	Sort   []string               `json:"sort"`
}
type SearchFormDataListResponse struct {
	Data SearchFormData `json:"data"`
}
type SearchFormData struct {
	Total    int64                    `json:"total"`
	Entities []map[string]interface{} `json:"entities"`
}

func (e Endpoints) SearchFormDataList(ctx context.Context, in *SearchFormDataListRequest) (*SearchFormDataListResponse, error) {
	resp, err := e.SearchFormDataListEndpoint(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp.(*SearchFormDataListResponse), err
}

func SearchFormDataListEndpoint(e QuanXiang) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*SearchFormDataListRequest)
		resp, err := e.SearchFormDataList(ctx, req)
		return resp, err
	}
}

type RefData struct {
	AppID   string         `json:"appID"`
	TableID string         `json:"tableID"`
	Type    string         `json:"type"`
	New     interface{}    `json:"new"` // []string    or   []CreateEntity
	Deleted []string       `json:"deleted"`
	Updated []UpdateEntity `json:"updated"`
}

type UpdateEntity struct {
	Entity map[string]interface{} `json:"entity"`
	Query  map[string]interface{} `json:"query"`
	Ref    map[string]RefData     `json:"ref"`
}
type CreateFormDataRequest struct {
	AppID  string
	FormID string
	Data   Data
}
type Data struct {
	Entity map[string]interface{}
	Ref    map[string]RefData
}
type CreateFormDataResponse struct {
}

func (e Endpoints) CreateFormData(ctx context.Context, in *CreateFormDataRequest) (*CreateFormDataResponse, error) {
	resp, err := e.CreateFormDataEndpoint(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp.(*CreateFormDataResponse), err
}
func CreateFormDataEndpoint(e QuanXiang) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*CreateFormDataRequest)
		resp, err := e.CreateFormData(ctx, req)
		return resp, err
	}
}

type UpdateFormDataRequest struct {
	AppID  string
	FormID string
	Data   UpdateEntity
}
type UpdateFormDataResponse struct {
}

func (e Endpoints) UpdateFormData(ctx context.Context, in *UpdateFormDataRequest) (*UpdateFormDataResponse, error) {
	resp, err := e.UpdateFormDataEndpoint(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp.(*UpdateFormDataResponse), err
}

func UpdateFormDataEndpoint(e QuanXiang) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*UpdateFormDataRequest)
		resp, err := e.UpdateFormData(ctx, req)
		return resp, err
	}
}

type GetAppInfoRequest struct {
	AppID  string `json:"id,omitempty"`
	UserID string `json:"-"`
}
type GetAppInfoResponse struct {
	Data *AppData `json:"data"`
}

type AppData struct {
	ID          string                 `json:"id"`
	AppName     string                 `json:"appName,omitempty"`
	AccessURL   string                 `json:"accessURL,omitempty"`
	AppIcon     string                 `json:"appIcon,omitempty"`
	CreateBy    string                 `json:"createBy,omitempty"`
	UpdateBy    string                 `json:"updateBy,omitempty"`
	CreateTime  int64                  `json:"createTime,omitempty"`
	UpdateTime  int64                  `json:"updateTime,omitempty"`
	UseStatus   int                    `json:"useStatus,omitempty"` //published:1，unpublished:-1
	Server      int                    `json:"server,omitempty"`
	DelFlag     int64                  `json:"delFlag,omitempty"`
	AppSign     string                 `json:"appSign,omitempty"`
	Extension   map[string]interface{} `json:"extension"`
	Description string                 `json:"description"`
	PerPoly     bool                   `json:"perPoly"`
}

func (e Endpoints) GetAppInfo(ctx context.Context, in *GetAppInfoRequest) (*GetAppInfoResponse, error) {
	resp, err := e.GetAppInfoEndpoint(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp.(*GetAppInfoResponse), err
}
func GetAppInfoEndpoint(e QuanXiang) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*GetAppInfoRequest)
		resp, err := e.GetAppInfo(ctx, req)
		return resp, err
	}
}

type GetUsersInfoRequest struct {
	ID string `json:"id,omitempty"`
}
type GetUsersInfoResponse struct {
	Data *UserData `json:"data"`
}

type UserData struct {
	ID        string `json:"id,omitempty" `
	Name      string `json:"name,omitempty" `
	Phone     string `json:"phone,omitempty" `
	Email     string `json:"email,omitempty" `
	SelfEmail string `json:"selfEmail,omitempty" `
	IDCard    string `json:"idCard,omitempty" `
	Address   string `json:"address,omitempty" `
	//1:normal，-2:invalid，-1:del，2:active,-3:no word
	UseStatus int    `json:"useStatus,omitempty" `
	Position  string `json:"position,omitempty" `
	Avatar    string `json:"avatar,omitempty" `
	JobNumber string `json:"jobNumber,omitempty" `
	//0:null,1:man,2:woman
	Gender int                `json:"gender,omitempty" `
	Source string             `json:"source,omitempty" `
	Dep    [][]DepOneResponse `json:"deps,omitempty"`
	Leader [][]Leader         `json:"leaders,omitempty"`
	// 0x1111 right first 0:need reset password
	Status int `json:"status"`
}
type DepOneResponse struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	UseStatus int    `json:"useStatus,omitempty"`
	PID       string `json:"pid"`
	SuperPID  string `json:"superID,omitempty"`
	Grade     int    `json:"grade,omitempty"`
	//1:company,2:department
	Attr  int              `json:"attr,omitempty"`
	Child []DepOneResponse `json:"child,omitempty"`
}

type Leader struct {
	ID        string `json:"id,omitempty" `
	Name      string `json:"name,omitempty" `
	Phone     string `json:"phone,omitempty" `
	Email     string `json:"email,omitempty" `
	SelfEmail string `json:"selfEmail,omitempty" `
	//1:normal，-2:invalid，-1:del，2:active,-3:no word
	UseStatus int    `json:"useStatus,omitempty" `
	Position  string `json:"position,omitempty" `
	Avatar    string `json:"avatar,omitempty" `
	JobNumber string `json:"jobNumber,omitempty" `
}

func (e Endpoints) GetUserInfo(ctx context.Context, in *GetUsersInfoRequest) (*GetUsersInfoResponse, error) {
	resp, err := e.GetUsersInfoEndpoint(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp.(*GetUsersInfoResponse), err
}
func GetUsersInfoEndpoint(e QuanXiang) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*GetUsersInfoRequest)
		resp, err := e.GetUserInfo(ctx, req)
		return resp, err
	}
}

type GetFormSchemaRequest struct {
	AppID  string
	FormID string
}
type GetFormSchemaResponse struct {
	ID     string                 `json:"id"`
	Schema map[string]interface{} `json:"schema"`
}

func (e Endpoints) GetFormSchema(c context.Context, in *GetFormSchemaRequest) (*GetFormSchemaResponse, error) {
	resp, err := e.GetFormSchemaEndpoint(c, in)
	if err != nil {
		return nil, err
	}
	return resp.(*GetFormSchemaResponse), err
}

func GetFormSchemaEndpoint(e QuanXiang) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*GetFormSchemaRequest)
		resp, err := e.GetFormSchema(ctx, req)
		return resp, err
	}
}

func makeData(src interface{}, entity interface{}) error {
	marshal, err := json.Marshal(src)
	if err != nil {
		return err
	}
	err = json.Unmarshal(marshal, entity)
	if err != nil {
		return err
	}
	return nil
}
