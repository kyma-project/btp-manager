package ymlutils

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	expectedLenAfterSplit = 2
)

func GatherChartGvks(chartPath string) ([]schema.GroupVersionKind, error) {
	var allGvks []schema.GroupVersionKind
	appendToSlice := func(gvk schema.GroupVersionKind) {
		if reflect.DeepEqual(gvk, schema.GroupVersionKind{}) {
			return
		}
		for _, v := range allGvks {
			if reflect.DeepEqual(gvk, v) {
				return
			}
		}
		allGvks = append(allGvks, gvk)
	}

	root := fmt.Sprintf("%s/templates/", chartPath)
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(info.Name(), ".yml") && !strings.HasSuffix(info.Name(), ".yaml") {
			return nil
		}

		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		fileGvks, err := ExtractGvkFromYml(string(bytes))
		if err != nil {
			return err
		}

		for _, gvk := range fileGvks {
			appendToSlice(gvk)
		}

		return nil
	}); err != nil {
		return []schema.GroupVersionKind{}, err
	}

	return allGvks, nil
}

func ExtractGvkFromYml(wholeFile string) ([]schema.GroupVersionKind, error) {
	var gvks []schema.GroupVersionKind
	parts := strings.Split(wholeFile, "---\n")
	for _, part := range parts {
		if part == "" {
			continue
		}
		var apiVersion, kind string
		lines := strings.Split(part, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "apiVersion:") {
				apiVersion = strings.TrimSpace(strings.Split(line, ":")[1])
			}
			if strings.HasPrefix(line, "kind:") {
				kind = strings.TrimSpace(strings.Split(line, ":")[1])
			}
		}
		if apiVersion != "" && kind != "" {
			apiVersion, err := schema.ParseGroupVersion(apiVersion)
			if err != nil {
				return nil, err
			}
			gvks = append(gvks, apiVersion.WithKind(kind))
		}
	}

	return gvks, nil
}

func ExtractStringValueFromYamlForGivenKey(filePath string, key string) (string, error) {
	if !strings.HasSuffix(key, ":") {
		key = key + ":"
	}

	file, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(file), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key) {
			result := strings.Split(line, ":")
			if len(result) != expectedLenAfterSplit {
				return "", fmt.Errorf("line after split has incorrent number of elements: %d", len(result))
			}
			return strings.TrimSpace(result[1]), nil
		}
	}

	return "", nil
}
