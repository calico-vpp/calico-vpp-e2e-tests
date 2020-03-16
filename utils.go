package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func applyYaml(yaml string) (err error) {
	c := exec.Command("kubectl", "apply", "-f", "-")
	c.Stdin = strings.NewReader(yaml)
	output, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error applying yaml: error: %v, output: %s", err, output)
	}
	return nil
}

func waitForPod(ctx context.Context, namespace, pod string) (err error) {

}
