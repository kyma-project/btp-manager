package utils

import "fmt"

func BuildFilenameWithExtension(filename, extension string) string {
	return fmt.Sprintf("%s.%s", filename, extension)
}
