package db

import "context"

type OldFlow struct {
	ID        string
	FlowJson  string
	Status    string
	AppID     string
	FormID    string
	CreatedBy string
}

type OldFlowRepo interface {
	Created(ctx context.Context, flow *OldFlow) error
	Updated(ctx context.Context, flow *OldFlow) error
	UpdatedStatus(ctx context.Context, flow *OldFlow) error
	Deleted(ctx context.Context, flow *OldFlow) error
	Get(ctx context.Context, id string) (*OldFlow, error)
	Search(ctx context.Context, page, limit int, appID string) ([]OldFlow, int, error)
}
