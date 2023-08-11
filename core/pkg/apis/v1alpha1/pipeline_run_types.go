package v1alpha1

// PipeplineRunSpec defines the desired state of PipeplineRun
type PipeplineRunSpec struct {
	// +optional
	Params []*KeyAndValue `json:"params,omitempty"`

	// +optional
	Communal []*KeyAndValue `json:"communal,omitempty"`

	// +optional
	PipelineRef string `json:"pipelineRef,omitempty"`
}

type NodeStatusSpec struct {
	// Name is task name
	// +require
	Name string `json:"name,omitempty"`

	// +optional
	Output []*KeyAndValue `json:"output,omitempty"`

	// Status is the RunStatus
	// +optional
	Status NodeStatus `json:"status,omitempty"`

	// StartTime is the time the task is actually started.
	// +optional
	StartTime int64 `json:"startTime,omitempty"`

	// CompletionTime is the time the task completed.
	// +optional
	CompletionTime int64 `json:"completionTime,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`
}

type PipelineSatus string

const (
	PipelineRunRunning PipelineSatus = "Running"
	PipelineRunFinish  PipelineSatus = "Finish"
	PipelineRunKill    PipelineSatus = "Kill"
)

func (p *PipelineSatus) IsFinish() bool {
	return *p == PipelineRunFinish || *p == PipelineRunKill
}

// PipeplineRunStatus defines the observed state of PipeplineRun
type PipeplineRunStatus struct {
	// Status of the condition, one of True, False, Unknown.
	// +required
	Status *PipelineSatus `json:"status"`

	// +optional
	Message string `json:"message,omitempty"`

	// +optional
	NodeRun []*NodeStatusSpec `json:"taskRun,omitempty"`
}

type PipeplineRun struct {
	Metadata Metadata `json:",inline"`

	Spec   PipeplineRunSpec   `json:"spec,omitempty"`
	Status PipeplineRunStatus `json:"status,omitempty"`
}

type Metadata struct {
	Annotations map[string]string
}
