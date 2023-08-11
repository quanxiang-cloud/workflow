package db

import "context"

type OldFlowVar struct {
	ID            string
	FlowID        string
	Code          string
	DefaultValue  string
	Desc          string
	FieldType     string
	Name          string
	Type          string
	Format        string
	CreatedAt     int64
	CreatorAvatar string
	CreatorID     string
	UpdatedAt     int64
	UpdatedID     string
	UpdatorName   string
}

type OldFlowVarRepo interface {
	Created(ctx context.Context, flow *OldFlowVar) error
	Updated(ctx context.Context, flow *OldFlowVar) error
	Deleted(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*OldFlowVar, error)
	GetByFlowID(ctx context.Context, flowID string) ([]OldFlowVar, error)
}
