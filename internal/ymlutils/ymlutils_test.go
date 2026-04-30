package ymlutils

import (
	"testing"
	"testing/fstest"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var gopkgYamlUnmarshal = yaml.Unmarshal

// --- extractStringValue ---

func TestExtractStringValue_SimpleKey(t *testing.T) {
	data := []byte("version: v1.2.3\nname: my-chart\n")
	got, err := extractStringValue(data, "version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "v1.2.3" {
		t.Errorf("got %q, want %q", got, "v1.2.3")
	}
}

func TestExtractStringValue_ValueWithColon(t *testing.T) {
	// regression: old split-on-colon would return "//registry.example.com/image" not the full URL
	data := []byte("image: registry.example.com/myimage:latest\n")
	got, err := extractStringValue(data, "image")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "registry.example.com/myimage:latest" {
		t.Errorf("got %q, want %q", got, "registry.example.com/myimage:latest")
	}
}

func TestExtractStringValue_MissingKey(t *testing.T) {
	data := []byte("name: foo\n")
	got, err := extractStringValue(data, "version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string for missing key, got %q", got)
	}
}

// --- ExtractGvkFromYml ---

func TestExtractGvkFromYml_SingleDoc(t *testing.T) {
	input := "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: foo\n"
	gvks, err := ExtractGvkFromYml(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gvks) != 1 {
		t.Fatalf("expected 1 gvk, got %d", len(gvks))
	}
	want := schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	if gvks[0] != want {
		t.Errorf("got %v, want %v", gvks[0], want)
	}
}

func TestExtractGvkFromYml_MultiDoc(t *testing.T) {
	input := "apiVersion: v1\nkind: ServiceAccount\n---\napiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRole\n"
	gvks, err := ExtractGvkFromYml(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gvks) != 2 {
		t.Fatalf("expected 2 gvks, got %d: %v", len(gvks), gvks)
	}
	want0 := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"}
	want1 := schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}
	if gvks[0] != want0 {
		t.Errorf("gvks[0] got %v, want %v", gvks[0], want0)
	}
	if gvks[1] != want1 {
		t.Errorf("gvks[1] got %v, want %v", gvks[1], want1)
	}
}

func TestExtractGvkFromYml_Deduplication(t *testing.T) {
	// same GVK appearing twice should not be deduplicated by ExtractGvkFromYml itself
	// (deduplication is GatherChartGvks' responsibility)
	input := "apiVersion: v1\nkind: ConfigMap\n---\napiVersion: v1\nkind: ConfigMap\n"
	gvks, err := ExtractGvkFromYml(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gvks) != 2 {
		t.Fatalf("expected 2 (no dedup at this level), got %d", len(gvks))
	}
}

func TestExtractGvkFromYml_HelmTemplateSyntax(t *testing.T) {
	// must not error on Helm {{ }} template tokens
	input := "apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: {{ .Release.Name }}\n  namespace: {{ .Release.Namespace }}\n"
	gvks, err := ExtractGvkFromYml(input)
	if err != nil {
		t.Fatalf("unexpected error on helm template: %v", err)
	}
	if len(gvks) != 1 {
		t.Fatalf("expected 1 gvk, got %d", len(gvks))
	}
	want := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"}
	if gvks[0] != want {
		t.Errorf("got %v, want %v", gvks[0], want)
	}
}

// --- GatherChartGvks ---

func TestGatherChartGvks(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/deploy.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: apps/v1\nkind: Deployment\n"),
		},
		"templates/sa.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ServiceAccount\n"),
		},
		"templates/sub/cm.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\n"),
		},
		"templates/not-yaml.txt": &fstest.MapFile{Data: []byte("ignored")},
	}
	gvks, err := GatherChartGvks(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gvks) != 3 {
		t.Errorf("expected 3 unique gvks, got %d: %v", len(gvks), gvks)
	}
}

func TestGatherChartGvks_Deduplication(t *testing.T) {
	fsys := fstest.MapFS{
		"templates/a.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\n"),
		},
		"templates/b.yaml": &fstest.MapFile{
			Data: []byte("apiVersion: v1\nkind: ConfigMap\n"),
		},
	}
	gvks, err := GatherChartGvks(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gvks) != 1 {
		t.Errorf("expected 1 deduplicated gvk, got %d", len(gvks))
	}
}

// --- updateChartVersionInContent ---

func TestUpdateChartVersionInContent(t *testing.T) {
	data := []byte("apiVersion: v2\nname: my-chart\nversion: v0.1.0\n")
	out, err := updateChartVersionInContent(data, "v9.9.9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := extractStringValue(out, "version")
	if err != nil {
		t.Fatalf("unexpected error reading back version: %v", err)
	}
	if got != "v9.9.9" {
		t.Errorf("got %q, want %q", got, "v9.9.9")
	}
	// other fields preserved
	name, _ := extractStringValue(out, "name")
	if name != "my-chart" {
		t.Errorf("name field corrupted: got %q", name)
	}
}

// --- addSuffixToNameInContent ---

func TestAddSuffixToNameInContent_MetadataName(t *testing.T) {
	data := []byte("apiVersion: v1\nkind: ServiceAccount\nmetadata:\n  name: btp-operator\n  namespace: kyma-system\n")
	out, err := addSuffixToNameInContent(data, "-updated", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed map[string]interface{}
	if err := gopkgYamlUnmarshal(out, &parsed); err != nil {
		t.Fatal(err)
	}
	meta := parsed["metadata"].(map[string]interface{})
	if meta["name"] != "btp-operator-updated" {
		t.Errorf("metadata.name got %q, want %q", meta["name"], "btp-operator-updated")
	}
}

func TestAddSuffixToNameInContent_SpecGroup(t *testing.T) {
	data := []byte("apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nmetadata:\n  name: servicebindings.services.cloud.sap.com\nspec:\n  group: services.cloud.sap.com\n")
	out, err := addSuffixToNameInContent(data, "-updated", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed map[string]interface{}
	if err := gopkgYamlUnmarshal(out, &parsed); err != nil {
		t.Fatal(err)
	}
	spec := parsed["spec"].(map[string]interface{})
	if spec["group"] != "services.cloud.sap.com-updated" {
		t.Errorf("spec.group got %q, want %q", spec["group"], "services.cloud.sap.com-updated")
	}
	meta := parsed["metadata"].(map[string]interface{})
	if meta["name"] != "servicebindings.services.cloud.sap.com-updated" {
		t.Errorf("metadata.name got %q, want %q", meta["name"], "servicebindings.services.cloud.sap.com-updated")
	}
}

func TestAddSuffixToNameInContent_NoSpecGroup(t *testing.T) {
	data := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: my-config\n")
	out, err := addSuffixToNameInContent(data, "-updated", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed map[string]interface{}
	if err := gopkgYamlUnmarshal(out, &parsed); err != nil {
		t.Fatal(err)
	}
	meta := parsed["metadata"].(map[string]interface{})
	if meta["name"] != "my-config-updated" {
		t.Errorf("metadata.name got %q, want %q", meta["name"], "my-config-updated")
	}
}
