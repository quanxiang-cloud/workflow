package node

import (
	"context"
	"net/http"

	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
)

type Result struct {
	Out      []*v1alpha1.KeyAndValue `json:"out,omitempty"`
	Communal []*v1alpha1.KeyAndValue `json:"communal,omitempty"`

	Status  v1alpha1.NodeStatus `json:"status,omitempty"`
	Message string              `json:"message,omitempty"`
}

type Request struct {
	Params   []*v1alpha1.KeyAndValue
	Metadata v1alpha1.Metadata
}

type Interface interface {
	Do(ctx context.Context, in *Request) (*Result, error)
}

type None struct{}

func (n *None) Do(ctx context.Context, in *Request) (*Result, error) {
	err := errors.NewErr(http.StatusInternalServerError, &errors.CodeError{
		Code:    http.StatusNotFound,
		Message: "Illegal node type",
	})
	return &Result{
		Status: v1alpha1.Finish,
	}, err
}

type Null struct{}

func (n *Null) Do(ctx context.Context, in *Request) (*Result, error) {
	return &Result{
		Status: v1alpha1.Finish,
	}, nil
}
