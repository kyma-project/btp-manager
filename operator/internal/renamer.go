package ymlutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func TransformCharts(chartPath string, sufix string, applySufix bool) error {
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
			if strings.HasPrefix(line, "  name:") {
				if !applySufix {
					split := strings.Split(line, sufix)
					lines[i] = split[0]
				} else {
					lines[i] = lines[i] + sufix
				}
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
