package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"strconv"
	"strings"

	"git.yunify.com/quanxiang/workflow/apis"
	"git.yunify.com/quanxiang/workflow/pkg/client/clientset/versioned"
	"github.com/go-kit/log/level"

	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
	"git.yunify.com/quanxiang/workflow/pkg/node"
	model "git.yunify.com/quanxiang/workflow/pkg/node/nodes/examine/db"
	"git.yunify.com/quanxiang/workflow/pkg/node/nodes/examine/db/mysql"
	"git.yunify.com/quanxiang/workflow/pkg/thirdparty/quanxiang"
	"github.com/go-kit/log"
	"github.com/quanxiang-cloud/cabin/id"
	"github.com/quanxiang-cloud/cabin/time"
)

type Task interface {
	node.Interface

	Agree(ctx context.Context, req *AgreeRequest) (*AgreeResponse, error)
	Reject(ctx context.Context, req *RejectRequest) (*RejectResponse, error)
	Recall(ctx context.Context, req *RecallRequest) (*RecallResponse, error)
	Transfer(ctx context.Context, req *TransferRequest) (*TransferResponse, error)
	Urge(ctx context.Context, req *UrgeRequest) (*UrgeResponse, error)
	Get(ctx context.Context, req *GetRequest) (*GetResponse, error)
	GetByUserIDAndTaskID(ctx context.Context, req *GetByUserIDAndTaskIDRequest) (*GetByUserIDAndTaskIDResponse, error)
	GetByFlowID(ctx context.Context, req *GetByFlowIDRequest) (*GetByFlowIDResponse, error)
	GetByUserID(ctx context.Context, req *GetByUserIDRequest) (*GetByUserIDResponse, error)
	GetByCreated(ctx context.Context, req *GetByCreatedRequest) (*GetByCreatedResponse, error)
	//List(ctx context.Context, req *ListRequest) (*ListResponse, error)
}

const (
	ResultAgree  = "agree"
	ResultReject = "reject"
	ResultRecall = "recall"
)

type task struct {
	node.Endpoints
	logger     log.Logger
	taskRepo   model.TaskRepo
	qx         quanxiang.QuanXiang
	piplineRun versioned.Client
	homeHost   string
}

func NewTask(db *sql.DB, instance []string, logger log.Logger, workFlowInstance, homeHost string) Task {
	qx := quanxiang.New(instance, logger)

	client := versioned.New(workFlowInstance, logger)

	return &task{
		qx:         qx,
		taskRepo:   mysql.NewExamineNode(db),
		piplineRun: client,
		logger:     logger,
		homeHost:   homeHost,
	}
}

type DoRequest struct {
	TaskID       string
	FlowID       string
	TaskType     string
	UserID       []string
	AppID        string
	FormID       string
	FormDataID   string
	CreatedBy    string
	SysAuditBool string
	NodeDefKey   string
}
type DoResponse struct {
	NodeType string
	Result   string
}

const (
	TaskID       = "database.pipelineRun/id"
	NodeID       = "database.pipelineRunNode/name"
	FlowID       = "flowID"
	TaskType     = "taskType"
	DealUsers    = "dealUsers" // leader.xxxx,field.xxxx,person.xxxx,field.xxxx
	AppID        = "appID"
	FormTableID  = "tableID"
	FormDataID   = "dataID"
	CreatedBy    = "created_by"
	ActionAgree  = "agree"
	ActionReject = "reject"
	SysAuditBool = "SYS_AUDIT_BOOL"
)

const (
	ExamineOr  = "or"
	ExamineAll = "and"
)

func (t *task) Do(ctx context.Context, in *node.Request) (*node.Result, error) {
	res := &node.Result{}
	res.Status = v1alpha1.Pending
	req := new(DoRequest)
	var s = ""

	for k := range in.Params {
		if in.Params[k].Key == AppID {
			req.AppID = in.Params[k].Value
		}
		if in.Params[k].Key == FlowID {
			req.FlowID = in.Params[k].Value
		}
		if in.Params[k].Key == FormTableID {
			req.FormID = in.Params[k].Value
		}
		if in.Params[k].Key == FormDataID {
			req.FormDataID = in.Params[k].Value
		}
		if in.Params[k].Key == CreatedBy {
			req.CreatedBy = in.Params[k].Value
		}
		if in.Params[k].Key == TaskType {
			req.TaskType = in.Params[k].Value
		}
		if in.Params[k].Key == DealUsers {
			s = in.Params[k].Value
		}
		if in.Params[k].Key == SysAuditBool {
			req.SysAuditBool = in.Params[k].Value
		}
	}

	for k := range in.Params {
		if in.Params[k].Key == req.SysAuditBool {
			fmt.Println("上一个审核节点结果=====", in.Params[k].Value)
			if in.Params[k].Value == "false" {

				res.Status = v1alpha1.Finish
				return res, nil
			}
		}
	}
	runID := in.Metadata.Annotations[TaskID]
	nodeID := in.Metadata.Annotations[NodeID]
	req.TaskID = runID
	req.NodeDefKey = nodeID
	level.Info(t.logger).Log("message", "examine do task", "id", req.TaskID, "nodeDefKey", req.NodeDefKey)
	userIDs, createUserID, err := t.resolutionDealObjects(ctx, s, req)
	if err != nil {
		level.Error(t.logger).Log("message", err, "userids", req.UserID)
		res.Status = v1alpha1.Kill
		return res, err
	}
	if len(userIDs) == 0 {
		level.Error(t.logger).Log("message", err, "can not find user ", s)
		res.Status = v1alpha1.Kill
		res.Message = "审核节点解析人员为空" + "，解析字段为" + s
		return res, nil
	}
	req.CreatedBy = createUserID
	level.Info(t.logger).Log("message", "examine do task", "dealUserIDs", userIDs)
	req.UserID = append(req.UserID, userIDs...)
	response, err := t.do(ctx, req)
	if err != nil {
		res.Status = v1alpha1.Kill
		return res, err
	}
	res.Status = v1alpha1.NodeStatus(response.NodeType)
	keyAndValue := &v1alpha1.KeyAndValue{}
	if res.Status == v1alpha1.Finish {
		level.Info(t.logger).Log("message", "examine do task", "id finish", req.TaskID)
		keyAndValue.Key = "agree"
		keyAndValue.Value = response.Result
		res.Out = []*v1alpha1.KeyAndValue{
			keyAndValue,
		}

		var flag bool
		if response.Result == "true" {
			flag = true
		}

		rb, _ := json.Marshal(flag)
		res.Communal = append(res.Communal, &v1alpha1.KeyAndValue{
			Key:   req.SysAuditBool,
			Value: string(rb),
		})
	}

	// 看数据
	marshal, _ := json.Marshal(res)
	fmt.Println(string(marshal))
	return res, nil
}

func (t *task) resolutionDealObjects(ctx context.Context, s string, req *DoRequest) ([]string, string, error) {
	if s == "" {
		return nil, "", errors.New("have no user to deal")
	}
	ctx = context.Background()
	formData, err := t.qx.GetFormData(ctx, &quanxiang.GetFormDataRequest{
		AppID:  req.AppID,
		FormID: req.FormID,
		DataID: req.FormDataID,
		Ref:    nil,
	})
	if err != nil {
		return nil, "", errors.Wrap(err, "get form data err")
	}
	if formData == nil || formData.Entity == nil {
		return nil, "", errors.New("no form data")
	}
	var createUserID = ""
	if creatorID, ok := formData.Entity["creator_id"]; creatorID != nil && ok {
		if v, ok1 := creatorID.(string); v != "" && ok1 {
			createUserID = v
		}

	}
	//todo 这里需要对人员id进行判断，是上级，还是表单字段
	users := strings.Split(s, ",")
	ids := make([]string, 0)
	for k := range users {
		split := strings.Split(users[k], ".")
		if len(split) >= 1 {
			switch split[0] {
			case "leader":
				//todo 朝org要用户信息，拿到leader的id

				if creatorID, ok := formData.Entity["creator_id"]; creatorID != nil && ok {
					if v, ok1 := creatorID.(string); ok1 {
						userInfo, err := t.qx.GetUserInfo(ctx, &quanxiang.GetUsersInfoRequest{
							ID: v,
						})
						if err != nil {
							return nil, "", err
						}
						if userInfo != nil && userInfo.Data.ID != "" {
							if len(userInfo.Data.Leader) > 0 && len(userInfo.Data.Leader[0]) > 0 {
								ids = append(ids, userInfo.Data.Leader[0][0].ID)
							} else {
								ids = append(ids, req.CreatedBy)
							}
						}
					} else {
						ids = append(ids, req.CreatedBy)
					}

				} else {
					ids = append(ids, req.CreatedBy)
				}

			case "field":
				//todo 朝org要用户信息，拿到用户的id

				if len(split) != 2 {
					ids = append(ids, req.CreatedBy)
				} else {
					if v, ok := formData.Entity[split[1]].([]interface{}); v != nil && ok {
						for kf := range v {
							if userFeildMap, ok1 := v[kf].(map[string]interface{}); userFeildMap != nil && ok1 {
								if userID, ok1 := userFeildMap["value"].(string); userID != "" && ok1 {
									ids = append(ids, userID)
								}
							}
						}

					}
				}
			case "person":
				ids = append(ids, split[1])
			case "formApplyUserID":
				if v, ok := formData.Entity["creator_id"].(string); v != "" && ok {
					userInfo, err := t.qx.GetUserInfo(ctx, &quanxiang.GetUsersInfoRequest{
						ID: v,
					})
					if err != nil {
						return nil, "", err
					}
					if userInfo != nil {
						ids = append(ids, userInfo.Data.ID)
					} else {
						ids = append(ids, req.CreatedBy)
					}

				} else {
					ids = append(ids, req.CreatedBy)
				}

			}

		}
	}
	return ids, createUserID, nil
}

func (t *task) do(ctx context.Context, req *DoRequest) (*DoResponse, error) {
	ctx = context.Background()
	res := new(DoResponse)
	tasks, err := t.taskRepo.ListByTaskIDAndNodeDefKey(ctx, req.TaskID, req.NodeDefKey)
	if err != nil {
		return res, err
	}

	if len(tasks) == 0 {
		datas := make([]*model.Task, 0, len(req.UserID))
		for k := range req.UserID {
			datas = append(datas, &model.Task{
				ID:          id.BaseUUID(),
				UserID:      req.UserID[k],
				TaskID:      req.TaskID,
				FlowID:      req.FlowID,
				AppID:       req.AppID,
				FormTableID: req.FormID,
				FormDataID:  req.FormDataID,
				NodeResult:  string(v1alpha1.Pending),
				CreatedAt:   time.NowUnix(),
				ExamineType: req.TaskType,
				CreatedBy:   req.CreatedBy,
				NodeDefKey:  req.NodeDefKey,
			})
		}
		res.NodeType = string(v1alpha1.Pending)
		err := t.taskRepo.InsertBranch(ctx, datas...)
		if err != nil {
			return res, err
		}
		////todo 发送邮件提醒待审核的人
		//for k := range datas {
		//	t.sendEmail(ctx, req.UserID, req.TaskID, datas[k].ID)
		//}

		return res, nil
	}
	var flag = true

	for k := range tasks {
		if tasks[k].NodeResult == string(v1alpha1.Pending) {
			flag = false
			break
		}
	}
	var result = "true"
	if flag {

		//FIXME 考虑后期事物介入问题，在修改NodeResult的时候要注意
		for k := range tasks {
			if tasks[k].Result == ActionReject || tasks[k].Result == ResultRecall {
				result = "false"
				break
			}
		}
	}

	if flag {
		res.NodeType = string(v1alpha1.Finish)
		res.Result = result
	} else {
		res.NodeType = string(v1alpha1.Pending)
	}
	return res, nil

}

type ExamineRequest struct {
	TaskID  string
	UserID  string
	forData map[string]interface{}
	Result  string
	Remark  string
}
type ExamineResponse struct {
}

func (t *task) Examine(ctx context.Context, req *ExamineRequest) (*ExamineResponse, error) {
	//var err error = nil
	//switch req.Result {
	//case ActionAgree:
	//	err = t.examineTask(ctx, req.UserID, req.TaskID, ResultAgree, req.Remark)
	//	//todo 这里可能需要进行表单数据的更新
	//case ActionReject:
	//	err = t.examineTask(ctx, req.UserID, req.TaskID, ResultReject, req.Remark)
	//}
	//if err != nil {
	//	return nil, err
	//}

	return &ExamineResponse{}, nil

}

type AgreeRequest struct {
	ExamineID string
	TaskID    string
	UserID    string
	Remark    string
	ForMData  *FormData
}
type FormData struct {
	Entity map[string]interface{}       `json:"entity"`
	Ref    map[string]quanxiang.RefData `json:"ref"`
}

type AgreeResponse struct {
}

func (t *task) Agree(ctx context.Context, req *AgreeRequest) (*AgreeResponse, error) {
	err := t.examineTask(ctx, req.ExamineID, ResultAgree, req.Remark)
	if err != nil {
		return &AgreeResponse{}, err
	}
	runID, err := strconv.ParseInt(req.TaskID, 10, 64)
	if err != nil {
		return &AgreeResponse{}, err
	}
	//todo 这里可能需要进行表单数据的更新
	if req.ForMData != nil {
		data, _ := t.taskRepo.GetByID(ctx, req.ExamineID)
		_, err = t.qx.UpdateFormData(ctx, &quanxiang.UpdateFormDataRequest{
			AppID:  data.AppID,
			FormID: data.FormTableID,
			Data: quanxiang.UpdateEntity{
				Query: map[string]interface{}{
					"term": map[string]interface{}{
						"_id": data.FormDataID,
					},
				},
				Entity: req.ForMData.Entity,
				Ref:    req.ForMData.Ref,
			},
		})
		if err != nil {
			level.Error(t.logger).Log("message", "examine update form data  ", "err", err.Error())
			return &AgreeResponse{}, err
		}
		level.Error(t.logger).Log("message", "examine update form data  ok")
	}

	err = t.piplineRun.ExecPipelineRun(ctx, &apis.ExecPipelineRun{
		ID: runID,
	})
	if err != nil {
		level.Error(t.logger).Log("message", "examine exec pipline run ", "err", err.Error())
		return &AgreeResponse{}, err
	}
	level.Info(t.logger).Log("message", "examine exec pipline run ok", "run id", runID)

	return &AgreeResponse{}, nil
}

type RejectRequest struct {
	ExamineID string
	TaskID    string
	UserID    string
	Remark    string
	ForMData  *FormData
}
type RejectResponse struct {
}

func (t *task) Reject(ctx context.Context, req *RejectRequest) (*RejectResponse, error) {
	err := t.examineTask(ctx, req.ExamineID, ResultReject, req.Remark)
	if err != nil {
		return nil, err
	}
	//todo 这里可能需要进行表单数据的更新
	if req.ForMData != nil && req.ForMData.Entity != nil {
		data, _ := t.taskRepo.GetByID(ctx, req.ExamineID)
		_, err = t.qx.UpdateFormData(ctx, &quanxiang.UpdateFormDataRequest{
			AppID:  data.AppID,
			FormID: data.FormTableID,
			Data: quanxiang.UpdateEntity{
				Query: map[string]interface{}{
					"term": map[string]interface{}{
						"_id": data.FormDataID,
					},
				},
				Entity: req.ForMData.Entity,
				Ref:    req.ForMData.Ref,
			},
		})
		if err != nil {
			level.Error(t.logger).Log("message", "examine update form data  ", "err", err.Error())
			return nil, err
		}
		level.Error(t.logger).Log("message", "examine update form data  ok")
	}
	runID, err := strconv.ParseInt(req.TaskID, 10, 64)
	if err != nil {
		return nil, err
	}
	err = t.piplineRun.ExecPipelineRun(ctx, &apis.ExecPipelineRun{
		ID: runID,
	})
	if err != nil {
		level.Error(t.logger).Log("message", "examine exec pipline run ", "err", err.Error())
		return nil, err
	}
	level.Info(t.logger).Log("message", "examine exec pipline run ok", "run id", runID)
	return &RejectResponse{}, nil
}

type RecallRequest struct {
	ID     string
	TaskID string
	UserID string
}
type RecallResponse struct {
}

func (t *task) Recall(ctx context.Context, req *RecallRequest) (*RecallResponse, error) {
	tasks, err := t.taskRepo.ListByTaskID(ctx, req.TaskID)
	if err != nil {
		return nil, err
	}
	if len(tasks) > 0 {
		aboutTask := &model.Task{
			TaskID:     req.TaskID,
			Result:     ResultRecall,
			NodeResult: string(v1alpha1.Finish),
		}

		err = t.taskRepo.UpdateByTaskID(ctx, aboutTask)
		if err != nil {
			return nil, err
		}
	}

	return &RecallResponse{}, nil
}

type TransferRequest struct {
	UserID     string
	TaskID     string
	Substitute string
}
type TransferResponse struct {
}

func (t *task) Transfer(ctx context.Context, req *TransferRequest) (*TransferResponse, error) {
	task, err := t.taskRepo.GetByUserIDAndTaskID(ctx, req.UserID, req.TaskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("当前任务与审核人不匹配")
	}
	task.UpdatedAt = time.NowUnix()
	task.Substitute = req.Substitute
	err = t.taskRepo.UpdateSubstitute(ctx, task)
	if err != nil {
		return nil, err
	}
	return &TransferResponse{}, nil
}

func (t *task) examineTask(ctx context.Context, examineID, result, remark string) error {
	task, err := t.taskRepo.GetByID(ctx, examineID)
	if err != nil {
		return err
	}
	if task == nil {
		return errors.New("当前任务与审核人不匹配")
	}
	if task.NodeResult == string(v1alpha1.Finish) {
		return errors.New("任务已经被执行完成，具体请看任务进度详情")
	}
	task.Result = result
	task.Remark = remark
	task.UpdatedAt = time.NowUnix()
	task.NodeResult = string(v1alpha1.Finish)
	err = t.taskRepo.UpdateResult(ctx, task)
	if err != nil {
		return err
	}
	aboutTask := &model.Task{
		TaskID: task.TaskID,
		Result: result,
	}
	switch task.ExamineType {
	case ExamineAll:
		if result == ActionReject {
			aboutTask.NodeResult = string(v1alpha1.Finish)
		}
	case ExamineOr:
		aboutTask.NodeResult = string(v1alpha1.Finish)
	}
	if aboutTask.NodeResult != "" {
		err = t.taskRepo.UpdateByTaskID(ctx, aboutTask)
		if err != nil {
			return err
		}
	}
	return nil
}

type ListRequest struct {
	UserID         string
	CreatedBy      string
	TaskID         string
	NodeType       string
	Result         string
	Page           int
	Limit          int
	FlowInStanceID string
}
type ListResponse struct {
	Total int64        `json:"total,omitempty"`
	Tasks []model.Task `json:"tasks,omitempty" json:"tasks,omitempty"`
}

//TODO 需要重写
//func (t *task) List(ctx context.Context, req *ListRequest) (*ListResponse, error) {
//	list, total, err := t.taskRepo.List(ctx, t.db, req.DealUsers, req.CreatedBy, req.FlowInStanceID, req.TaskID, req.Result, req.NodeType, req.Page, req.Limit)
//	res := &ListResponse{
//		Total: total,
//		Tasks: list,
//	}
//	return res, err
//}

type UrgeRequest struct {
	UserID   string
	TaskID   string
	NodeType string
	Result   string
	Page     int
	Limit    int
}
type UrgeResponse struct {
}

func (t *task) Urge(ctx context.Context, req *UrgeRequest) (*UrgeResponse, error) {
	task, err := t.taskRepo.GetByUserIDAndTaskID(ctx, req.UserID, req.TaskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errors.New("当前任务与审核人不匹配")
	}
	task.UpdatedAt = time.NowUnix()
	task.UrgeTimes = task.UrgeTimes + 1
	err = t.taskRepo.UpdateUrgeTimes(ctx, task)
	if err != nil {
		return nil, err
	}
	return &UrgeResponse{}, nil
}

type GetRequest struct {
	ID string
}
type GetResponse struct {
	ID          string
	TaskID      string
	UserID      string
	Substitute  string //替代审核人的id
	CreatedBy   string //发起人id
	ExamineType string //单人或或签审批：or,多人会签：and
	Result      string //agree｜reject
	NodeResult  string //Pending｜Finish｜
	CreatedAt   int64
	UpdatedAt   int64
	AppID       string
	FormTableID string
	FormDataID  string
	FormRef     string
	UrgeTimes   int64 //催办次数
	Remark      string
	FlowID      string
	NodeDefKey  string
}

func (t *task) Get(ctx context.Context, req *GetRequest) (*GetResponse, error) {
	res, err := t.taskRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return &GetResponse{
		ID:          res.ID,
		TaskID:      res.TaskID,
		UserID:      res.UserID,
		Substitute:  res.Substitute,
		CreatedBy:   res.CreatedBy,
		ExamineType: res.ExamineType,
		Result:      res.Result,
		NodeResult:  res.NodeResult,
		CreatedAt:   res.CreatedAt,
		UpdatedAt:   res.UpdatedAt,
		AppID:       res.AppID,
		FormTableID: res.FormTableID,
		FormDataID:  res.FormDataID,
		FormRef:     res.FormRef,
		UrgeTimes:   res.UrgeTimes,
		Remark:      res.Remark,
		FlowID:      res.FlowID,
		NodeDefKey:  res.NodeDefKey,
	}, nil
}

type GetByUserIDRequest struct {
	UserID     string
	NodeResult string
	Page       int
	Limit      int
}
type GetByUserIDResponse struct {
	Total int
	Data  []ExamineNodeInfo
}

type ExamineNodeInfo struct {
	ID          string
	TaskID      string
	UserID      string
	Substitute  string //替代审核人的id
	CreatedBy   string //发起人id
	ExamineType string //单人或或签审批：or,多人会签：and
	Result      string //agree｜reject
	NodeResult  string //Pending｜Finish｜
	CreatedAt   int64
	UpdatedAt   int64
	AppID       string
	FormTableID string
	FormDataID  string
	FormRef     string
	UrgeTimes   int64 //催办次数
	Remark      string
	FlowID      string
	NodeDefKey  string
}

func (t *task) GetByUserID(ctx context.Context, req *GetByUserIDRequest) (*GetByUserIDResponse, error) {
	list, total, err := t.taskRepo.GetByUserID(ctx, req.UserID, req.NodeResult, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}
	response := &GetByUserIDResponse{}
	for k := range list {
		response.Data = append(response.Data, ExamineNodeInfo{
			ID:          list[k].ID,
			TaskID:      list[k].TaskID,
			UserID:      list[k].UserID,
			Substitute:  list[k].Substitute,
			CreatedBy:   list[k].CreatedBy,
			ExamineType: list[k].ExamineType,
			Result:      list[k].Result,
			NodeResult:  list[k].NodeResult,
			CreatedAt:   list[k].CreatedAt,
			UpdatedAt:   list[k].UpdatedAt,
			AppID:       list[k].AppID,
			FormTableID: list[k].FormTableID,
			FormDataID:  list[k].FormDataID,
			FormRef:     list[k].FormRef,
			UrgeTimes:   list[k].UrgeTimes,
			Remark:      list[k].Remark,
			FlowID:      list[k].FlowID,
			NodeDefKey:  list[k].NodeDefKey,
		})
	}
	response.Total = total
	return response, nil
}

type GetByCreatedRequest struct {
	UserID string
	Page   int
	Limit  int
}
type GetByCreatedResponse struct {
	Total int
	Data  []ExamineNodeInfo
}

func (t *task) GetByCreated(ctx context.Context, req *GetByCreatedRequest) (*GetByCreatedResponse, error) {
	list, total, err := t.taskRepo.GetByCreated(ctx, req.UserID, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}
	response := &GetByCreatedResponse{}
	for k := range list {
		response.Data = append(response.Data, ExamineNodeInfo{
			ID:          list[k].ID,
			TaskID:      list[k].TaskID,
			UserID:      list[k].UserID,
			Substitute:  list[k].Substitute,
			CreatedBy:   list[k].CreatedBy,
			ExamineType: list[k].ExamineType,
			Result:      list[k].Result,
			NodeResult:  list[k].NodeResult,
			CreatedAt:   list[k].CreatedAt,
			UpdatedAt:   list[k].UpdatedAt,
			AppID:       list[k].AppID,
			FormTableID: list[k].FormTableID,
			FormDataID:  list[k].FormDataID,
			FormRef:     list[k].FormRef,
			UrgeTimes:   list[k].UrgeTimes,
			Remark:      list[k].Remark,
			FlowID:      list[k].FlowID,
			NodeDefKey:  list[k].NodeDefKey,
		})
	}
	response.Total = total
	return response, nil
}

type GetByUserIDAndTaskIDRequest struct {
	TaskID string
	UserID string
}
type GetByUserIDAndTaskIDResponse struct {
	Data *ExamineNodeInfo `json:"data"`
}

func (t *task) GetByUserIDAndTaskID(ctx context.Context, req *GetByUserIDAndTaskIDRequest) (*GetByUserIDAndTaskIDResponse, error) {
	data, err := t.taskRepo.GetByUserIDAndTaskID(ctx, req.UserID, req.TaskID)
	if err != nil {
		return nil, err
	}
	response := &GetByUserIDAndTaskIDResponse{}
	response.Data = &ExamineNodeInfo{
		ID:          data.ID,
		TaskID:      data.TaskID,
		UserID:      data.UserID,
		Substitute:  data.Substitute,
		CreatedBy:   data.CreatedBy,
		ExamineType: data.ExamineType,
		Result:      data.Result,
		NodeResult:  data.NodeResult,
		CreatedAt:   data.CreatedAt,
		UpdatedAt:   data.UpdatedAt,
		AppID:       data.AppID,
		FormTableID: data.FormTableID,
		FormDataID:  data.FormDataID,
		FormRef:     data.FormRef,
		UrgeTimes:   data.UrgeTimes,
		Remark:      data.Remark,
		FlowID:      data.FlowID,
		NodeDefKey:  data.NodeDefKey,
	}
	return response, nil
}

type GetByFlowIDRequest struct {
	TaskID string
	FlowID string
}

type GetByFlowIDResponse struct {
	Data []ExamineNodeInfo
}

func (t *task) GetByFlowID(ctx context.Context, req *GetByFlowIDRequest) (*GetByFlowIDResponse, error) {
	tasks, err := t.taskRepo.ListByFlowID(ctx, req.FlowID)
	if err != nil {
		return nil, err
	}
	response := &GetByFlowIDResponse{}
	for k := range tasks {
		resp := ExamineNodeInfo{
			ID:          tasks[k].ID,
			TaskID:      tasks[k].TaskID,
			UserID:      tasks[k].UserID,
			Substitute:  tasks[k].Substitute,
			CreatedBy:   tasks[k].CreatedBy,
			ExamineType: tasks[k].ExamineType,
			Result:      tasks[k].Result,
			NodeResult:  tasks[k].NodeResult,
			CreatedAt:   tasks[k].CreatedAt,
			UpdatedAt:   tasks[k].UpdatedAt,
			AppID:       tasks[k].AppID,
			FormTableID: tasks[k].FormTableID,
			FormDataID:  tasks[k].FormDataID,
			FormRef:     tasks[k].FormRef,
			UrgeTimes:   tasks[k].UrgeTimes,
			Remark:      tasks[k].Remark,
			FlowID:      tasks[k].FlowID,
			NodeDefKey:  tasks[k].NodeDefKey,
		}
		response.Data = append(response.Data, resp)
	}
	return response, nil
}
func (t *task) sendEmail(ctx context.Context, userIDs []string, runID, taskID string) {
	ctx = context.Background()
	sendReqs := make([]*quanxiang.CreateReq, 0)
	for k := range userIDs {
		usersInfo, err := t.qx.GetUserInfo(ctx, &quanxiang.GetUsersInfoRequest{
			ID: userIDs[k],
		})
		if err != nil {
			level.Error(t.logger).Log("message", "examine do send message", "userID", userIDs[k], "err", err)
			continue
		}
		if usersInfo != nil && usersInfo.Data.ID != "" {
			m := new(quanxiang.CreateReq)
			email := new(quanxiang.Email)
			content := "您有新的" + "流程的审批，请点击查看：" + t.homeHost + "/approvals/" +
				runID + "/" + taskID + "/WAIT_HANDLE_PAGE"

			email.Title = "审批提醒"

			email.Content = &quanxiang.Content{
				Content:    content,
				TemplateID: "quanliang",
			}
			email.To = []string{usersInfo.Data.Email}
			//todo 这里还需要对content进行处理，可能是从表单字段取值又或者是公式计算
			m.Email = email
			sendReqs = append(sendReqs, m)
		}

		if len(sendReqs) > 0 {
			_, err := t.qx.SendMessage(ctx, sendReqs)
			if err != nil {
				level.Error(t.logger).Log("message", "examine do send message", "err", err)
				continue
			}
			level.Info(t.logger).Log("message", "examine do send message ok", "email", usersInfo.Data.Email)
		}

	}
}
