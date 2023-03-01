package utils

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

func BuildKeyNameWithExtension(filename, extension string) string {
	return fmt.Sprintf("%s.%s", filename, extension)
}

func StructToByteArray(s any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(s)
	if err != nil {
		return []byte{}, err
	}

	return buffer.Bytes(), nil
}
