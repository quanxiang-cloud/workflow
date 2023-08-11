package trigger

import "context"

type Data interface{}

type Interface interface {
	Add(ctx context.Context, name, piplineName string, data Data) error
	Remove(ctx context.Context, name string) error
	GetDataType(ctx context.Context) Data
	Run(ctx context.Context)
}
