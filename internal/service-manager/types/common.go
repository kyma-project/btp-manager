package types

const (
	NamespaceLabel = "_namespace"
	K8sNameLabel   = "_k8sname"
	ClusterIDLabel = "_clusterid"

	ContextClusterID    = "clusterid"
	ContextSubaccountID = "subaccount_id"
	ContextNamespace    = "namespace"
)

type Labels map[string][]string

type Common struct {
	ID          string `json:"id,omitempty" yaml:"id,omitempty"`
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	CreatedAt   string `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
	Ready       bool   `json:"ready,omitempty" yaml:"ready,omitempty"`
	Labels      Labels `json:"labels,omitempty" yaml:"labels,omitempty"`
}
