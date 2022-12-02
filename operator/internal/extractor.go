package ymlutils

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

type btpOperatorGvk struct {
	APIVersion string
	Kind       string
}

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

		if !strings.HasSuffix(info.Name(), ".yml") {
			return nil
		}

		bytes, err := os.ReadFile(fmt.Sprintf("%s/%s", root, info.Name()))
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
		var yamlGvk btpOperatorGvk
		lines := strings.Split(part, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "apiVersion:") {
				yamlGvk.APIVersion = strings.TrimSpace(strings.Split(line, ":")[1])
			}

			if strings.HasPrefix(line, "kind:") {
				yamlGvk.Kind = strings.TrimSpace(strings.Split(line, ":")[1])
			}
		}
		if yamlGvk.Kind != "" && yamlGvk.APIVersion != "" {
			apiVersion := strings.Split(yamlGvk.APIVersion, "/")
			if len(apiVersion) == 1 {
				gvks = append(gvks, schema.GroupVersionKind{
					Kind:    yamlGvk.Kind,
					Version: apiVersion[0],
					Group:   "",
				})
			} else if len(apiVersion) == 2 {
				gvks = append(gvks, schema.GroupVersionKind{
					Kind:    yamlGvk.Kind,
					Version: apiVersion[1],
					Group:   apiVersion[0],
				})
			} else {
				return nil, fmt.Errorf("incorrect split of apiVersion")
			}
		}
	}

	return gvks, nil
}

func ExtractValueFromLine(filePath string, key string) (string, error) {
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
			return result[1], nil
		}
	}

	return "", nil
}
