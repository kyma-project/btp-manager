package main

import (
	"log"
	"os/exec"
	"strings"
	"testing"
)

func TestEndToEnd(t *testing.T) {
	log.Println("Starting E2E test...")

	// Check that the BTPOperator resource is created and has the expected status.
	log.Println("kubectl get BTPOperator")
	c, b := exec.Command("kubectl", "get", "BTPOperator"), new(strings.Builder)

	c.Stdout = b
	c.Run()
	out := b.String()

	log.Println(out)

	if !strings.Contains(out, "Running") {
		t.Error("BTPOperator is not in Running state")
	}

	// Check that the btp-operator-controller is created and has the expected status.
	log.Println("kubectl get btp-operator-controller")
	c, b = exec.Command("kubectl", "get", "btp-operator-controller"), new(strings.Builder)

	c.Stdout = b
	c.Run()
	out = b.String()

	log.Println(out)

	if !strings.Contains(out, "Running") {
		t.Error("btp-operator-controller is not in Running state")
	}

	// Create a dummy serviceInstance using the dummy secret and check that it is in error state.
	log.Println("kubectl create -f deployments/prerequisites.yaml")
	c, b = exec.Command("kubectl", "create", "-f", "deployments/prerequisites.yaml"), new(strings.Builder)

	c.Stdout = b
	c.Run()
	out = b.String()

	log.Println(out)

	log.Println("kubectl get serviceInstance")
	c, b = exec.Command("kubectl", "get", "serviceInstance"), new(strings.Builder)

	c.Stdout = b
	c.Run()
	out = b.String()

	log.Println(out)

	if !strings.Contains(out, "Error") {
		t.Error("serviceInstance is not in Error state")
	}
}
