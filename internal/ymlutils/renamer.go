package ymlutils

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// AddSuffixToNameInManifests walks dir and appends suffix to metadata.name and spec.group
// in every .yml file, skipping names listed in excludeNames.
// Delegates per-file work to addSuffixToNameInContent.
func AddSuffixToNameInManifests(manifestsDir, suffix string, excludeNames ...string) error {
	excluded := make(map[string]bool, len(excludeNames))
	for _, n := range excludeNames {
		excluded[n] = true
	}
	return fs.WalkDir(os.DirFS(manifestsDir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".yml") {
			return nil
		}
		fullPath := filepath.Join(manifestsDir, path)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}
		updated, err := addSuffixToNameInContent(data, suffix, excluded)
		if err != nil {
			return err
		}
		return os.WriteFile(fullPath, updated, 0644)
	})
}

// addSuffixToNameInContent processes a potentially multi-document YAML byte slice,
// appending suffix to metadata.name and spec.group in every document,
// skipping documents whose metadata.name is in excluded.
func addSuffixToNameInContent(data []byte, suffix string, excluded map[string]bool) ([]byte, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(4)

	for {
		var node yaml.Node
		err := dec.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// A document node wraps the actual content node.
		// node.Kind == yaml.DocumentNode, node.Content[0] is the root.
		if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
			applyNameSuffix(node.Content[0], suffix, excluded)
		}

		if err := enc.Encode(&node); err != nil {
			return nil, err
		}
	}

	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// applyNameSuffix mutates a yaml.Node tree to append suffix to metadata.name and spec.group,
// skipping documents whose metadata.name is in excluded.
func applyNameSuffix(node *yaml.Node, suffix string, excluded map[string]bool) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	var metadataNode, specNode *yaml.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		val := node.Content[i+1]
		switch key.Value {
		case "metadata":
			metadataNode = val
		case "spec":
			specNode = val
		}
	}
	if metadataNode != nil {
		if excluded[getStringField(metadataNode, "name")] {
			return
		}
		appendToMappingStringField(metadataNode, "name", suffix)
	}
	if specNode != nil {
		appendToMappingStringField(specNode, "group", suffix)
	}
}

// appendToMappingStringField finds key in a MappingNode and appends suffix to its string value.
func appendToMappingStringField(node *yaml.Node, key, suffix string) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key && node.Content[i+1].Kind == yaml.ScalarNode {
			node.Content[i+1].Value += suffix
			return
		}
	}
}

// getStringField returns the string value of key in a MappingNode, or "" if not found.
func getStringField(node *yaml.Node, key string) string {
	if node == nil || node.Kind != yaml.MappingNode {
		return ""
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key && node.Content[i+1].Kind == yaml.ScalarNode {
			return node.Content[i+1].Value
		}
	}
	return ""
}

// UpdateChartVersion sets the version field in Chart.yaml at chartPath/Chart.yaml.
func UpdateChartVersion(chartPath, newVersion string) error {
	filename := filepath.Join(chartPath, "Chart.yaml")
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	updated, err := updateChartVersionInContent(data, newVersion)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, updated, 0644)
}

func updateChartVersionInContent(data []byte, newVersion string) ([]byte, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	obj["version"] = newVersion
	return yaml.Marshal(obj)
}
