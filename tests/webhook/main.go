package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	certFile := getEnv("TLS_CERT_FILE", "/etc/tls/tls.crt")
	keyFile := getEnv("TLS_KEY_FILE", "/etc/tls/tls.key")
	addr := getEnv("LISTEN_ADDR", ":8443")
	ns := getEnv("WATCH_NAMESPACE", "kyma-system")
	secretName := getEnv("CA_BUNDLE_SECRET", "ca-bundle")

	restCfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	k8s, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/mutate", func(w http.ResponseWriter, r *http.Request) {
		var review admissionv1.AdmissionReview
		if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		_, err := k8s.CoreV1().Secrets(ns).Get(context.Background(), secretName, metav1.GetOptions{})
		secretExists := err == nil

		var patch []byte
		if secretExists {
			patch = buildPatch(secretName, review.Request.Object.Raw)
		} else {
			patch = []byte("[]")
		}

		patchType := admissionv1.PatchTypeJSONPatch
		review.Response = &admissionv1.AdmissionResponse{
			UID:       review.Request.UID,
			Allowed:   true,
			Patch:     patch,
			PatchType: &patchType,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(review); err != nil {
			log.Printf("failed to encode response: %v", err)
		}
	})

	log.Printf("webhook listening on %s", addr)
	log.Fatal(http.ListenAndServeTLS(addr, certFile, keyFile, nil))
}

func buildPatch(secretName string, rawObj []byte) []byte {
	// Detect whether /spec/volumes and /spec/containers/0/volumeMounts already exist
	// so we use "add" with the correct path (array append vs field init).
	var pod struct {
		Spec struct {
			Volumes    []interface{} `json:"volumes"`
			Containers []struct {
				VolumeMounts []interface{} `json:"volumeMounts"`
			} `json:"containers"`
		} `json:"spec"`
	}
	_ = json.Unmarshal(rawObj, &pod)

	volumesPath := "/spec/volumes/-"
	if len(pod.Spec.Volumes) == 0 {
		volumesPath = "/spec/volumes"
	}
	mountsPath := "/spec/containers/0/volumeMounts/-"
	if len(pod.Spec.Containers) == 0 || len(pod.Spec.Containers[0].VolumeMounts) == 0 {
		mountsPath = "/spec/containers/0/volumeMounts"
	}

	volumeValue := map[string]interface{}{
		"name": "rt-bootstrapper-certs",
		"secret": map[string]interface{}{
			"secretName": secretName,
		},
	}
	mountValue := map[string]interface{}{
		"name":      "rt-bootstrapper-certs",
		"mountPath": "/etc/ssl/certs",
		"readOnly":  true,
	}

	var patch []map[string]interface{}
	if len(pod.Spec.Volumes) == 0 {
		patch = append(patch, map[string]interface{}{"op": "add", "path": volumesPath, "value": []interface{}{volumeValue}})
	} else {
		patch = append(patch, map[string]interface{}{"op": "add", "path": volumesPath, "value": volumeValue})
	}
	if len(pod.Spec.Containers) == 0 || len(pod.Spec.Containers[0].VolumeMounts) == 0 {
		patch = append(patch, map[string]interface{}{"op": "add", "path": mountsPath, "value": []interface{}{mountValue}})
	} else {
		patch = append(patch, map[string]interface{}{"op": "add", "path": mountsPath, "value": mountValue})
	}

	b, _ := json.Marshal(patch)
	return b
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
