package driver

// ParamType is the canonical string name of a supported parameter/return type.
type ParamType = string

const (
	TypeInt          ParamType = "int"
	TypeFloat        ParamType = "float"
	TypeString       ParamType = "string"
	TypeBool         ParamType = "bool"
	TypeIntArray     ParamType = "int[]"
	TypeFloatArray   ParamType = "float[]"
	TypeStringArray  ParamType = "string[]"
	TypeBoolArray    ParamType = "bool[]"
	TypeIntMatrix    ParamType = "int[][]"
	TypeStringMatrix ParamType = "string[][]"
	TypeListNode     ParamType = "ListNode"
	TypeTreeNode     ParamType = "TreeNode"
)

// Parameter is a single named+typed argument in a function signature.
type Parameter struct {
	Name string    `json:"name"`
	Type ParamType `json:"type"`
}

// FunctionSpec describes the callable interface for a problem.
// It is stored as JSONB on the problems table and used to generate
// per-language driver code at submission time.
type FunctionSpec struct {
	FunctionName string      `json:"function_name"`
	Parameters   []Parameter `json:"parameters"`
	ReturnType   ParamType   `json:"return_type"`
}
