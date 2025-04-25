package ymlutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func AddSuffixToNameInManifests(manifestsDir, suffix string) error {
	if err := filepath.Walk(manifestsDir, func(path string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(info.Name(), ".yml") {
			return nil
		}

		filename := fmt.Sprint(path)
		input, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		reachedMetadata, reachedSpec := false, false
		lines := strings.Split(string(input), "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "metadata:") {
				reachedMetadata = true
				continue
			}
			if strings.HasPrefix(line, "spec:") {
				reachedSpec = true
				continue
			}
			if reachedMetadata && strings.HasPrefix(strings.TrimSpace(line), "name:") {
				lines[i] = lines[i] + suffix
				reachedMetadata = false
				continue
			}
			if reachedSpec && strings.HasPrefix(strings.TrimSpace(line), "group:") {
				lines[i] = lines[i] + suffix
				reachedSpec = false
				continue
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

func UpdateChartVersion(chartPath, newVersion string) error {
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
	err = os.WriteFile(filename, []byte(output), 0700)
	if err != nil {
		return err
	}
	return nil
}
