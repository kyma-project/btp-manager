package ymlutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func TransformCharts(chartPath string, suffix string) error {
	root := fmt.Sprintf("%s/templates/", chartPath)
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(info.Name(), ".yml") {
			return nil
		}

		filename := fmt.Sprintf("%s/%s", root, info.Name())
		input, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		lines := strings.Split(string(input), "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "name:") {
				lines[i] = lines[i] + suffix
			}
		}
		output := strings.Join(lines, "\n")
		err = os.WriteFile(filename, []byte(output), 0644)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func UpdateVersion(chartPath, newVersion string) error {
	filename := fmt.Sprintf("%s/%s", chartPath, "Chart.yaml")
	input, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	const versionKey = "version: "
	lines := strings.Split(string(input), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, versionKey) {
			lines[i] = versionKey + newVersion
		}
	}
	output := strings.Join(lines, "\n")
	err = os.WriteFile(filename, []byte(output), 0644)
	if err != nil {
		return err
	}
	return nil
}
