package ymlutils

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GatherChartGvks collects unique GVKs from all YAML templates under fsys.
// Production callers: pass os.DirFS(chartPath).
// Tests: pass fstest.MapFS.
func GatherChartGvks(fsys fs.FS) ([]schema.GroupVersionKind, error) {
	var allGvks []schema.GroupVersionKind

	err := fs.WalkDir(fsys, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			return nil
		}
		f, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		fileGvks, err := ExtractGvkFromYml(string(data))
		if err != nil {
			return err
		}
		for _, gvk := range fileGvks {
			if !containsGvk(allGvks, gvk) {
				allGvks = append(allGvks, gvk)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return allGvks, nil
}

func containsGvk(gvks []schema.GroupVersionKind, gvk schema.GroupVersionKind) bool {
	for _, v := range gvks {
		if v == gvk {
			return true
		}
	}
	return false
}

// ExtractGvkFromYml parses a multi-document YAML string and returns all GVKs found.
// It uses line scanning rather than a full YAML parser so that it tolerates
// Helm template syntax ({{ ... }}) present in chart template files.
func ExtractGvkFromYml(wholeFile string) ([]schema.GroupVersionKind, error) {
	var gvks []schema.GroupVersionKind
	for _, part := range strings.Split(wholeFile, "---\n") {
		if strings.TrimSpace(part) == "" {
			continue
		}
		var apiVersion, kind string
		for _, line := range strings.Split(part, "\n") {
			if strings.HasPrefix(line, "apiVersion:") {
				apiVersion = strings.TrimSpace(strings.TrimPrefix(line, "apiVersion:"))
			}
			if strings.HasPrefix(line, "kind:") {
				kind = strings.TrimSpace(strings.TrimPrefix(line, "kind:"))
			}
		}
		if apiVersion == "" || kind == "" {
			continue
		}
		gv, err := schema.ParseGroupVersion(apiVersion)
		if err != nil {
			return nil, err
		}
		gvks = append(gvks, gv.WithKind(kind))
	}
	return gvks, nil
}

// ExtractStringValueFromYamlForGivenKey reads filePath and returns the string value for key.
func ExtractStringValueFromYamlForGivenKey(filePath string, key string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return extractStringValue(data, key)
}

func extractStringValue(data []byte, key string) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return "", err
	}
	val, ok := obj[key]
	if !ok {
		return "", nil
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("value for key %q is not a string", key)
	}
	return str, nil
}

// CopyManifestsFromYamlsIntoOneYaml appends all YAML files from fsys into targetYaml (a file path).
// Production callers: pass os.DirFS(sourceDir).
// Tests: pass fstest.MapFS.
func CopyManifestsFromYamlsIntoOneYaml(fsys fs.FS, targetYaml string) error {
	target, err := os.OpenFile(targetYaml, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	walkErr := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".yml") && !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}
		f, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		_, err = target.Write(data)
		return err
	})

	closeErr := target.Close()
	if walkErr != nil {
		return walkErr
	}
	return closeErr
}
