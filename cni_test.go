package main

import "testing"

var singlePodSpec = ```
apiVersion: v1
kind: Pod
metadata:
  name: cni-test
  namespace: default
spec:
  containers:
  - name: main
    image: calicovpp/netshoot:latest
    command: ["bash"]
    args: ["-c, "sleep 36000"]
```

// Checks that a container starts, gets an interface, an IP address and a default route
func TestCNI(t *testing.T) {
	err := applyYaml(singlePodSpec)
	if err != nil {
		t.Errorf("Failed to create pod: %v", err)
	}
}


