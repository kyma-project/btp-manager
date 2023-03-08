package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func TestEndToEnd(t *testing.T) {
	log.Println("Starting end-to-end test")

	cmd := exec.Command("make", "-v")
	err := cmd.Run()
	if err != nil {
		t.Fatal("make is not installed")
	}

	cmd = exec.Command("kubectl", "version")
	err = cmd.Run()
	if err != nil {
		t.Fatal("kubectl is not installed")
	}

	err = exec.Command("make", "-C", "../../", "install").Run()
	if err != nil {
		t.Errorf("Error running command 'make install': %v", err)
	}

	prNumber := os.Getenv("PRNUMBER")
	if prNumber == "" {
		t.Error("PRNUMBER env variable is not set")
	}

	_, err = strconv.Atoi(prNumber)
	if err != nil {
		t.Errorf("PRNUMBER env variable is not a number: %v", err)
	}

	exec.Command("kubectl", "create", "namespace", "kyma-system").Run()

	// expected to return exit status 2 as Prometheus is not installed, hence error suppressed
	exec.Command("make", "-C", "../../", "deploy",
		"IMG=europe-docker.pkg.dev/kyma-project/dev/btp-manager:PR-"+prNumber).Run()

	out, err := exec.Command("kubectl", "rollout", "status", "--namespace=kyma-system",
		"deployment/btp-manager-controller-manager", "--timeout=300s").Output()
	fmt.Println(string(out))
	if err != nil {
		t.Errorf("Error running command 'kubectl rollout status --namespace=kyma-system deployment/btp-manager"+
			"-controller-manager --timeout=60s': %s", err)
	}

	out, err = exec.Command("kubectl", "apply", "-f", "../../deployments/prerequisites.yaml", "-f",
		"../../examples/btp-manager-secret.yaml", "-f", "../../examples/btp-operator.yaml").Output()
	fmt.Println(string(out))
	if err != nil {
		t.Errorf("Error running command 'kubectl apply -f deployments/prerequisites.yaml -f examples/btp-manager-secret.yaml -f examples/btp-operator.yaml': %v", err)
	}

	out, err = exec.Command("kubectl", "get", "priorityclass", "kyma-system").Output()
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
