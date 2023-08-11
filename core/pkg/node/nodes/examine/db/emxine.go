package db

import (
	"context"
)

type Task struct {
	ID          string
	TaskID      string
	FlowID      string
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
	NodeDefKey  string
}

type TaskRepo interface {
	InsertBranch(ctx context.Context, data ...*Task) (err error)
	UpdateSubstitute(ctx context.Context, tasks *Task) error
	UpdateResult(ctx context.Context, tasks *Task) error
	UpdateByTaskID(ctx context.Context, tasks *Task) error
	GetByUserIDAndTaskID(ctx context.Context, userID, taskID string) (*Task, error)
	ListByTaskID(ctx context.Context, taskID string) ([]Task, error)
	ListByTaskIDAndNodeDefKey(ctx context.Context, taskID, nodeDefKey string) ([]Task, error)
	ListByFlowID(ctx context.Context, flowID string) ([]Task, error)
	UpdateUrgeTimes(ctx context.Context, tasks *Task) error
	GetByID(ctx context.Context, id string) (*Task, error)
	GetByUserID(ctx context.Context, userID, nodeType string, page, limit int) ([]Task, int, error)
	GetByCreated(ctx context.Context, userID string, page, limit int) ([]Task, int, error)
	//----------
	//CountTimeOut(ctx context.Context, userID, result, nodeType string) (int64, error)
	//CountUrgeTimes(ctx context.Context, userID, result, nodeType string) (int64, error)
	//GetByFlowInstanceID(ctx context.Context, flowInstanceID string)
	//UpdateByFlowInstanceID(ctx context.Context, task *Task) error
}
