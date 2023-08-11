package versioned

import (
	"context"
	"testing"

	"git.yunify.com/quanxiang/workflow/apis"
	"git.yunify.com/quanxiang/workflow/pkg/helper/logger"
)

func TestExecPipeline(t *testing.T) {
	logger := logger.NewLogger("debug")
	c := New("localhost:80", logger)
	err := c.Exec(context.Background(), &apis.ExecPipeline{
		Name: "test",
	})
	if err != nil {
		t.Fail()
	}
}
