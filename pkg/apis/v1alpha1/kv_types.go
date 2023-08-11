package v1alpha1

type KeyAndValue struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// ParamSpec defines arbitrary parameters needed beyond typed inputs.
type ParamSpec struct {
	// Name declares the name by which a parameter is referenced.
	Name string `json:"name"`

	// Default is the value a parameter takes if no input value is supplied.
	// +optional
	Default string `json:"default,omitempty"`

	// Description is a user-facing description of the parameter that may be
	// used to populate a UI.
	// +optional
	Description string `json:"description,omitempty"`
}
