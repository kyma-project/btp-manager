package gvksutils

import (
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GvksToStr(gvks []schema.GroupVersionKind) (string, error) {
	bytes, err := yaml.Marshal(gvks)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func StrToGvks(str string) ([]schema.GroupVersionKind, error) {
	var out []schema.GroupVersionKind
	err := yaml.Unmarshal([]byte(str), &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
