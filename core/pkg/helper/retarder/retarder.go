package retarder

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultRingBuffer int64 = 100
)

type Retarder struct {
	sync.Mutex

	p             int64
	ringBufferLen int64

	ringBuffer [][]*carrier

	// Call, when the delay arrives, the callback function is called
	Call func(data Data)
}

type Data interface{}

type carrier struct {
	Data Data
}

func New(bl int64, call func(data Data)) *Retarder {
	if bl == 0 {
		bl = defaultRingBuffer
	}
	return &Retarder{
		ringBufferLen: bl,
		ringBuffer:    make([][]*carrier, bl),
		Call:          call,
	}
}

func (r *Retarder) Add(data Data, time int64 /*second*/) error {
	if time > r.ringBufferLen {
		return errors.New("over maximum length")
	}
	if time == 0 {
		return errors.New("invalid time")
	}

	p := (atomic.LoadInt64(&r.p) + time) % r.ringBufferLen
	r.Lock()
	defer r.Unlock()

	if r.ringBuffer[p] == nil {
		r.ringBuffer[p] = make([]*carrier, 0)
	}

	r.ringBuffer[p] = append(r.ringBuffer[p], &carrier{
		Data: data,
	})

	return nil
}

func (r *Retarder) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			r.Lock()
			carriers := r.ringBuffer[r.p]
			r.ringBuffer[r.p] = make([]*carrier, 0)
			atomic.SwapInt64(&r.p, (atomic.LoadInt64(&r.p)+1)%r.ringBufferLen)
			r.Unlock()
			go r.call(carriers)
		case <-ctx.Done():
			return
		}
	}
}

func (r *Retarder) call(carriers []*carrier) {
	for _, carrier := range carriers {
		r.Call(carrier.Data)
	}
}
