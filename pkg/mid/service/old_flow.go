package service

import (
	"context"
	"encoding/json"
	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
	"github.com/pkg/errors"
	"strconv"
	"strings"

	"git.yunify.com/quanxiang/workflow/internal/common"
	"git.yunify.com/quanxiang/workflow/pkg/mid/db"
	oldflowmysql "git.yunify.com/quanxiang/workflow/pkg/mid/db/mysql"
	"git.yunify.com/quanxiang/workflow/pkg/node"
	examineservice "git.yunify.com/quanxiang/workflow/pkg/node/nodes/examine/service"
	"git.yunify.com/quanxiang/workflow/pkg/thirdparty/quanxiang"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/log"
	"github.com/quanxiang-cloud/cabin/id"
	"github.com/quanxiang-cloud/cabin/time"
)

type OldFlow interface {
	SaveFlow(ctx context.Context, req *SaveFlowRequest) (*SaveFlowRequest, error)
	UpdateFlow(ctx context.Context, req *UpdateFlowRequest) (*UpdateFlowResponse, error)
	DeleteFlow(ctx context.Context, req *DeleteFlowRequest) (*DeleteFlowResponse, error)
	GetFlowInfo(ctx context.Context, req *GetFlowInfoRequest) (*GetFlowInfoResponse, error)
	GetFlowList(ctx context.Context, req *GetFlowListRequest) (*GetFlowListResponse, error)

	SaveFlowVariable(ctx context.Context, req *SaveFlowVariableRequest) (*SaveFlowVariableResponse, error)
	DeleteFlowVariable(ctx context.Context, req *DeleteFlowVariableRequest) (*DeleteFlowVariableResponse, error)
	GetFlowVariableList(ctx context.Context, req *GetFlowVariableListRequest) (*GetFlowVariableListResponse, error)
	GetFlowVariableListByFlowID(ctx context.Context, req *GetFlowVariableListByFlowIDRequest) (*GetFlowVariableListByFlowIDResponse, error)
	//------------
	GetFormData(ctx context.Context, req *GetFormDataRequest) (*GetFormDataResponse, error)
	GetFlowProcess(ctx context.Context, req *GetFlowProcessRequest) (*GetFlowProcessResponse, error)
	GetFormFieldPermission(ctx context.Context, req *GetFormFieldPermissionRequest) (*GetFormFieldPermissionResponse, error)
	CCToMeList(ctx context.Context, req *CCToMeListRequest) (*CCToMeListResponse, error)
	ExaminedList(ctx context.Context, req *ExaminedListRequest) (*ExaminedListResponse, error)
	PendingExamineList(ctx context.Context, req *PendingExamineListRequest) (*PendingExamineListResponse, error)
	PendingExamineCount(ctx context.Context, req *PendingExamineCountRequest) (*PendingExamineCountResponse, error)
	MyApplyList(ctx context.Context, req *MyApplyListRequest) (*MyApplyListResponse, error)

	Examine(ctx context.Context, req *ExamineRequest) (bool, error)
	Urge(ctx context.Context, req *UrgeRequest) (*UrgeResponse, error)
	Recall(ctx context.Context, req *RecallRequest) (*RecallResponse, error)
	AppExport(ctx context.Context, req *AppExportRequest) (*AppExportResponse, error)
	AppImport(ctx context.Context, req *AppImportRequest) (*AppImportResponse, error)
}

type oldFlow struct {
	node.Endpoints
	examineService examineservice.Task
	quanxiang      quanxiang.QuanXiang
	oldFlowRepo    db.OldFlowRepo
	logger         log.Logger
	oldFlowVarRepo db.OldFlowVarRepo
}

const (
	AppActiveStatus  = "ACTIVE"
	AppSuspendStatus = "SUSPEND"
	AppDeleteStatus  = "DELETE"
)

func NewOldFlow(logger log.Logger, mysqlConf common.Mysql, workFlowInstance, homeHost string) OldFlow {
	newDB, err := oldflowmysql.NewDB(&mysqlConf)
	if err != nil {
		panic(err)
	}
	examineTask := examineservice.NewTask(newDB, nil, logger, workFlowInstance, homeHost)
	quanxiang := quanxiang.New(nil, logger)

	oldFlowRepo := oldflowmysql.NewOldFlow(newDB)
	if err != nil {
		panic(err)
	}
	oldFlowVarRepo := oldflowmysql.NewOldFlowVar(newDB)
	return &oldFlow{
		examineService: examineTask,
		quanxiang:      quanxiang,
		oldFlowRepo:    oldFlowRepo,
		logger:         logger,
		oldFlowVarRepo: oldFlowVarRepo,
	}
}

type BaseModel struct {
	ID string `json:"id"`

	CreatorID  string `json:"creatorId"`
	CreateTime string `json:"createTime"`
	ModifierID string `json:"modifierId"`
	ModifyTime string `json:"modifyTime"`

	CreatorName   string `gorm:"-" json:"creatorName"`
	CreatorAvatar string `gorm:"-" json:"creatorAvatar"`
	ModifierName  string `gorm:"-" json:"modifierName"`
}
type SaveFlowRequest struct {
	BaseModel
	AppID       string `json:"appId" binding:"required"`
	AppStatus   string `json:"appStatus"`
	SourceID    string `json:"sourceId"` // flow initial model id
	Name        string `json:"name" binding:"required"`
	TriggerMode string `json:"triggerMode" binding:"required"` // FORM_DATA|FORM_TIME
	FormID      string `json:"formId"`
	BpmnText    string `json:"bpmnText"`   // flow model json
	Cron        string `json:"cron"`       // if TriggerMode eq FORM_TIME required
	ProcessKey  string `json:"processKey"` // Process key, used to start the process by the id
	Status      string `json:"status"`
	CanCancel   int8   `json:"canCancel"`
	/**
	1:It can only be cancel when the next event is not processed
	2:Any event can be cancel
	3:Cancel under the specified event
	*/
	CanCancelType    int8   `json:"canCancelType"`
	CanCancelNodes   string `json:"canCancelNodes"` // taskDefKey array
	CanUrge          int8   `json:"canUrge"`
	CanViewStatusMsg int8   `json:"canViewStatusMsg"`
	CanMsg           int8   `json:"canMsg"`
	InstanceName     string `json:"instanceName"` // Instance name template
	KeyFields        string `json:"keyFields"`    // Flow key fields
	ProcessID        string `json:"processID"`    // Process id
	UserID           string `json:"-"`
	UserName         string `json:"-"`

	Variables []*Variables `json:"variables"`
}

func (t *oldFlow) SaveFlow(ctx context.Context, req *SaveFlowRequest) (*SaveFlowRequest, error) {

	flow := &db.OldFlow{}
	flow.AppID = req.AppID
	flow.FormID = req.FormID
	flow.CreatedBy = req.UserID
	req.CreatorID = req.UserID
	req.CreatorName = req.UserName
	req.ModifierID = req.UserID
	req.ModifierName = req.UserName
	req.ModifyTime = time.Format(time.NowUnix())
	if req.ID == "" {
		uuid := id.BaseUUID()
		req.ID = uuid
		req.CreateTime = time.Format(time.NowUnix())
		req.CreatorID = req.UserID
		req.CreatorName = req.UserName

		marshal, err := json.Marshal(req)
		if err != nil {
			return &SaveFlowRequest{}, err
		}
		flow.ID = uuid
		flow.Status = "DISABLE"
		flow.FlowJson = string(marshal)

		err = t.oldFlowRepo.Created(ctx, flow)
		if err != nil {
			level.Error(t.logger).Log("message", "save flow", "err", err.Error())
			return &SaveFlowRequest{}, err
		}
		err = t.createSysVars(ctx, req.ID, req.UserID)
		if err != nil {
			return &SaveFlowRequest{}, err
		}
		level.Info(t.logger).Log("message", "save flow success", "id", req.ID)
		return req, err
	}
	get, err := t.oldFlowRepo.Get(ctx, req.ID)
	if err != nil {
		return &SaveFlowRequest{}, err
	}
	if get == nil {
		return &SaveFlowRequest{}, errors.New("save err can not find data from old flow")
	}
	marshal, err := json.Marshal(req)
	if err != nil {
		return &SaveFlowRequest{}, err
	}
	flow.ID = req.ID

	flow.FlowJson = string(marshal)
	err = t.oldFlowRepo.Updated(ctx, flow)
	if err != nil {
		level.Error(t.logger).Log("message", "save update flow", "err", err.Error())
		return &SaveFlowRequest{}, err
	}
	oldFlowVars, err := t.oldFlowVarRepo.GetByFlowID(ctx, req.ID)
	if err != nil {
		level.Error(t.logger).Log("message", "update flow get vars", "err", err.Error())
		return &SaveFlowRequest{}, err
	}
	var creatFlag = true
	for k := range oldFlowVars {
		if oldFlowVars[k].Name == "SYS_AUDIT_BOOL" {
			creatFlag = false
			break
		}
	}
	if creatFlag {
		err = t.createSysVars(ctx, req.ID, req.UserID)
		if err != nil {
			return &SaveFlowRequest{}, err
		}
	}
	level.Info(t.logger).Log("message", "save update flow success", "id", req.ID)
	return req, nil

}

func (t *oldFlow) createSysVars(ctx context.Context, flowID, userID string) error {
	id := id.BaseUUID()
	oldFlowVar := &db.OldFlowVar{
		ID:           id,
		FlowID:       flowID,
		Name:         "SYS_AUDIT_BOOL",
		Type:         "SYSTEM",
		Code:         "flowVar_" + id,
		FieldType:    "SYSTEM",
		DefaultValue: "True",
		Desc:         "系统初始化变量，不可修改",
		CreatedAt:    time.NowUnix(),
		UpdatedAt:    time.NowUnix(),
		CreatorID:    userID,
	}
	err := t.oldFlowVarRepo.Created(ctx, oldFlowVar)
	if err != nil {
		level.Error(t.logger).Log("message", "save flow create var", "err", err.Error())
		return err
	}
	return err
}

type UpdateFlowRequest struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}
type UpdateFlowResponse struct {
	Flag        bool
	TriggerMode string
	AppID       string
	FormID      string
	FlowInfo    *SaveFlowRequest
}

func (t *oldFlow) UpdateFlow(ctx context.Context, req *UpdateFlowRequest) (*UpdateFlowResponse, error) {
	flow, err := t.oldFlowRepo.Get(ctx, req.ID)
	if err != nil {
		level.Error(t.logger).Log("message", "update flow status", "get flow err", err.Error())
		return nil, err
	}
	if flow == nil {
		return nil, errors.New("flow data not exist")
	}
	flow.Status = req.Status
	err = t.oldFlowRepo.UpdatedStatus(ctx, flow)
	if err != nil {
		level.Error(t.logger).Log("message", "update flow status", "err", err.Error())
		return nil, err
	}
	request := SaveFlowRequest{}
	json.Unmarshal([]byte(flow.FlowJson), &request)
	level.Error(t.logger).Log("message", "update flow status success", "id", req.ID)
	return &UpdateFlowResponse{
		Flag:     true,
		FlowInfo: &request,
		AppID:    flow.AppID,
		FormID:   flow.FormID,
	}, nil
}

type DeleteFlowRequest struct {
	ID string //参数在url
}
type DeleteFlowResponse struct {
	bool
}

func (t *oldFlow) DeleteFlow(ctx context.Context, req *DeleteFlowRequest) (*DeleteFlowResponse, error) {
	flow, err := t.oldFlowRepo.Get(ctx, req.ID)
	if err != nil {
		level.Error(t.logger).Log("message", "update flow status", "get flow err", err.Error())
		return nil, err
	}
	if flow == nil {
		return nil, errors.New("flow data not exist")
	}
	if flow.Status == "ENABLE" {
		return nil, errors.New("flow is using,can not delete")
	}
	err = t.oldFlowRepo.Deleted(ctx, flow)
	if err != nil {
		level.Error(t.logger).Log("message", "del flow", "err", err.Error())
		return nil, err
	}
	level.Error(t.logger).Log("message", "del flow success", "id", req.ID)
	return &DeleteFlowResponse{}, nil
}

type GetFlowInfoRequest struct {
	ID     string //参数在url
	UserID string `json:"-"`
}
type GetFlowInfoResponse struct {
	ID         string `json:"id,omitempty"`
	CreatorID  string `json:"creatorId"`
	CreateTime string `json:"createTime"`
	ModifierID string `json:"modifierId"`
	ModifyTime string `json:"modifyTime"`

	CreatorName   string `gorm:"-" json:"creatorName"`
	CreatorAvatar string `gorm:"-" json:"creatorAvatar"`
	ModifierName  string `gorm:"-" json:"modifierName"`
	AppID         string `json:"appId" binding:"required"`
	AppStatus     string `json:"appStatus"`
	SourceID      string `json:"sourceId"` // flow initial model id
	Name          string `json:"name" binding:"required"`
	TriggerMode   string `json:"triggerMode" binding:"required"` // FORM_DATA|FORM_TIME
	FormID        string `json:"formId"`
	BpmnText      string `json:"bpmnText"`   // flow model json
	Cron          string `json:"cron"`       // if TriggerMode eq FORM_TIME required
	ProcessKey    string `json:"processKey"` // Process key, used to start the process by the id
	Status        string `json:"status"`
	CanCancel     int8   `json:"canCancel"`
	/**
	1:It can only be cancel when the next event is not processed
	2:Any event can be cancel
	3:Cancel under the specified event
	*/
	CanCancelType    int8   `json:"canCancelType"`
	CanCancelNodes   string `json:"canCancelNodes"` // taskDefKey array
	CanUrge          int8   `json:"canUrge"`
	CanViewStatusMsg int8   `json:"canViewStatusMsg"`
	CanMsg           int8   `json:"canMsg"`
	InstanceName     string `json:"instanceName"` // Instance name template
	KeyFields        string `json:"keyFields"`    // Flow key fields
	ProcessID        string `json:"processID"`    // Process id

	Variables []*Variables `json:"variables" gorm:"-"`
}
type Variables struct {
	ID            string `json:"id"`
	FlowID        string `json:"flowId"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Code          string `json:"code"`
	FieldType     string `json:"fieldType"`
	Format        string `json:"format"`
	DefaultValue  string `json:"defaultValue"`
	Desc          string `json:"desc"`
	CreateTime    string `json:"createTime"`
	CreatorAvatar string `json:"creatorAvatar"`
	CreatorId     string `json:"creatorId"`
	ModifyTime    string `json:"modifyTime"`
}

func (t *oldFlow) GetFlowInfo(ctx context.Context, req *GetFlowInfoRequest) (*GetFlowInfoResponse, error) {

	//这里还需要对数据进行组装，按照旧流程结构生成数据
	flow, err := t.oldFlowRepo.Get(ctx, req.ID)
	if err != nil {
		level.Error(t.logger).Log("message", "get flow info", "err", err)
		return nil, err
	}
	if flow == nil {
		return nil, nil
	}
	appInfo, err := t.quanxiang.GetAppInfo(ctx, &quanxiang.GetAppInfoRequest{
		AppID:  flow.AppID,
		UserID: req.UserID,
	})
	if err != nil {
		level.Error(t.logger).Log("message", err, "get flow info appID", flow.AppID)
		return &GetFlowInfoResponse{}, err
	}
	if appInfo == nil {
		level.Error(t.logger).Log("message", "get flow info app info is null")
		return &GetFlowInfoResponse{}, nil
	}
	var appStatus = ""
	if appInfo.Data.UseStatus == 1 {
		appStatus = "ACTIVE"
	} else {
		appStatus = "SUSPEND"
	}
	resp := &GetFlowInfoResponse{}
	err = json.Unmarshal([]byte(flow.FlowJson), resp)
	if err != nil {
		return &GetFlowInfoResponse{}, err
	}
	resp.Status = flow.Status
	resp.AppStatus = appStatus
	flowVars, err := t.oldFlowVarRepo.GetByFlowID(ctx, req.ID)
	if err != nil {
		level.Error(t.logger).Log("message", "get flow info vars", "err", err)
		return &GetFlowInfoResponse{}, err
	}
	variables := make([]*Variables, 0)
	for k := range flowVars {
		response := &Variables{}
		response.ID = flowVars[k].ID
		response.FlowID = flowVars[k].FlowID
		response.Code = flowVars[k].Code
		response.DefaultValue = flowVars[k].DefaultValue
		response.Desc = flowVars[k].Desc
		response.FieldType = flowVars[k].FieldType
		response.Name = flowVars[k].Name
		response.Type = flowVars[k].Type
		response.Format = flowVars[k].Format
		response.CreateTime = time.Format(flowVars[k].CreatedAt)
		response.ModifyTime = time.Format(flowVars[k].UpdatedAt)
		response.CreatorId = flowVars[k].CreatorID
		variables = append(variables, response)
	}
	resp.Variables = variables
	return resp, nil
}

type GetFlowListRequest struct {
	Page   int    `json:"page"`
	Size   int    `json:"size"`
	AppID  string `json:"appId"`
	UserID string
}
type GetFlowListResponse struct {
	Total int                   `json:"total"`
	Data  []GetFlowInfoResponse `json:"dataList"`
}

func (t *oldFlow) GetFlowList(ctx context.Context, req *GetFlowListRequest) (*GetFlowListResponse, error) {
	level.Info(t.logger).Log("message", "manager request flow list")
	flows, total, err := t.oldFlowRepo.Search(ctx, req.Page, req.Size, req.AppID)
	if err != nil {
		level.Error(t.logger).Log("message", "get list flow", "err", err)
		return nil, err
	}
	appInfo, err := t.quanxiang.GetAppInfo(ctx, &quanxiang.GetAppInfoRequest{
		AppID:  req.AppID,
		UserID: req.UserID,
	})
	if err != nil {
		level.Error(t.logger).Log("message", err, "appID", req.AppID)
		return &GetFlowListResponse{}, err
	}
	if appInfo == nil {
		level.Error(t.logger).Log("message", "app info is null")
		return &GetFlowListResponse{}, nil
	}
	var appStatus = ""
	if appInfo.Data.UseStatus == 1 {
		appStatus = "ACTIVE"
	} else {
		appStatus = "SUSPEND"
	}
	responses := make([]GetFlowInfoResponse, 0)
	for k := range flows {
		response := GetFlowInfoResponse{}
		err := json.Unmarshal([]byte(flows[k].FlowJson), &response)
		if err != nil {
			level.Error(t.logger).Log("message", "get list flow", "string to struct err", err)
			return &GetFlowListResponse{}, err
		}
		response.AppStatus = appStatus
		response.Status = flows[k].Status
		flowVars, err := t.oldFlowVarRepo.GetByFlowID(ctx, flows[k].ID)
		if err != nil {
			level.Error(t.logger).Log("message", "get flow info vars", "err", err)
			return &GetFlowListResponse{}, err
		}
		variables := make([]*Variables, 0)
		for k1 := range flowVars {
			variable := &Variables{}
			variable.ID = flowVars[k1].ID
			variable.FlowID = flowVars[k1].FlowID
			variable.Code = flowVars[k1].Code
			variable.DefaultValue = flowVars[k1].DefaultValue
			variable.Desc = flowVars[k1].Desc
			variable.FieldType = flowVars[k1].FieldType
			variable.Name = flowVars[k1].Name
			variable.Type = flowVars[k1].Type
			variable.Format = flowVars[k1].Format
			variable.CreateTime = time.Format(flowVars[k1].CreatedAt)
			variable.ModifyTime = time.Format(flowVars[k1].UpdatedAt)
			variable.CreatorId = flowVars[k1].CreatorID
			variables = append(variables, variable)
		}
		response.Variables = variables
		responses = append(responses, response)
	}
	return &GetFlowListResponse{
		Data:  responses,
		Total: total,
	}, nil
}

type SaveFlowVariableRequest struct {
	ID           string      `json:"id"`
	FlowID       string      `json:"flowId"`
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Code         string      `json:"code"`
	FieldType    string      `json:"fieldType"`
	Format       string      `json:"format"`
	DefaultValue interface{} `json:"defaultValue"`
	Desc         string      `json:"desc"`
	UserID       string
}
type SaveFlowVariableResponse struct {
	ID            string `json:"id"`
	FlowID        string `json:"flowId"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Code          string `json:"code"`
	FieldType     string `json:"fieldType"`
	Format        string `json:"format"`
	DefaultValue  string `json:"defaultValue"`
	Desc          string `json:"desc"`
	CreateTime    string `json:"createTime"`
	CreatorAvatar string `json:"creatorAvatar"`
	CreatorId     string `json:"creatorId"`
	ModifyTime    string `json:"modifyTime"`
}

func (t *oldFlow) SaveFlowVariable(ctx context.Context, req *SaveFlowVariableRequest) (*SaveFlowVariableResponse, error) {
	if req.FlowID == "" {
		level.Error(t.logger).Log("message", "save flow var  flowID is null")
		return &SaveFlowVariableResponse{}, errors.New("flow id is null")
	}
	unix := time.NowUnix()
	response := &SaveFlowVariableResponse{}
	if req.ID == "" {

		flowVar := &db.OldFlowVar{}
		flowVar.ID = id.BaseUUID()
		flowVar.FlowID = req.FlowID
		flowVar.Code = "flowVar_" + flowVar.ID
		flowVar.DefaultValue = dealDefaultValueToString(req.DefaultValue)
		flowVar.Desc = req.Desc
		flowVar.FieldType = req.FieldType
		flowVar.Name = req.Name
		flowVar.Type = req.Type
		flowVar.Format = req.Format
		flowVar.CreatedAt = unix
		flowVar.UpdatedAt = unix
		flowVar.CreatorID = req.UserID
		err := t.oldFlowVarRepo.Created(ctx, flowVar)
		if err != nil {
			level.Error(t.logger).Log("message", "create flow var ", "err", err)
			return &SaveFlowVariableResponse{}, err
		}
		level.Info(t.logger).Log("message", "create flow var ok")
		response.ID = flowVar.ID
		response.FlowID = flowVar.FlowID
		response.Code = flowVar.Code
		response.DefaultValue = flowVar.DefaultValue
		response.Desc = flowVar.Desc
		response.FieldType = flowVar.FieldType
		response.Name = flowVar.Name
		response.Type = flowVar.Type
		response.Format = flowVar.Format
		response.CreateTime = time.Format(unix)
		response.ModifyTime = time.Format(unix)
		response.CreatorId = flowVar.CreatorID
		return response, nil
	}
	get, err := t.oldFlowVarRepo.Get(ctx, req.ID)
	if err != nil {
		level.Error(t.logger).Log("message", "get flow var ", "err", err)
		return &SaveFlowVariableResponse{}, err
	}
	if get == nil {
		level.Error(t.logger).Log("message", "get flow var is null")
		return &SaveFlowVariableResponse{}, errors.New("get flow var is null")
	}
	get.Name = req.Name
	get.UpdatedAt = unix
	get.UpdatedID = req.UserID
	get.DefaultValue = dealDefaultValueToString(req.DefaultValue)
	err = t.oldFlowVarRepo.Updated(ctx, get)
	if err != nil {
		level.Error(t.logger).Log("message", "update flow var ", "err", err)
		return &SaveFlowVariableResponse{}, err
	}

	response.ID = get.ID
	response.FlowID = get.FlowID
	response.Code = get.Code
	response.DefaultValue = get.DefaultValue
	response.Desc = get.Desc
	response.FieldType = get.FieldType
	response.Name = get.Name
	response.Type = get.Type
	response.Format = get.Format
	response.CreateTime = time.Format(get.CreatedAt)
	response.ModifyTime = time.Format(unix)
	response.CreatorId = get.CreatorID
	return response, nil
}

func dealDefaultValueToString(value interface{}) string {
	// interface 转 string
	var key string
	if value == nil {
		return ""
	}

	switch value.(type) {
	case float64:
		ft := value.(float64)
		key = strconv.FormatFloat(ft, 'f', -1, 64)
	case float32:
		ft := value.(float32)
		key = strconv.FormatFloat(float64(ft), 'f', -1, 64)
	case int:
		it := value.(int)
		key = strconv.Itoa(it)
	case uint:
		it := value.(uint)
		key = strconv.Itoa(int(it))
	case int8:
		it := value.(int8)
		key = strconv.Itoa(int(it))
	case uint8:
		it := value.(uint8)
		key = strconv.Itoa(int(it))
	case int16:
		it := value.(int16)
		key = strconv.Itoa(int(it))
	case uint16:
		it := value.(uint16)
		key = strconv.Itoa(int(it))
	case int32:
		it := value.(int32)
		key = strconv.Itoa(int(it))
	case uint32:
		it := value.(uint32)
		key = strconv.Itoa(int(it))
	case int64:
		it := value.(int64)
		key = strconv.FormatInt(it, 10)
	case uint64:
		it := value.(uint64)
		key = strconv.FormatUint(it, 10)
	case string:
		key = value.(string)
	case []byte:
		key = string(value.([]byte))
	case bool:
		key = strconv.FormatBool(value.(bool))
	default:
		newValue, _ := json.Marshal(value)
		key = string(newValue)
	}
	return key
}

type DeleteFlowVariableRequest struct {
	ID string //参数在url
}
type DeleteFlowVariableResponse struct {
}

func (t *oldFlow) DeleteFlowVariable(ctx context.Context, req *DeleteFlowVariableRequest) (*DeleteFlowVariableResponse, error) {
	get, err := t.oldFlowVarRepo.Get(ctx, req.ID)
	if err != nil {
		level.Error(t.logger).Log("message", "delete get flow var ", "err", err)
		return &DeleteFlowVariableResponse{}, err
	}
	if get == nil {
		level.Error(t.logger).Log("message", "get flow var is null")
		return &DeleteFlowVariableResponse{}, errors.New("delete  flow var is null")
	}
	err = t.oldFlowVarRepo.Deleted(ctx, req.ID)
	if err != nil {
		level.Error(t.logger).Log("message", "delete flow var", "err", err)
		return &DeleteFlowVariableResponse{}, err
	}
	return &DeleteFlowVariableResponse{}, nil
}

type GetFlowVariableListRequest struct {
	ID string //参数在url
}
type GetFlowVariableListResponse struct {
	Data []struct {
		ID            string `json:"id"`
		FlowID        string `json:"flowId"`
		Name          string `json:"name"`
		Type          string `json:"type"`
		Code          string `json:"code"`
		FieldType     string `json:"fieldType"`
		Format        string `json:"format"`
		DefaultValue  string `json:"defaultValue"`
		Desc          string `json:"desc"`
		CreateTime    string `json:"createTime"`
		CreatorAvatar string `json:"creatorAvatar"`
		CreatorId     string `json:"creatorId"`
		ModifyTime    string `json:"modifyTime"`
	}
}

func (t *oldFlow) GetFlowVariableList(ctx context.Context, req *GetFlowVariableListRequest) (*GetFlowVariableListResponse, error) {
	flowVars, err := t.oldFlowVarRepo.GetByFlowID(ctx, req.ID)
	if err != nil {
		level.Error(t.logger).Log("message", "get flow  vars ", "flowID", req.ID)
		return &GetFlowVariableListResponse{}, err
	}
	response := GetFlowVariableListResponse{}
	for k := range flowVars {
		response.Data = append(response.Data, struct {
			ID            string `json:"id"`
			FlowID        string `json:"flowId"`
			Name          string `json:"name"`
			Type          string `json:"type"`
			Code          string `json:"code"`
			FieldType     string `json:"fieldType"`
			Format        string `json:"format"`
			DefaultValue  string `json:"defaultValue"`
			Desc          string `json:"desc"`
			CreateTime    string `json:"createTime"`
			CreatorAvatar string `json:"creatorAvatar"`
			CreatorId     string `json:"creatorId"`
			ModifyTime    string `json:"modifyTime"`
		}{
			ID:            flowVars[k].ID,
			FlowID:        flowVars[k].FlowID,
			Name:          flowVars[k].Name,
			Type:          flowVars[k].Type,
			Code:          flowVars[k].Code,
			FieldType:     flowVars[k].FieldType,
			Format:        flowVars[k].Format,
			DefaultValue:  flowVars[k].DefaultValue,
			Desc:          flowVars[k].Desc,
			CreateTime:    time.Format(flowVars[k].CreatedAt),
			CreatorAvatar: flowVars[k].CreatorAvatar,
			CreatorId:     flowVars[k].CreatorID,
			ModifyTime:    time.Format(flowVars[k].UpdatedAt),
		})
	}
	return &response, nil
}

type GetFlowVariableListByFlowIDRequest struct {
	FlowID string
}
type GetFlowVariableListByFlowIDResponse struct {
	List []SaveFlowVariableResponse
}

func (t *oldFlow) GetFlowVariableListByFlowID(ctx context.Context, req *GetFlowVariableListByFlowIDRequest) (*GetFlowVariableListByFlowIDResponse, error) {
	vars, err := t.oldFlowVarRepo.GetByFlowID(ctx, req.FlowID)
	if err != nil {
		level.Error(t.logger).Log("message", "get flow vars by flow id", "err", err)
		return &GetFlowVariableListByFlowIDResponse{}, err
	}
	responses := make([]SaveFlowVariableResponse, 0)
	for k := range vars {
		response := SaveFlowVariableResponse{}
		response.ID = vars[k].ID
		response.FlowID = vars[k].FlowID
		response.Code = vars[k].Code
		response.DefaultValue = vars[k].DefaultValue
		response.Desc = vars[k].Desc
		response.FieldType = vars[k].FieldType
		response.Name = vars[k].Name
		response.Type = vars[k].Type
		response.Format = vars[k].Format
		response.CreateTime = time.Format(vars[k].CreatedAt)
		response.ModifyTime = time.Format(vars[k].UpdatedAt)
		response.CreatorId = vars[k].CreatorID
		responses = append(responses, response)
	}
	return &GetFlowVariableListByFlowIDResponse{
		List: responses,
	}, nil
}

// -------------------
type GetFormDataRequest struct {
	ProcessInstanceID string                 `json:"processInstanceID"` //参数在url
	TaskID            string                 `json:"taskID"`            //参数在url
	Ref               map[string]interface{} `json:"ref"`
}
type GetFormDataResponse struct {
	Data map[string]interface{}
}

func (t *oldFlow) GetFormData(ctx context.Context, req *GetFormDataRequest) (*GetFormDataResponse, error) {
	exmaineInfo, err := t.examineService.Get(ctx, &examineservice.GetRequest{
		ID: req.TaskID,
	})
	if err != nil {
		level.Error(t.logger).Log("message", "get form data by flow", "examineID="+req.TaskID+"err", err)
		return &GetFormDataResponse{}, err
	}
	//todo 这里返回的是data，要处理
	formData, err := t.quanxiang.GetFormData(ctx, &quanxiang.GetFormDataRequest{
		AppID:  exmaineInfo.AppID,
		FormID: exmaineInfo.FormTableID,
		DataID: exmaineInfo.FormDataID,
		Ref:    req.Ref,
	})
	return &GetFormDataResponse{
		Data: formData.Entity,
	}, nil
}

type GetFlowProcessRequest struct {
	ProcessInstanceID string //参数在url
	UserID            string
}
type GetFlowProcessResponse struct {
	Data []*InstanceStep
}
type InstanceStep struct {
	ID                string             `json:"id"`
	ProcessInstanceID string             `json:"processInstanceId"`
	TaskID            string             `json:"taskId"`
	TaskType          string             `json:"taskType"` // 节点类型：或签、会签、任填、全填、开始、结束
	TaskDefKey        string             `json:"taskDefKey"`
	TaskName          string             `json:"taskName"`
	HandleUserIDs     string             `json:"handleUserIds"`
	Status            string             `json:"status"` // 步骤处理结果，通过、拒绝、完成填写、已回退、打回重填、自动跳过、自动交给管理员
	NodeInstanceID    string             `json:"nodeInstanceId"`
	OperationRecords  []*OperationRecord `gorm:"-" json:"operationRecords"`
	FlowName          string             `gorm:"-" json:"flowName"`
	Reason            string             `gorm:"-" json:"reason"`
	CreatorID         string             `json:"creatorId"`
	CreateTime        string             `json:"createTime"`
	ModifierID        string             `json:"modifierId"`
	ModifyTime        string             `json:"modifyTime"`

	CreatorName   string `gorm:"-" json:"creatorName"`
	CreatorAvatar string `gorm:"-" json:"creatorAvatar"`
	ModifierName  string `gorm:"-" json:"modifierName"`
}
type OperationRecord struct {
	ID                string `json:"id"`
	ProcessInstanceID string `json:"processInstanceID"`
	InstanceStepID    string `json:"instanceStepID"` // step id
	HandleType        string `json:"handleType"`
	HandleUserID      string `json:"handleUserID"`
	HandleDesc        string `json:"handleDesc"`
	Remark            string `json:"remark"`
	Status            string `json:"status"` // COMPLETED,ACTIVE

	TaskID          string          `json:"taskId"`
	TaskName        string          `json:"taskName"`
	TaskDefKey      string          `json:"taskDefKey"`
	CorrelationData string          `json:"correlationData"`
	HandleTaskModel HandleTaskModel `gorm:"-" json:"handleTaskModel"`
	CurrentNodeType string          `gorm:"-" json:"currentNodeType"`
	RelNodeDefKey   string          `json:"RelNodeDefKey"`
	CreatorID       string          `json:"creatorId"`
	CreateTime      string          `json:"createTime"`
	ModifierID      string          `json:"modifierId"`
	ModifyTime      string          `json:"modifyTime"`

	CreatorName   string `gorm:"-" json:"creatorName"`
	CreatorAvatar string `gorm:"-" json:"creatorAvatar"`
	ModifierName  string `gorm:"-" json:"modifierName"`
}
type HandleTaskModel struct {
	HandleType       string                 `json:"handleType"`
	HandleDesc       string                 `json:"handleDesc"`
	Remark           string                 `json:"remark"`
	TaskDefKey       string                 `json:"taskDefKey"`
	AttachFiles      []AttachFileModel      `json:"attachFiles"`
	HandleUserIDs    []string               `json:"handleUserIds"`
	CorrelationIDs   []string               `json:"correlationIds"`
	FormData         map[string]interface{} `json:"formData"`
	AutoReviewUserID string                 `json:"autoReviewUserId"`
	RelNodeDefKey    string                 `json:"RelNodeDefKey"`
}
type AttachFileModel struct {
	FileName string `json:"fileName"`
	FileURL  string `json:"fileUrl"`
}

func (t *oldFlow) GetFlowProcess(ctx context.Context, req *GetFlowProcessRequest) (*GetFlowProcessResponse, error) {
	task, err := t.examineService.GetByUserIDAndTaskID(ctx, &examineservice.GetByUserIDAndTaskIDRequest{
		TaskID: req.ProcessInstanceID,
		UserID: req.UserID,
	})
	if err != nil {
		level.Error(t.logger).Log("message", "GetFlowProcess", "get task err", err)
		return nil, err
	}
	if task == nil {
		return nil, errors.New("exmiane task not exist")
	}
	flow, err := t.oldFlowRepo.Get(ctx, task.Data.FlowID)
	flowInfoResponse := GetFlowInfoResponse{}
	err = json.Unmarshal([]byte(flow.FlowJson), &flowInfoResponse)
	if err != nil {
		level.Error(t.logger).Log("message", "GetFlowProcess decode flow info", req.ProcessInstanceID, err)
		return nil, err
	}
	examineNodes, err := getExamineNodes(ctx, flowInfoResponse.BpmnText)
	if err != nil {
		level.Error(t.logger).Log("message", "GetFlowProcess", "getExamineNodes err ", err)
		return nil, err
	}
	response := &GetFlowProcessResponse{}
	if len(examineNodes) > 0 {
		formData, err := t.quanxiang.GetFormData(ctx, &quanxiang.GetFormDataRequest{
			AppID:  task.Data.AppID,
			FormID: task.Data.FormTableID,
			DataID: task.Data.FormDataID,
			Ref:    nil,
		})
		if err != nil {
			level.Error(t.logger).Log("message", "GetFlowProcess", "GetFormData err ", err)
			return response, err
		}
		if formData == nil || formData.Entity == nil {
			return nil, errors.New("GetFlowProcess get form data nulll")
		}
		createdBy := formData.Entity["creator_id"].(string)
		applyUser, err := t.quanxiang.GetUserInfo(ctx, &quanxiang.GetUsersInfoRequest{
			ID: createdBy,
		})
		if err != nil {
			level.Error(t.logger).Log("message", "GetFlowProcess ", "GetUserInfo err ", err)
			return response, err
		}
		if applyUser == nil {
			return response, errors.New("GetFlowProcess get user data nulll")
		}
		exmaineTasks, err := t.examineService.GetByFlowID(ctx, &examineservice.GetByFlowIDRequest{
			FlowID: flow.ID,
		})
		if err != nil {
			level.Error(t.logger).Log("message", "GetFlowProcess", "getExamineNodes by flowID err ", err)
			return nil, err
		}
		examMap := make(map[string]*examineservice.ExamineNodeInfo)
		for k := range exmaineTasks.Data {
			examMap[exmaineTasks.Data[k].NodeDefKey] = &exmaineTasks.Data[k]
		}
		for i := len(examineNodes) - 1; i >= 0; i-- {
			for k, v := range examMap {
				instanceStep := &InstanceStep{}
				if k == examineNodes[i].ID {

					instanceStep.TaskDefKey = examineNodes[i].ID
					instanceStep.CreatorID = createdBy
					instanceStep.CreateTime = time.Format(time.NowUnix())
					instanceStep.ModifyTime = time.Format(time.NowUnix())
					instanceStep.Status = "REVIEW"
					instanceStep.TaskName = "审批"
					instanceStep.CreatorID = createdBy
					examineType := examineNodes[i].Data.BusinessData["basicConfig"].(map[string]interface{})["multiplePersonWay"].(string)
					switch examineType {
					case "or":
						instanceStep.TaskType = "OR_APPROVE"
					case "and":
						instanceStep.TaskType = "AND_APPROVE"
					}

					instanceStep.ProcessInstanceID = v.TaskID
					instanceStep.CreateTime = time.Format(v.CreatedAt)
					instanceStep.ModifyTime = time.Format(v.UpdatedAt)
					instanceStep.TaskDefKey = examineNodes[i].ID
					op := &OperationRecord{}
					op.CreatorID = v.UserID
					userInfo, _ := t.quanxiang.GetUserInfo(ctx, &quanxiang.GetUsersInfoRequest{
						ID: v.UserID,
					})
					op.TaskDefKey = examineNodes[i].ID
					op.ID = v.ID
					op.InstanceStepID = v.TaskID
					op.ProcessInstanceID = v.TaskID
					op.TaskID = v.ID
					op.CreatorName = userInfo.Data.Name
					switch v.Result {
					case "agree":
						op.HandleType = "AGREE"
					case "reject":
						op.HandleType = "REFUSE"
					}
					switch v.NodeResult {
					case string(v1alpha1.Finish):
						op.Status = "COMPLETE"
					case string(v1alpha1.Pending):
						op.Status = "ACTIVE"
					}
					op.HandleType = "UNTREATED"
					instanceStep.OperationRecords = append(instanceStep.OperationRecords, op)
					response.Data = append(response.Data, instanceStep)
				} else {
					if approvePersons, ok1 := examineNodes[i].Data.BusinessData["basicConfig"].(map[string]interface{})["approvePersons"].(map[string]interface{}); approvePersons != nil && ok1 {
						switch approvePersons["type"] {
						case "person":
							if users, ok2 := approvePersons["users"].([]interface{}); users != nil && ok2 {
								for k := range users {
									if u, ok3 := users[k].(map[string]interface{}); u != nil && ok3 {
										op := &OperationRecord{}
										op.CreatorID = u["id"].(string)
										op.CreatorName = u["ownerName"].(string)
										op.HandleType = "UNTREATED"
										instanceStep.OperationRecords = append(instanceStep.OperationRecords, op)
									}
								}
							}
							// TODO 这里还需要对字段和变量字段进行查询

							//case "field":
							//	if fields, ok2 := approvePersons["fields"].([]string); fields != nil && ok2 {
							//		for k := range fields {
							//			dealUsers = append(dealUsers, "field."+fields[k])
							//		}
							//	}
							//case "superior":
							//	if fields, ok2 := approvePersons["fields"].([]string); fields != nil && ok2 {
							//		for k := range fields {
							//			dealUsers = append(dealUsers, "leader."+fields[k])
							//		}
							//	}
						}
						response.Data = append(response.Data, instanceStep)
					}
				}

				response.Data = append(response.Data, &InstanceStep{
					TaskType:    "START",
					CreatorID:   createdBy,
					CreatorName: applyUser.Data.Name,
					CreateTime:  time.Format(time.NowUnix()),
					ModifyTime:  time.Format(time.NowUnix()),
					Status:      "SUBMIT",
					FlowName:    flowInfoResponse.Name,
					ID:          id.BaseUUID(),
					OperationRecords: []*OperationRecord{
						{
							ID:           id.BaseUUID(),
							CreatorID:    createdBy,
							CreatorName:  applyUser.Data.Name,
							Status:       "COMPLETE",
							HandleType:   "发起流程",
							HandleUserID: createdBy,
						},
					},
				})
			}

		}
	}
	return response, nil
}

func getExamineNodes(ctx context.Context, flowBpm string) ([]ShapeModel, error) {

	p := &ProcessModel{}
	err := json.Unmarshal([]byte(flowBpm), p)
	if err != nil {
		return nil, err
	}

	node := getFirstNode(p)

	nodeID := make(map[string]struct{})
	exmaineNodes := make([]ShapeModel, 0)

	nodes := []*ShapeModel{getNode(p, node.Data.NodeData.ChildrenID[0])}
DONE:
	for len(nodes) != 0 {
		cp := nodes
		nodes = make([]*ShapeModel, 0)
		for _, node := range cp {
			if _, ok := nodeID[node.ID]; ok {
				continue
			}
			nodeID[node.ID] = struct{}{}

			switch node.Type {
			case "end":
				break DONE
			case "approve":
				exmaineNodes = append(exmaineNodes, *node)

			case "processBranch":
				if rule, ok := node.Data.BusinessData["rule"]; ok {
					if rule, ok := rule.(string); ok {
						if strings.Contains(rule, "true") {
							for _, childID := range node.Data.NodeData.ChildrenID {
								nodes = append(nodes, getNode(p, childID))
							}
						}
					}
				}
			}
			for _, childID := range node.Data.NodeData.ChildrenID {
				nodes = append(nodes, getNode(p, childID))
			}
		}

	}
	return exmaineNodes, err
}

type GetFormFieldPermissionRequest struct {
	ProcessInstanceID string `json:"processInstanceID"` // 参数在url
	TaskID            string `json:"taskId"`
	Type              string `json:"type"` // Type:APPLY_PAGE我发起的,WAIT_HANDLE_PAGE待办,HANDLED_PAGE已办,CC_PAGE抄送,ALL_PAGE全部
	UserID            string `json:"-"`
}
type GetFormFieldPermissionResponse struct {
	FlowName            string             `json:"flowName"`
	CanViewStatusAndMsg bool               `json:"canViewStatusAndMsg"`
	CanMsg              bool               `json:"canMsg"`
	TaskDetailModels    []*TaskDetailModel `json:"taskDetailModels"`
	AppID               string             `json:"appId"`
	TableID             string             `json:"tableId"`
}

// TaskDetailModel task detail
type TaskDetailModel struct {
	TaskID             string      `json:"taskId"`
	TaskName           string      `json:"taskName"`
	TaskType           string      `json:"taskType"` // review、correlation
	TaskDefKey         string      `json:"taskDefKey"`
	FormSchema         interface{} `json:"formSchema"`
	FieldPermission    interface{} `json:"fieldPermission"` // map[string]convert.FieldPermissionModel
	OperatorPermission interface{} `json:"operatorPermission"`
	HasCancelBtn       bool        `json:"hasCancelBtn"`
	HasResubmitBtn     bool        `json:"hasResubmitBtn"`
	HasReadHandleBtn   bool        `json:"hasReadHandleBtn"`
	HasCcHandleBtn     bool        `json:"hasCcHandleBtn"`
	HasUrgeBtn         bool        `json:"hasUrgeBtn"`
}
type OperatorPermissionModel struct {
	Custom []OperatorPermissionItemModel `json:"custom"`
	System []OperatorPermissionItemModel `json:"system"`
}
type OperatorPermissionItemModel struct {
	Enabled        bool   `json:"enabled"`
	Changeable     bool   `json:"changeable"`
	Name           string `json:"name"`
	DefaultText    string `json:"defaultText"`
	Text           string `json:"text"`
	Only           string `json:"only"`
	ReasonRequired bool   `json:"reasonRequired"`
	Value          string `json:"value"`
}

func (t *oldFlow) GetFormFieldPermission(ctx context.Context, req *GetFormFieldPermissionRequest) (*GetFormFieldPermissionResponse, error) {
	examineData, err := t.examineService.GetByUserIDAndTaskID(ctx, &examineservice.GetByUserIDAndTaskIDRequest{
		TaskID: req.ProcessInstanceID,
		UserID: req.UserID,
	})
	if err != nil {
		level.Error(t.logger).Log("message", "get exameine data", req.ProcessInstanceID, err)
		return nil, err
	}
	if examineData != nil && examineData.Data.ID != "" {
		flow, err := t.oldFlowRepo.Get(ctx, examineData.Data.FlowID)
		if err != nil {
			level.Error(t.logger).Log("message", "get flow data by examine flow id", req.ProcessInstanceID, err)
			return nil, err
		}
		if flow == nil {
			return nil, errors.New("flow info no exist")
		}
		flowInfoResponse := GetFlowInfoResponse{}
		err = json.Unmarshal([]byte(flow.FlowJson), &flowInfoResponse)
		if err != nil {
			level.Error(t.logger).Log("message", "decode flow info", req.ProcessInstanceID, err)
			return nil, err
		}
		response := &GetFormFieldPermissionResponse{}
		response.AppID = flow.AppID
		response.CanMsg = flowInfoResponse.CanMsg == 1
		response.CanViewStatusAndMsg = flowInfoResponse.CanViewStatusMsg == 1
		response.FlowName = flowInfoResponse.Name
		taskDetail := &TaskDetailModel{}
		taskDetail.TaskID = examineData.Data.ID
		// FIXME 这里需要动态修改
		taskDetail.TaskName = "审批"
		taskDetail.TaskType = "REVIEW"

		taskDetail.TaskDefKey = examineData.Data.NodeDefKey
		p := &ProcessModel{}
		err = json.Unmarshal([]byte(flowInfoResponse.BpmnText), p)
		if err != nil {
			return nil, err
		}
		node := getNode(p, examineData.Data.NodeDefKey)
		fieldPermissionModel := node.Data.BusinessData["fieldPermission"]
		taskDetail.FieldPermission = fieldPermissionModel
		operatorPermissionObj := node.Data.BusinessData["operatorPermission"]
		operatorPermission, err := json.Marshal(operatorPermissionObj)
		operatorPermissionModel := &OperatorPermissionModel{}
		err = json.Unmarshal(operatorPermission, operatorPermissionModel)
		if err != nil {
			return nil, err
		}
		taskDetail.OperatorPermission = dealOperatorPermission(req.Type, examineData.Data.NodeResult, operatorPermissionModel)
		schema, err := t.quanxiang.GetFormSchema(ctx, &quanxiang.GetFormSchemaRequest{
			AppID:  examineData.Data.AppID,
			FormID: examineData.Data.FormTableID,
		})
		if err != nil {
			return nil, err
		}
		taskDetail.FormSchema = schema.Schema
		response.TaskDetailModels = append(response.TaskDetailModels, taskDetail)
		return response, nil
	}
	return &GetFormFieldPermissionResponse{}, nil
}

func dealOperatorPermission(handType, nodeResult string, model *OperatorPermissionModel) *OperatorPermissionModel {
	//TODO 这里逻辑还需要完善
	switch handType {
	case "HANDLED_PAGE":
		for k := range model.System {
			model.System[k].Enabled = false
		}
	case "WAIT_HANDLE_PAGE":
	case "CC_PAGE":
	case "ALL_PAGE":
	case "APPLY_PAGE":
		if nodeResult == string(v1alpha1.Finish) {
			for k := range model.System {
				model.System[k].Enabled = false
			}
		}
	}
	return model
}

// NodeDataModel info
type NodeDataModel struct {
	Name                  string   `json:"name"`
	BranchTargetElementID string   `json:"branchTargetElementID"`
	ChildrenID            []string `json:"childrenID"`
	BranchID              string   `json:"branchID"`
}

// ShapeDataModel info
type ShapeDataModel struct {
	NodeData     NodeDataModel          `json:"nodeData"`
	BusinessData map[string]interface{} `json:"businessData"`
}

// ShapeModel struct
type ShapeModel struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Data   ShapeDataModel `json:"data"`
	Source string         `json:"source"`
	Target string         `json:"target"`
}

// ProcessModel struct
type ProcessModel struct {
	Version string       `json:"version"`
	Shapes  []ShapeModel `json:"shapes"`
}

func getFirstNode(p *ProcessModel) *ShapeModel {
	for _, elem := range p.Shapes {
		if elem.Type == "formData" {
			return &elem
		}
	}
	return nil
}

func getNode(p *ProcessModel, id string) *ShapeModel {
	for _, elem := range p.Shapes {
		if elem.ID == id {
			return &elem
		}
	}
	return nil
}

type CCToMeListRequest struct {
	ReqPage
	AppID     string `json:"appId"`
	Status    int8   `json:"status"` // status: 1 ，0, -1查全部
	Keyword   string `json:"keyword"`
	OrderType string `json:"orderType"` // orderType: ASC|DESC
}
type ReqPage struct {
	Page   int         `json:"page"`
	Size   int         `json:"size"`
	Orders []OrderItem `json:"orders"`
}
type OrderItem struct {
	Column    string `json:"column"`
	Direction string `json:"direction"` // asc|desc
}
type CCToMeListResponse struct {
	PageSize    int         `json:"-"`
	TotalCount  int64       `json:"total"`
	TotalPage   int         `json:"-"`
	CurrentPage int         `json:"-"`
	StartIndex  int         `json:"-"`
	Data        interface{} `json:"dataList"`
}

func (t *oldFlow) CCToMeList(ctx context.Context, req *CCToMeListRequest) (*CCToMeListResponse, error) {
	return &CCToMeListResponse{}, nil
}

type ExaminedListRequest struct {
	Keyword   string `json:"keyword"`
	OrderType string `json:"orderType"` // orderType: ASC|DESC
	Page      int    `json:"page"`
	Size      int    `json:"size"`
	UserID    string
}
type ExaminedListResponse struct {
	PageSize    int                  `json:"-"`
	TotalCount  int                  `json:"total"`
	TotalPage   int                  `json:"-"`
	CurrentPage int                  `json:"-"`
	StartIndex  int                  `json:"-"`
	Data        []FlowInstanceEntity `json:"dataList"`
}

func (t *oldFlow) ExaminedList(ctx context.Context, req *ExaminedListRequest) (*ExaminedListResponse, error) {
	tasks, err := t.examineService.GetByUserID(ctx, &examineservice.GetByUserIDRequest{
		UserID:     req.UserID,
		NodeResult: string(v1alpha1.Finish),
		Page:       req.Page,
		Limit:      req.Size,
	})
	if err != nil {
		return &ExaminedListResponse{}, err
	}
	//todo 这里要获取表单数据和人员数据，还有表单schema
	response := &ExaminedListResponse{}
	resp := make([]FlowInstanceEntity, 0)
	for k := range tasks.Data {
		formData, err := t.quanxiang.GetFormData(ctx, &quanxiang.GetFormDataRequest{
			AppID:  tasks.Data[k].AppID,
			FormID: tasks.Data[k].FormTableID,
			DataID: tasks.Data[k].FormDataID,
			Ref:    nil,
		})
		if err != nil {
			level.Error(t.logger).Log("message", "ExaminedList get qx form data", "appID:"+tasks.Data[k].AppID+"===FormID:"+tasks.Data[k].FormTableID+"====dataID:"+tasks.Data[k].FormDataID, err)
			return &ExaminedListResponse{}, err
		}
		if formData == nil || len(formData.Entity) == 0 {
			continue
		}
		formSchema, err := t.quanxiang.GetFormSchema(ctx, &quanxiang.GetFormSchemaRequest{
			AppID:  tasks.Data[k].AppID,
			FormID: tasks.Data[k].FormTableID,
		})
		if err != nil {
			level.Error(t.logger).Log("message", "ExaminedList get qx form schema", "appID:"+tasks.Data[k].AppID+"===FormID:"+tasks.Data[k].FormTableID, err)
			return &ExaminedListResponse{}, err
		}
		if formSchema == nil {
			continue
		}
		flow, err := t.oldFlowRepo.Get(ctx, tasks.Data[k].FlowID)
		if err != nil {
			level.Error(t.logger).Log("message", " ExaminedList get flow info err", tasks.Data[k].FlowID, err)
			return &ExaminedListResponse{}, err
		}
		flowInfo := SaveFlowRequest{}
		err = json.Unmarshal([]byte(flow.FlowJson), &flowInfo)
		if err != nil {
			level.Error(t.logger).Log("message", "ExaminedList get flow info decode err", tasks.Data[k].FlowID, err)
			return &ExaminedListResponse{}, err
		}
		appInfo, err := t.quanxiang.GetAppInfo(ctx, &quanxiang.GetAppInfoRequest{
			AppID:  tasks.Data[k].AppID,
			UserID: req.UserID,
		})
		if err != nil {
			return &ExaminedListResponse{}, err
		}
		var appStatus = ""
		if appInfo.Data.UseStatus == 1 {
			appStatus = "ACTIVE"
		} else {
			appStatus = "SUSPEND"
		}

		flowInstanceEntity := FlowInstanceEntity{
			AppID:     tasks.Data[k].AppID,
			AppName:   appInfo.Data.AppName,
			AppStatus: appStatus,
			FormID:    tasks.Data[k].FormTableID,
			FormData:  formData.Entity,
		}
		if v, ok := formData.Entity["creator_id"].(string); v != "" && ok {
			flowInstanceEntity.ApplyUserID = v
			flowInstanceEntity.CreatorID = v
		}
		if v, ok := formData.Entity["creator_name"].(string); v != "" && ok {
			flowInstanceEntity.ApplyUserName = v
			flowInstanceEntity.CreatorName = v
		}
		flowInstanceEntity.FormSchema = formSchema.Schema
		flowInstanceEntity.FormData = formData.Entity
		flowInstanceEntity.ProcessInstanceID = tasks.Data[k].TaskID
		flowInstanceEntity.Name = flowInfo.Name
		flowInstanceEntity.FlowID = flowInfo.ID
		flowInstanceEntity.CreateTime = time.Format(tasks.Data[k].CreatedAt)
		if tasks.Data[k].Result == "agree" {
			flowInstanceEntity.Status = "AGREE"
		}
		if tasks.Data[k].Result == "reject" {
			flowInstanceEntity.Status = "REFUSE"
		}
		split := strings.Split(flowInfo.KeyFields, ",")
		flowInstanceEntity.KeyFields = split
		flowInstanceEntity.ProcessInstanceID = tasks.Data[k].TaskID
		flowInstanceEntity.ID = tasks.Data[k].ID
		resp = append(resp, flowInstanceEntity)
	}
	response.Data = resp
	response.TotalCount = tasks.Total
	return response, nil
}

type PendingExamineListRequest struct {
	HandleType string `json:"handleType"` // handle type : REVIEW, WRITE， READ， OTHER
	Keyword    string `json:"keyword"`
	OrderType  string `json:"orderType"` // orderType: ASC|DESC
	Page       int    `json:"page"`
	Size       int    `json:"size"`
	TagType    string `json:"tagType"` // tag type : OVERTIME, URGE
	UserID     string
}
type PendingExamineListResponse struct {
	PageSize    int                  `json:"-"`
	TotalCount  int                  `json:"total"`
	TotalPage   int                  `json:"-"`
	CurrentPage int                  `json:"-"`
	StartIndex  int                  `json:"-"`
	Data        []PendingExamineData `json:"dataList"`
}
type PendingExamineData struct {
	ID                 string             `json:"id"`
	TaskDefKey         string             `json:"taskDefKey"`
	ProcInstID         string             `json:"procInstId"`
	ActInstID          string             `json:"actInstId"`
	Name               string             `json:"name"`
	Description        string             `json:"description"`
	Owner              string             `json:"owner"`
	Assignee           string             `json:"assignee"`
	StartTime          string             `json:"startTime"`
	EndTime            string             `json:"endTime"`
	Duration           int                `json:"duration"`
	DueDate            string             `json:"dueDate"`
	FlowInstanceEntity FlowInstanceEntity `json:"flowInstanceEntity"`
	UrgeNum            int64              `json:"urgeNum"`
	Handled            string             `json:"handled"`
}
type FlowInstanceEntity struct {
	AppID             string                 `json:"appId"`
	AppName           string                 `json:"appName"`
	AppStatus         string                 `json:"appStatus"`
	ApplyNo           string                 `json:"applyNo"`
	ApplyUserID       string                 `json:"applyUserId"`
	ApplyUserName     string                 `json:"applyUserName"`
	CreateTime        string                 `json:"createTime"`
	CreatorAvatar     string                 `json:"creatorAvatar"`
	CreatorID         string                 `json:"creatorId"`
	CreatorName       string                 `json:"creatorName"`
	FlowID            string                 `json:"flowId"`
	FormData          map[string]interface{} `json:"formData"`
	FormID            string                 `json:"formId"`
	FormInstanceID    string                 `json:"formInstanceId"`
	FormSchema        map[string]interface{} `json:"formSchema"`
	ID                string                 `json:"id"`
	KeyFields         interface{}            `json:"keyFields"`
	ModifierID        string                 `json:"modifierId"`
	ModifierName      string                 `json:"modifierName"`
	ModifyTime        string                 `json:"modifyTime"`
	Name              string                 `json:"name"`
	Nodes             interface{}            `json:"nodes"`
	ProcessInstanceID string                 `json:"processInstanceId"`
	RequestID         string                 `json:"requestID"`
	Status            string                 `json:"status"`
	Tasks             interface{}            `json:"tasks"`
}

func (t *oldFlow) PendingExamineList(ctx context.Context, req *PendingExamineListRequest) (*PendingExamineListResponse, error) {
	tasks, err := t.examineService.GetByUserID(ctx, &examineservice.GetByUserIDRequest{
		UserID:     req.UserID,
		NodeResult: string(v1alpha1.Pending),
		Page:       req.Page,
		Limit:      req.Size,
	})
	if err != nil {
		return &PendingExamineListResponse{}, err
	}
	//todo 这里要获取表单数据和人员数据，还有表单schema
	response := &PendingExamineListResponse{}
	resp := make([]PendingExamineData, 0)
	for k := range tasks.Data {
		formData, err := t.quanxiang.GetFormData(ctx, &quanxiang.GetFormDataRequest{
			AppID:  tasks.Data[k].AppID,
			FormID: tasks.Data[k].FormTableID,
			DataID: tasks.Data[k].FormDataID,
			Ref:    nil,
		})
		if err != nil {
			level.Error(t.logger).Log("message", "get qx form data", "appID:"+tasks.Data[k].AppID+"===FormID:"+tasks.Data[k].FormTableID+"====dataID:"+tasks.Data[k].FormDataID, err)
			return &PendingExamineListResponse{}, err
		}
		if formData == nil {
			continue
		}
		formSchema, err := t.quanxiang.GetFormSchema(ctx, &quanxiang.GetFormSchemaRequest{
			AppID:  tasks.Data[k].AppID,
			FormID: tasks.Data[k].FormTableID,
		})
		if err != nil {
			level.Error(t.logger).Log("message", "get qx form schema", "appID:"+tasks.Data[k].AppID+"===FormID:"+tasks.Data[k].FormTableID, err)
			return &PendingExamineListResponse{}, err
		}
		if formSchema == nil {
			continue
		}
		data := PendingExamineData{}
		data.ID = tasks.Data[k].ID
		data.Name = "审批"
		data.ProcInstID = tasks.Data[k].TaskID
		data.StartTime = time.Format(tasks.Data[k].CreatedAt)
		data.UrgeNum = tasks.Data[k].UrgeTimes
		data.StartTime = time.Format(tasks.Data[k].CreatedAt)
		data.TaskDefKey = tasks.Data[k].NodeDefKey //?
		appInfo, err := t.quanxiang.GetAppInfo(ctx, &quanxiang.GetAppInfoRequest{
			AppID:  tasks.Data[k].AppID,
			UserID: req.UserID,
		})
		if err != nil {
			return &PendingExamineListResponse{}, err
		}
		var appStatus = ""
		if appInfo.Data.UseStatus == 1 {
			appStatus = "ACTIVE"
		} else {
			appStatus = "SUSPEND"
		}
		//FIXME ? handled and FlowInstanceEntity.status 有点疑问
		flow, err := t.oldFlowRepo.Get(ctx, tasks.Data[k].FlowID)
		if err != nil {
			level.Error(t.logger).Log("message", "get flow info err", tasks.Data[k].FlowID, err)
			return &PendingExamineListResponse{}, err
		}
		flowInfo := SaveFlowRequest{}
		err = json.Unmarshal([]byte(flow.FlowJson), &flowInfo)
		if err != nil {
			level.Error(t.logger).Log("message", "get flow info decode err", tasks.Data[k].FlowID, err)
			return &PendingExamineListResponse{}, err
		}
		data.Handled = "ACTIVE"
		data.Description = "REVIEW"
		data.FlowInstanceEntity = FlowInstanceEntity{
			ID:                tasks.Data[k].TaskID,
			AppID:             tasks.Data[k].AppID,
			AppName:           appInfo.Data.AppName,
			AppStatus:         appStatus,
			FormID:            tasks.Data[k].FormTableID,
			FormData:          formData.Entity,
			Status:            "REVIEW",
			FormSchema:        formSchema.Schema,
			ProcessInstanceID: tasks.Data[k].TaskID,
			Name:              flowInfo.Name,
			CreateTime:        time.Format(tasks.Data[k].CreatedAt),
			FlowID:            flowInfo.ID,
		}
		if v, ok := formData.Entity["creator_id"].(string); v != "" && ok {
			data.FlowInstanceEntity.ApplyUserID = v
			data.FlowInstanceEntity.CreatorID = v
		}
		if v, ok := formData.Entity["creator_name"].(string); v != "" && ok {
			data.FlowInstanceEntity.ApplyUserName = v
			data.FlowInstanceEntity.CreatorName = v
		}
		split := strings.Split(flowInfo.KeyFields, ",")
		data.FlowInstanceEntity.KeyFields = split
		resp = append(resp, data)
	}
	response.Data = resp
	response.TotalCount = tasks.Total
	return response, nil
}

type PendingExamineCountRequest struct {
	UserID string //参数在header里面
}
type PendingExamineCountResponse struct {
	CCToMeCount     int `json:"ccToMeCount"`
	OverTimeCount   int `json:"overTimeCount"`
	UrgeCount       int `json:"urgeCount"`
	WaitHandleCount int `json:"waitHandleCount"`
}

func (t *oldFlow) PendingExamineCount(ctx context.Context, req *PendingExamineCountRequest) (*PendingExamineCountResponse, error) {
	response, err := t.examineService.GetByUserID(ctx, &examineservice.GetByUserIDRequest{
		UserID:     req.UserID,
		NodeResult: string(v1alpha1.Pending),
		Page:       1,
		Limit:      999,
	})
	res := &PendingExamineCountResponse{}
	if err != nil {
		level.Error(t.logger).Log("message", "count examine num", "userid:"+req.UserID+" err", err)
		return res, err
	}
	res.WaitHandleCount = response.Total
	return res, nil
}

type MyApplyListRequest struct {
	ReqPage
	Status    string `json:"status"` // handle type : ALL，REVIEW, WRITE， READ， OTHER
	BeginDate string `json:"beginDate"`
	EndDate   string `json:"endDate"`
	Keyword   string `json:"keyword"`
	OrderType string `json:"orderType"` // orderType: ASC|DESC
	UserID    string
}
type MyApplyListResponse struct {
	PageSize    int                  `json:"-"`
	TotalCount  int                  `json:"total"`
	TotalPage   int                  `json:"-"`
	CurrentPage int                  `json:"-"`
	StartIndex  int                  `json:"-"`
	Data        []FlowInstanceEntity `json:"dataList"`
}

func (t *oldFlow) MyApplyList(ctx context.Context, req *MyApplyListRequest) (*MyApplyListResponse, error) {
	tasks, err := t.examineService.GetByCreated(ctx, &examineservice.GetByCreatedRequest{
		UserID: req.UserID,
		Page:   req.Page,
		Limit:  req.Size,
	})
	if err != nil {
		return &MyApplyListResponse{}, err
	}
	var flowEventIsFinish = true
	var flowEventResult = "agree"
	for k := range tasks.Data {
		if tasks.Data[k].NodeResult == string(v1alpha1.Pending) {
			flowEventIsFinish = false
		}
		if tasks.Data[k].Result == examineservice.ResultReject {
			flowEventResult = "reject"
		}
		if tasks.Data[k].Result == examineservice.ResultRecall {
			flowEventResult = "recall"
		}
	}

	response := &MyApplyListResponse{}
	resp := make([]FlowInstanceEntity, 0)
	for k := range tasks.Data {
		formData, err := t.quanxiang.GetFormData(ctx, &quanxiang.GetFormDataRequest{
			AppID:  tasks.Data[k].AppID,
			FormID: tasks.Data[k].FormTableID,
			DataID: tasks.Data[k].FormDataID,
			Ref:    nil,
		})
		if err != nil {
			level.Error(t.logger).Log("message", "ExaminedList get qx form data", "appID:"+tasks.Data[k].AppID+"===FormID:"+tasks.Data[k].FormTableID+"====dataID:"+tasks.Data[k].FormDataID, err)
			return &MyApplyListResponse{}, err
		}
		if formData == nil || len(formData.Entity) == 0 {
			continue
		}
		formSchema, err := t.quanxiang.GetFormSchema(ctx, &quanxiang.GetFormSchemaRequest{
			AppID:  tasks.Data[k].AppID,
			FormID: tasks.Data[k].FormTableID,
		})
		if err != nil {
			level.Error(t.logger).Log("message", "ExaminedList get qx form schema", "appID:"+tasks.Data[k].AppID+"===FormID:"+tasks.Data[k].FormTableID, err)
			return &MyApplyListResponse{}, err
		}
		if formSchema == nil {
			continue
		}
		flow, err := t.oldFlowRepo.Get(ctx, tasks.Data[k].FlowID)
		if err != nil {
			level.Error(t.logger).Log("message", " ExaminedList get flow info err", tasks.Data[k].FlowID, err)
			return &MyApplyListResponse{}, err
		}
		flowInfo := SaveFlowRequest{}
		err = json.Unmarshal([]byte(flow.FlowJson), &flowInfo)
		if err != nil {
			level.Error(t.logger).Log("message", "ExaminedList get flow info decode err", tasks.Data[k].FlowID, err)
			return &MyApplyListResponse{}, err
		}
		appInfo, err := t.quanxiang.GetAppInfo(ctx, &quanxiang.GetAppInfoRequest{
			AppID:  tasks.Data[k].AppID,
			UserID: req.UserID,
		})
		if err != nil {
			return &MyApplyListResponse{}, err
		}
		var appStatus = ""
		if appInfo.Data.UseStatus == 1 {
			appStatus = "ACTIVE"
		} else {
			appStatus = "SUSPEND"
		}

		flowInstanceEntity := FlowInstanceEntity{
			AppID:     tasks.Data[k].AppID,
			AppName:   appInfo.Data.AppName,
			AppStatus: appStatus,
			FormID:    tasks.Data[k].FormTableID,
			FormData:  formData.Entity,
		}
		if v, ok := formData.Entity["creator_id"].(string); v != "" && ok {
			flowInstanceEntity.ApplyUserID = v
			flowInstanceEntity.CreatorID = v
		}
		if v, ok := formData.Entity["creator_name"].(string); v != "" && ok {
			flowInstanceEntity.ApplyUserName = v
			flowInstanceEntity.CreatorName = v
		}
		split := strings.Split(flowInfo.KeyFields, ",")
		flowInstanceEntity.KeyFields = split
		flowInstanceEntity.FormSchema = formSchema.Schema
		flowInstanceEntity.FormData = formData.Entity
		flowInstanceEntity.ProcessInstanceID = tasks.Data[k].TaskID
		flowInstanceEntity.Name = flowInfo.Name
		flowInstanceEntity.FlowID = flowInfo.ID
		flowInstanceEntity.CreateTime = time.Format(tasks.Data[k].CreatedAt)
		if flowEventIsFinish {
			if flowEventResult == "agree" {
				flowInstanceEntity.Status = "AGREE"
			}
			if flowEventResult == "reject" {
				flowInstanceEntity.Status = "REFUSE"
			}
		} else {
			flowInstanceEntity.Status = "REVIEW"
		}

		flowInstanceEntity.ProcessInstanceID = tasks.Data[k].TaskID
		flowInstanceEntity.ID = tasks.Data[k].ID
		resp = append(resp, flowInstanceEntity)
	}
	response.Data = resp
	response.TotalCount = tasks.Total
	return response, nil

}

type ExamineRequest struct {
	UserID            string                   //header里面
	ProcessInstanceID string                   //url里面
	TaskID            string                   //url里面
	HandleType        string                   `json:"handleType"`
	HandleDesc        string                   `json:"handleDesc"`
	Remark            string                   `json:"remark"`
	TaskDefKey        string                   `json:"taskDefKey"`
	AttachFiles       []AttachFileModel        `json:"attachFiles"`
	HandleUserIDs     []string                 `json:"handleUserIds"`
	CorrelationIDs    []string                 `json:"correlationIds"`
	FormData          *examineservice.FormData `json:"formData"`
	AutoReviewUserID  string                   `json:"autoReviewUserId"`
	RelNodeDefKey     string                   `json:"RelNodeDefKey"`
}

func (t *oldFlow) Examine(ctx context.Context, req *ExamineRequest) (bool, error) {

	// todo 值返回bool
	var b = false
	var err error = nil
	switch req.HandleType {
	case "AGREE":
		_, err := t.examineService.Agree(ctx, &examineservice.AgreeRequest{
			UserID:    req.UserID,
			ExamineID: req.TaskID,
			TaskID:    req.ProcessInstanceID,
			ForMData:  req.FormData,
			Remark:    req.Remark,
		})
		if err != nil {
			return b, err
		}
		b = true

	case "REFUSE":
		_, err := t.examineService.Reject(ctx, &examineservice.RejectRequest{
			UserID:    req.UserID,
			ExamineID: req.TaskID,
			TaskID:    req.ProcessInstanceID,
			Remark:    req.Remark,
			ForMData:  req.FormData,
		})
		if err != nil {
			return b, err
		}
		b = true
	}
	return b, err
}

type UrgeRequest struct {
	ProcessInstanceID string `json:"processInstanceID" binding:"required"`
	UserID            string
}
type UrgeResponse struct {
}

func (t *oldFlow) Urge(ctx context.Context, req *UrgeRequest) (*UrgeResponse, error) {
	_, err := t.examineService.Urge(ctx, &examineservice.UrgeRequest{TaskID: req.ProcessInstanceID, UserID: req.UserID})
	return &UrgeResponse{}, err
}

type RecallRequest struct {
	ProcessInstanceID string `json:"processInstanceID"`
}
type RecallResponse struct {
	bool
}

func (t *oldFlow) Recall(ctx context.Context, req *RecallRequest) (*RecallResponse, error) {
	//todo 撤回流程时，如果流程没有完成可以撤回，完成了则不行
	_, err := t.examineService.Recall(ctx, &examineservice.RecallRequest{TaskID: req.ProcessInstanceID})
	return &RecallResponse{}, err
}

type AppExportRequest struct {
	AppID    string `json:"appID"`
	UserID   string `json:"-"`
	UserName string `json:"-"`
}
type AppExportResponse struct {
	Data string
}

func (t *oldFlow) AppExport(ctx context.Context, req *AppExportRequest) (*AppExportResponse, error) {
	flows, _, err := t.oldFlowRepo.Search(ctx, 1, 999, req.AppID)
	if err != nil {
		return nil, errors.Wrap(err, "export flow err")
	}
	saveFlows := make([]SaveFlowRequest, 0)
	for k := range flows {
		flow := SaveFlowRequest{}
		err := json.Unmarshal([]byte(flows[k].FlowJson), &saveFlows)
		if err != nil {
			level.Error(t.logger).Log("message", "export flow get flow vars", "err", err.Error())
			continue
		}

		flowVars, err := t.oldFlowVarRepo.GetByFlowID(ctx, flows[k].ID)
		if err != nil {
			level.Error(t.logger).Log("message", "export flow get flow vars err")
			continue
		}
		variables := make([]*Variables, 0)
		for k1 := range flowVars {
			variables = append(variables, &Variables{
				ID:            flowVars[k1].ID,
				FlowID:        flowVars[k1].FlowID,
				Name:          flowVars[k1].Name,
				Type:          flowVars[k1].Type,
				Code:          flowVars[k1].Code,
				FieldType:     flowVars[k1].FieldType,
				Format:        flowVars[k1].Format,
				DefaultValue:  flowVars[k1].DefaultValue,
				Desc:          flowVars[k1].Desc,
				CreateTime:    time.Format(flowVars[k1].CreatedAt),
				CreatorAvatar: flowVars[k1].CreatorAvatar,
				CreatorId:     flowVars[k1].CreatorID,
				ModifyTime:    time.Format(flowVars[k1].UpdatedAt),
			})
		}
		flow.Variables = variables
	}
	marshal, err := json.Marshal(flows)
	if err != nil {
		return nil, errors.Wrap(err, "expot flow marshal err")
	}

	return &AppExportResponse{
		Data: string(marshal),
	}, nil
}

type AppImportRequest struct {
	AppID string `json:"appID"`
	// FormID map[string]string `json:"formID"`
	Flows    string `json:"flows"`
	UserID   string `json:"-"`
	UserName string `json:"-"`
}
type AppImportResponse struct {
	ImportFlows []SaveFlowRequest
	Data        bool
}

func (t *oldFlow) AppImport(ctx context.Context, req *AppImportRequest) (*AppImportResponse, error) {
	if req.AppID == "" || req.Flows == "" {
		level.Error(t.logger).Log("message", "AppImport appID or flows is null")
		return &AppImportResponse{Data: false}, errors.New("appID or flows is null")
	}
	saveFlowRequests := make([]SaveFlowRequest, 0)
	err := json.Unmarshal([]byte(req.Flows), &saveFlowRequests)
	if err != nil {
		level.Error(t.logger).Log("message", "decode import flow info", "err", err.Error())
		return &AppImportResponse{Data: false}, errors.Wrap(err, "decode import flow info err")
	}
	for k := range saveFlowRequests {
		saveFlowRequests[k].BpmnText = strings.Replace(saveFlowRequests[k].BpmnText, saveFlowRequests[k].AppID, req.AppID, -1)
		saveFlowRequests[k].ID = ""
		saveFlowRequests[k].Status = "DISABLE"
		saveFlowRequests[k].AppID = req.AppID
		saveFlowRequests[k].CreatorID = req.UserID
		saveFlowRequests[k].CreatorName = req.UserName
		saveFlowRequests[k].CreateTime = time.Format(time.NowUnix())
		saveFlowRequests[k].ModifierID = req.UserID
		saveFlowRequests[k].ModifierName = req.UserName
		saveFlowRequests[k].ModifyTime = time.Format(time.NowUnix())
		_, err := t.SaveFlow(ctx, &saveFlowRequests[k])
		if err != nil {
			level.Error(t.logger).Log("message", "decode import flow save", "err", err.Error())
			return nil, errors.Wrap(err, "import flow err ,appID"+req.AppID)
		}
		for k1 := range saveFlowRequests[k].Variables {
			if saveFlowRequests[k].Variables[k1].Type == "CUSTOM" {
				flowVar := &db.OldFlowVar{}
				flowVar.ID = id.BaseUUID()
				flowVar.FlowID = saveFlowRequests[k].ID
				flowVar.Code = "flowVar_" + flowVar.ID
				flowVar.DefaultValue = saveFlowRequests[k].Variables[k1].DefaultValue
				flowVar.Desc = saveFlowRequests[k].Variables[k1].Desc
				flowVar.FieldType = saveFlowRequests[k].Variables[k1].FieldType
				flowVar.Name = saveFlowRequests[k].Variables[k1].Name
				flowVar.Type = saveFlowRequests[k].Variables[k1].Type
				flowVar.Format = saveFlowRequests[k].Variables[k1].Format
				flowVar.CreatedAt = time.NowUnix()
				flowVar.UpdatedAt = time.NowUnix()
				flowVar.CreatorID = req.UserID
				err := t.oldFlowVarRepo.Created(ctx, flowVar)
				if err != nil {
					level.Error(t.logger).Log("message", "decode import flow save vars", "err", err.Error())
					return nil, errors.Wrap(err, "import flow save var err")
				}
			}
		}

	}
	return &AppImportResponse{
		Data:        true,
		ImportFlows: saveFlowRequests,
	}, nil
}
