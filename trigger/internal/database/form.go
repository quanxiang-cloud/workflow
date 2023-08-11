package database

import "context"

type FormTrigger struct {
	ID           int64
	Name         string
	PipelineName string
	AppID        string
	TableID      string
	Type         string
	Filters      []string
	CreatedAt    int64
	UpdatedAt    int64
}

type FormRepo interface {
	Create(ctx context.Context, fl *FormTrigger) error
	Delete(ctx context.Context, id int64) error
	Get(ctx context.Context, name string) (*FormTrigger, error)
	List(ctx context.Context) ([]*FormTrigger, error)
	GetByTableID(ctx context.Context, tableID, _t string) ([]*FormTrigger, error)
}
