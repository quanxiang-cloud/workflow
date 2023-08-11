package retarder

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRetarder(t *testing.T) {
	r := New(100, func(data Data) {
		fmt.Println(data)
	})

	tests := []struct {
		time   int64
		data   int64
		expect bool
	}{
		{
			time:   10,
			data:   1,
			expect: true,
		},
		{
			time:   5,
			data:   2,
			expect: true,
		},
		{
			time:   15,
			data:   3,
			expect: true,
		}, {
			time:   150,
			data:   3,
			expect: false,
		},
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	go r.Run(ctx)
	for _, test := range tests {
		if err := r.Add(test.data, test.time); (err == nil) != test.expect {
			t.Fail()
		}
	}
	<-ctx.Done()
}
