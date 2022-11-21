package main

import (
	"log"
	"os/exec"
	"strings"
	"testing"
)

func TestEndToEnd(t *testing.T) {
	log.Println("Starting E2E test...")

	log.Println("kubectl get crd")
	c, b := exec.Command("kubectl", "get", "crd"), new(strings.Builder)

	c.Stdout = b
	c.Run()
	out := b.String()

	log.Println(out)

	if len(out) == 0 {
		t.Error("No CRD found")
	}
}
