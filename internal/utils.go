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

func GetValueByKey(key string, data map[string][]byte) ([]byte, error) {
	value, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("while getting data for key: %s", key)
	}
	if value == nil || bytes.Equal(value, []byte{}) {
		return nil, fmt.Errorf("empty data for key: %s", key)
	}
	return value, nil
}
