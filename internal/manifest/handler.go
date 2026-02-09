package manifest

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

type Handler struct {
	Scheme               *runtime.Scheme
	manifestDeserializer runtime.Decoder
}

func (h *Handler) CollectObjectsFromDir(resourcesPath string) ([]runtime.Object, error) {
	manifests, err := h.GetManifestsFromDir(resourcesPath)
	if err != nil {
		return nil, fmt.Errorf("while getting manifests from %s directory: %w", resourcesPath, err)
	}

	objects, err := h.CreateObjectsFromManifests(manifests)
	if err != nil {
		return nil, fmt.Errorf("while creating objects from manifests: %w", err)
	}

	return objects, nil
}

func (h *Handler) GetManifestsFromDir(resourcesPath string) ([]string, error) {
	files, err := os.ReadDir(resourcesPath)
	if err != nil {
		return nil, err
	}

	manifests := make([]string, 0, len(files))
	for _, file := range files {
		if !isYamlFile(file.Name()) {
			continue
		}
		manifestsFromSingleYamlFile, err := h.GetManifestsFromYaml(fmt.Sprintf("%s%c%s", resourcesPath, os.PathSeparator, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("while getting manifests from YAML: %w", err)
		}
		manifests = append(manifests, manifestsFromSingleYamlFile...)
	}

	return manifests, nil
}

func isYamlFile(fileName string) bool {
	return strings.HasSuffix(fileName, ".yml") || strings.HasSuffix(fileName, ".yaml")
}

func (h *Handler) GetManifestsFromYaml(yamlFile string) ([]string, error) {
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		return nil, err
	}

	manifests := make([]string, 0)
	// matches lines that start with "---" and may have trailing spaces after and have a newline
	re := regexp.MustCompile(`(?m)^---\s*\n`)
	yamlParts := re.Split(string(data), -1)
	for _, part := range yamlParts {
		if part == "" || part == "\n" {
			continue
		}
		manifests = append(manifests, part)
	}
	if len(manifests) == 0 {
		return nil, nil
	}

	return manifests, nil
}

func (h *Handler) CreateObjectsFromManifests(manifests []string) ([]runtime.Object, error) {
	objects := make([]runtime.Object, 0, len(manifests))
	for _, manifest := range manifests {
		obj, err := h.CreateObjectFromManifest(manifest)
		if err != nil {
			return nil, fmt.Errorf("while creating object from manifest: %w", err)
		}
		objects = append(objects, obj)
	}

	return objects, nil
}

func (h *Handler) CreateObjectFromManifest(manifest string) (runtime.Object, error) {
	obj, _, err := h.getManifestDeserializer().Decode([]byte(manifest), nil, nil)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (h *Handler) getManifestDeserializer() runtime.Decoder {
	if h.manifestDeserializer == nil {
		h.manifestDeserializer = serializer.NewCodecFactory(h.Scheme).UniversalDeserializer()
	}
	return h.manifestDeserializer
}

func (h *Handler) ObjectsToUnstructured(objects []runtime.Object) ([]*unstructured.Unstructured, error) {
	us := make([]*unstructured.Unstructured, 0, len(objects))
	for _, obj := range objects {
		u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, fmt.Errorf("while creating Unstructured from Object: %w", err)
		}
		us = append(us, &unstructured.Unstructured{Object: u})
	}

	return us, nil
}

func (h *Handler) CreateUnstructuredObjectsFromManifestsDir(manifestsDir string) ([]*unstructured.Unstructured, error) {
	objects, err := h.CollectObjectsFromDir(manifestsDir)
	if err != nil {
		return nil, fmt.Errorf("while collecting objects from directory %s: %w", manifestsDir, err)
	}

	unstructuredObjects, err := h.ObjectsToUnstructured(objects)
	if err != nil {
		return nil, fmt.Errorf("while converting to unstructured: %w", err)
	}

	return unstructuredObjects, nil
}
