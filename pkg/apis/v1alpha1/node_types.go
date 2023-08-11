package v1alpha1

type NodeStatus string

const (
	Skip    NodeStatus = "Skip"
	Pending NodeStatus = "Pending"
	Finish  NodeStatus = "Finish"
	Kill    NodeStatus = "Kill"
)

type When struct {
	Input string `json:"input,omitempty"`

	Operator string `json:"operator,omitempty"`

	Values []string `json:"values,omitempty"`
}

type NodeSpec struct {
	Type string `json:"type,omitempty"`

	// Dependencies is a set of execution dependent tasks.
	// Only when all dependent tasks are completed can the current task be executed.
	// +optional
	Dependencies []string `json:"dependencies,omitempty"`

	// Params is a list of input parameters required to run the task.
	// +optional
	Params []*KeyAndValue `json:"params,omitempty"`

	OutPut []string `json:"outPut,omitempty"`

	// +optional
	When []When `json:"when,omitempty"`
}

type Node struct {
	Name string `json:"name,omitempty"`

	Spec NodeSpec `json:"spec,omitempty"`
}
