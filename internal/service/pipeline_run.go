package service

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.yunify.com/quanxiang/workflow/apis"
	"git.yunify.com/quanxiang/workflow/internal/common"
	"git.yunify.com/quanxiang/workflow/internal/database"
	"git.yunify.com/quanxiang/workflow/internal/database/mysql"
	"git.yunify.com/quanxiang/workflow/pkg/apis/v1alpha1"
	"git.yunify.com/quanxiang/workflow/pkg/helper/errors"
	"git.yunify.com/quanxiang/workflow/pkg/helper/retarder"
	pn "git.yunify.com/quanxiang/workflow/pkg/node"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func NewPipelineRunService(ctx context.Context) (apis.PipelineRunService, error) {
	return &pipelineRunService{
		ctx: ctx,
		runner: runner{
			ch: make(chan int64, 10000),
		},
	}, nil
}

type pipelineRunService struct {
	ctx    context.Context
	conf   *common.Config
	logger log.Logger

	runner   runner
	retarder *retarder.Retarder

	pipelineRunRepo database.PipelineRunRepo
}

func (p *pipelineRunService) SetLogger(logger log.Logger) {
	p.logger = log.With(logger, "module", "service")
}

func (p *pipelineRunService) SetDB(db *sql.DB) {
	p.pipelineRunRepo = mysql.NewPipelineRun(db)
	p.runner.pipelineRunRepo = p.pipelineRunRepo
}

func (s *pipelineRunService) SetConfig(conf *common.Config) {
	s.conf = conf

	s.init()
}

func (s *pipelineRunService) init() {
	s.runner.logger = s.logger
	s.runner.nodes = make(map[string]pn.Interface)
	for _, n := range s.conf.Nodes {
		s.runner.nodes[n.Type] = pn.New(n.Host, s.logger)
	}

	s.runner.nodes["null"] = &pn.Null{}
	s.runner.Run(s.ctx, s.conf.Parallel)

	if s.conf.Retarder.Enable {
		s.retarder = retarder.New(int64(s.conf.Retarder.BufferSize), func(data retarder.Data) {
			level.Info(s.logger).Log("message", "retry exec pipeline run", "pipelineRunID", data.(int64))
			go s.runner.set(data.(int64))
		})
		s.runner.delay = s.conf.Retarder.Delay
		s.runner.retarder = s.retarder

		go s.retarder.Run(s.ctx)
	}

	// try to exec unfinished pipeline run
	lostFound, err := s.pipelineRunRepo.ListRunning(s.ctx)
	if err != nil {
		level.Error(s.logger).Log("message", err)
		return
	}
	if len(lostFound) != 0 {
		go func() {
			<-time.After(time.Second * 10)
			for _, id := range lostFound {
				s.runner.set(id)
			}
			level.Info(s.logger).Log("message", "the unfinished task is loaded")
		}()
	}

}

func (p *pipelineRunService) Create(ctx context.Context, in *apis.CreatePipelineRun) error {
	//  save pipeline run to DB
	plr := &database.PipelineRun{
		Pipeline: *in.Pipeline,
		Spec: v1alpha1.PipeplineRunSpec{
			Params:      in.Params,
			PipelineRef: in.Pipeline.Name,
		},
		Status:    v1alpha1.PipeplineRunStatus{},
		CreatedAt: time.Now().Unix(),
	}
	err := p.pipelineRunRepo.Create(ctx, plr)
	if err != nil {
		return errors.Wrap(err, "fail save pipepline run to database")
	}

	err = p.pipelineRunRepo.Update(ctx, plr)
	if err != nil {
		return errors.Wrap(err, "fail update pipepline run to database")
	}

	p.runner.set(plr.ID /*row id*/)
	return nil
}

func (p *pipelineRunService) Exec(ctx context.Context, in *apis.ExecPipelineRun) error {
	plr, err := p.pipelineRunRepo.Get(ctx, in.ID)
	if err != nil {
		level.Error(p.logger).Log("message", err.Error())
		return errors.Wrap(err, "fail get pipepline run from database")
	}
	if plr == nil {
		return errors.NewErr(http.StatusNotFound, &errors.CodeError{
			Code:    http.StatusNotFound,
			Message: "pipeline run not exists",
		})
	}
	p.runner.set(in.ID)
	return nil
}

type runner struct {
	logger          log.Logger
	pipelineRunRepo database.PipelineRunRepo
	retarder        *retarder.Retarder
	ch              chan int64

	delay int64

	nodes map[string]pn.Interface
}

func (r *runner) set(id int64) {
	r.ch <- id
}

func (r *runner) getNode(_t string) pn.Interface {
	i, ok := r.nodes[_t]
	if !ok {
		return &pn.None{}
	}

	return i
}

func (r *runner) Run(ctx context.Context, parallel int) {
	for ; parallel > 0; parallel-- {
		level.Info(r.logger).Log("message", "runner is ready", "runnerID", parallel)
		go func(r *runner, runnerID int) {
			for {
				select {
				case <-ctx.Done():
					level.Info(r.logger).Log("message", "try to exec pipeline", "runnerID", runnerID)
					return
				case plrID := <-r.ch:
					level.Info(r.logger).Log("message", "try to exec pipeline", "id", plrID)
					r.run(plrID)
				}
			}
		}(r, parallel)
	}

}

func (r *runner) run(pipelineRunID int64) {
	ctx := context.Background()
	plr, err := r.pipelineRunRepo.Get(ctx, pipelineRunID)
	if err != nil {
		level.Error(r.logger).Log("message", err.Error(), "pipelineRunID", pipelineRunID)
		return
	}

	// TODO: 增加业务锁，防止重复执行

	if plr == nil {
		level.Error(r.logger).Log("message", "fail get pipeline run", "pipelineRunID", pipelineRunID)
		return
	}

	if plr.State.IsFinish() {
		// Do nothing
		level.Info(r.logger).Log("message", "pipeline run is finish", "pipelineRunID", pipelineRunID)
		return
	}

	var state = v1alpha1.PipelineRunRunning
	plr.Status.Status = &state

	node := r.getNodeToExecute(plr)
	if node != nil {
		err = r.exec(ctx, node, plr)
		if err != nil {
			level.Error(r.logger).Log("message", err, "pipelineRunID", plr.ID, "nodeName", node.Name)
			// Join retry queue
			if r.retarder != nil {
				err = r.retarder.Add(pipelineRunID, r.delay)
				if err != nil {
					level.Error(r.logger).Log("message", err, "pipelineRunID", plr.ID, "delay", r.delay)
				}
				level.Info(r.logger).Log("message", "try to delay", "pipelineRunID", plr.ID, "delay", r.delay)
			}
			return
		}
		level.Info(r.logger).Log("message", "exec node", "pipelineRunID", pipelineRunID, "nodeName", node.Name)
	} else {
		state = v1alpha1.PipelineRunFinish
		plr.State = v1alpha1.PipelineRunFinish
		plr.Status.Status = &state
	}

	if plr.Status.Status.IsFinish() && !plr.State.IsFinish() {
		plr.State = v1alpha1.PipelineRunFinish
	}

	err = r.pipelineRunRepo.Update(ctx, plr)
	if err != nil {
		level.Error(r.logger).Log("message", err.Error(), "pipelineRunID", pipelineRunID)
		return
	}
	if plr.Status.NodeRun[len(plr.Status.NodeRun)-1].Status != v1alpha1.Pending {
		// try to exec next node
		r.set(plr.ID)
	}
}

func (r *runner) exec(ctx context.Context, node *v1alpha1.Node, plr *database.PipelineRun) error {
	result, err := r.getNode(node.Spec.Type).Do(ctx, &pn.Request{
		Params: r.parseParams(node.Spec.Params, plr),
		Metadata: v1alpha1.Metadata{
			Annotations: map[string]string{
				"database.pipelineRun/id":       strconv.Itoa(int(plr.ID)),
				"database.pipelineRunNode/name": node.Name,
			},
		},
	})

	status := plr.Status.NodeRun[len(plr.Status.NodeRun)-1]
	if err != nil {
		level.Error(r.logger).Log("message", err)
		status.Message = err.Error()
		return err
	} else {
		status.Status = result.Status
		status.Output = result.Out
		status.Message = result.Message

		for _, communal := range result.Communal {
			cp := communal
			cl := getKV(communal.Key, plr.Spec.Communal)
			if cl == nil {
				plr.Spec.Communal = append(plr.Spec.Communal, cp)
			} else {
				cl.Value = cp.Value
			}
		}

		switch status.Status {
		case v1alpha1.Finish:
			status.CompletionTime = time.Now().Unix()
		case v1alpha1.Kill:
			plr.State = v1alpha1.PipelineRunKill
		default:
			status.Status = v1alpha1.Pending
		}
	}

	plr.Status.NodeRun[len(plr.Status.NodeRun)-1] = status
	return nil
}

func (r *runner) parseParams(input []*v1alpha1.KeyAndValue, plr *database.PipelineRun) (result []*v1alpha1.KeyAndValue) {
	var getValueFromKV = func(name string, kvs []*v1alpha1.KeyAndValue) *string {
		for _, kv := range kvs {
			if kv.Key == name {
				return &kv.Value
			}
		}

		return nil
	}

	var genKV = func(kvs []*v1alpha1.KeyAndValue, pss []v1alpha1.ParamSpec) []*v1alpha1.KeyAndValue {
		result := make([]*v1alpha1.KeyAndValue, 0, len(pss))
		for _, param := range pss {
			kv := &v1alpha1.KeyAndValue{
				Key: param.Name,
			}

			value := getValueFromKV(kv.Key, kvs)
			if value == nil {
				kv.Value = param.Default
			} else {
				kv.Value = *value
			}

			result = append(result, kv)
		}

		return result
	}

	params := genKV(plr.Spec.Params, plr.Pipeline.Spec.Params)
	communal := genKV(plr.Spec.Communal, plr.Pipeline.Spec.Communal)
	for _, param := range plr.Pipeline.Spec.Communal {
		kv := &v1alpha1.KeyAndValue{
			Key: param.Name,
		}

		value := getValueFromKV(kv.Key, plr.Spec.Params)
		if value == nil {
			kv.Value = param.Default
		} else {
			kv.Value = *value
		}

		params = append(params, kv)
	}

	for _, param := range input {
		kv := &v1alpha1.KeyAndValue{
			Key: param.Key,
		}

		var value *string
		switch {
		case strings.HasPrefix(param.Value, "$(params."):
			name := strings.TrimSuffix(strings.TrimPrefix(param.Value, "$(params."), ")")
			value = getValueFromKV(name, params)
		case strings.HasPrefix(param.Value, "$(task."):
			nwn := strings.Split(strings.TrimSuffix(strings.TrimPrefix(param.Value, "$(task."), ")"), ".output.")
			if len(nwn) == 2 {
				for _, node := range plr.Status.NodeRun {
					if node.Name == nwn[0] {
						value = getValueFromKV(nwn[1], node.Output)
						break
					}
				}
			}
		case strings.HasPrefix(param.Value, "$(communal."):
			name := strings.TrimSuffix(strings.TrimPrefix(param.Value, "$(communal."), ")")
			value = getValueFromKV(name, communal)
		default:
			value = &param.Value
		}

		if value == nil {
			null := ""
			value = &null
		}

		kv.Value = *value
		result = append(result, kv)
	}
	return
}

func (r *runner) getNodeToExecute(plr *database.PipelineRun) *v1alpha1.Node {
	var skipByDependency = func(depencies []string, status []*v1alpha1.NodeStatusSpec) bool {
		if len(depencies) == 0 {
			return false
		}
		if len(status) == 0 {
			return true
		}

		for _, depency := range depencies {
			if func(depency string, status []*v1alpha1.NodeStatusSpec) bool {
				for _, ss := range status {
					if ss.Name == depency && ss.Status == v1alpha1.Finish {
						return false
					}
				}
				return true
			}(depency, status) {
				return true
			}
		}

		return false
	}

	var skipByWhen = func(when []v1alpha1.When, plr *database.PipelineRun) bool {
		for _, w := range when {
			if len(w.Values) == 0 {
				return false
			}

			input := r.parseParams([]*v1alpha1.KeyAndValue{{Value: w.Input}}, plr)[0].Value

			values := make([]string, 0, len(w.Values))
			for _, value := range w.Values {
				values = append(values, r.parseParams([]*v1alpha1.KeyAndValue{{Value: value}}, plr)[0].Value)
			}

			switch w.Operator {
			case "eq":
				return input != values[0]
			case "in":
				for _, v := range values {
					if v == input {
						return false
					}
				}

				return true
			default:
				return true
			}

		}
		return false
	}

	// try to get a node, If no node can be found, nil is returned
	var cursor int
	if len(plr.Status.NodeRun) == 0 {
		cursor = 0
	} else {
		cursor = len(plr.Status.NodeRun) - 1
		if plr.Status.NodeRun[cursor].Status == v1alpha1.Pending {
			return &plr.Pipeline.Spec.Nodes[cursor]
		}
		cursor++
	}

	var sholdSkip = func(node v1alpha1.Node, plr *database.PipelineRun) bool {
		if skipByDependency(node.Spec.Dependencies, plr.Status.NodeRun) {
			return true
		}

		return skipByWhen(node.Spec.When, plr)
	}

	for ; cursor < len(plr.Pipeline.Spec.Nodes); cursor++ {
		state := &v1alpha1.NodeStatusSpec{
			Name:      plr.Pipeline.Spec.Nodes[cursor].Name,
			StartTime: time.Now().Unix(),
		}

		plr.Status.NodeRun = append(plr.Status.NodeRun, state)
		if sholdSkip(plr.Pipeline.Spec.Nodes[cursor], plr) {
			state.Status = v1alpha1.Skip
			state.CompletionTime = time.Now().Unix()
			continue
		}
		return &plr.Pipeline.Spec.Nodes[cursor]
	}

	return nil
}

func getKV(name string, kvs []*v1alpha1.KeyAndValue) *v1alpha1.KeyAndValue {
	for _, elem := range kvs {
		if elem.Key == name {
			return elem
		}
	}
	return nil
}
