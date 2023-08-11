package v1alpha1

type PipelineSpec struct {
	Params   []ParamSpec `json:"params,omitempty"`
	Nodes    []Node      `json:"nodes,omitempty"`
	Communal []ParamSpec `json:"communal,omitempty"`
}

type Pipeline struct {
	Name string       `json:"name,omitempty"`
	Spec PipelineSpec `json:"spec,omitempty"`
}
