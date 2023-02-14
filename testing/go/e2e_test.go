package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const (
	run_module_image_script_name = "../../hack/run_module_image.sh"
	helm_uninstall_invocation    = "helm uninstall btp-manager"
	image_name_env               = "IMAGE_NAME"
)

func TestEndToEnd(t *testing.T) {
	log.Println("Starting end-to-end test")

	checkPrerequisites(t)
	imageReference := getImageName(t)

	out, err := exec.Command(run_module_image_script_name, imageReference).Output()
	if err != nil {
		t.Fatalf("Expected script %s to be run successfully, but got error: %v and output: \n%s", run_module_image_script_name, err, out)
	}

	checkResourcesExistence(t, err)

	err = exec.Command(helm_uninstall_invocation).Run()
	if err != nil {
		t.Fatalf("Expected 'helm uninstall' successfully, but got error: %v", err)
	}
}

func checkResourcesExistence(t *testing.T, err error) {
	log.Println("Checking resources")

	out, err := exec.Command("kubectl", "get", "priorityclass", "kyma-system").Output()
	fmt.Println(string(out))
	if err != nil {
		t.Errorf("Expected priorityclass kyma-system to exist, but got error: %v", err)
	}
	if !strings.Contains(string(out), "kyma-system") {
		t.Errorf("Expected output 'kyma-system', but got: %s", string(out))
	}

	out, err = exec.Command("kubectl", "get", "secret", "sap-btp-manager", "-n", "kyma-system").Output()
	fmt.Println(string(out))
	if err != nil {
		t.Errorf("Expected secret sap-btp-manager in namespace kyma-system to exist, but got error: %v", err)
	}
	if !strings.Contains(string(out), "sap-btp-manager") {
		t.Errorf("Expected output 'sap-btp-manager', but got: %s", string(out))
	}

	out, err = exec.Command("kubectl", "get", "btpoperator", "btpoperator-sample").Output()
	if err != nil {
		t.Errorf("Expected btpoperator btpoperator-sample to exist, but got error: %v", err)
	}

	out, err = exec.Command("kubectl", "get", "btpoperator", "btpoperator-sample").Output()
	if err != nil {
		t.Errorf("Expected btpoperator btpoperator-sample to exist, but got error: %v", err)
	}
	fmt.Println(string(out))
}

func checkPrerequisites(t *testing.T) {
	log.Println("Checking prerequisites")

	out, err := exec.Command("helm", "version").Output()
	if err != nil {
		t.Fatal("helm is not installed")
	}

	log.Printf("helm version:%s", out)

	cmd := exec.Command("kubectl", "version")
	err = cmd.Run()
	if err != nil {
		t.Fatal("kubectl is not installed")
	}
}

func getImageName(t *testing.T) string {
	log.Println("Getting image name from the environment")

	imageName := os.Getenv(image_name_env)
	if imageName == "" {
		t.Fatalf("%s env variable is not set", image_name_env)
	}
	return imageName
}
