package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	defaultCNIPath       = "../cni-plugin"
	defaultManagerPath   = "../vpp-manager"
	defaultCalicoVPPPath = "../calico-vpp"
	defaultVagrantPath   = "../k8s-vagrant-multi-node"

	cniPathEnvVar       = "CALICOVPP_TEST_CNI_PLUGIN_PATH"
	cniBranchEnvVar     = "CALICOVPP_TEST_CNI_PLUGIN_BRANCH"
	nodePathEnvVar      = "CALICOVPP_TEST_CALICOVPP_PATH"
	nodeBranchEnvVar    = "CALICOVPP_TEST_CALICOVPP_BRANCH"
	managerPathEnvVar   = "CALICOVPP_TEST_VPP_MANAGER_PATH"
	managerBranchEnvVar = "CALICOVPP_TEST_VPP_MANAGER_BRANCH"
	vagrantPathEnvVar   = "CALICOVPP_TEST_VAGRANT_PATH"
	vagrantBranchEnvVar = "CALICOVPP_TEST_VAGRANT_BRANCH"

	cniImageName     = "calico/cni:latest"
	nodeImageName    = "calicovpp/node:latest"
	managerImageName = "calicovpp/vpp:latest"

	calicovppYamlPath = "etc/k8s/calico-vpp.yaml"
)

func TestMain(m *testing.M) {
	var err error
	var nodePath, vagrantPath string

	flag.Parse()

	err, nodePath = createImages()
	if err != nil {
		logrus.Errorf("Error preparing images: %v", err)
		os.Exit(1)
	}

	err, vagrantPath = createCluster()
	if err != nil {
		logrus.Errorf("Error preparing cluster: %v", err)
		os.Exit(1)
	}

	err = pushImages(vagrantPath)
	if err != nil {
		logrus.Errorf("Error pushing images: %v", err)
		os.Exit(1)
	}

	err = deployCalicoVPP(nodePath)
	if err != nil {
		logrus.Errorf("Error deploying calico-vpp: %v", err)
		os.Exit(1)
	}

	result := m.Run()

	err = cleanupCluster(nodePath)
	if err != nil {
		logrus.Warnf("Error cleaning up tests: %v", err)
	}

	os.Exit(result)
}

func createImages() (err error, nodePath string) {
	logrus.Info("Building images")
	err = createCNIImage()
	if err != nil {
		return errors.Wrap(err, "Error creating CNI image"), ""
	}

	err, nodePath = createNodeImage()
	if err != nil {
		return errors.Wrap(err, "Error creating node image"), ""
	}

	err = createVPPImage()
	if err != nil {
		return errors.Wrap(err, "Error creating node image"), ""
	}

	return nil, nodePath
}

func createCNIImage() (err error) {
	logrus.Info("Building CNI image")
	cniPath := os.Getenv(cniPathEnvVar)
	if cniPath == "" {
		cniPath = defaultCNIPath
	}

	cniBranch := os.Getenv(cniBranchEnvVar)
	if cniBranch != "" {
		if err = moveToBranch(cniPath, cniBranch); err != nil {
			return errors.Wrapf(err, "Error checking out desired CNI branch %s in path %s", cniBranch, cniPath)
		}
	}

	err = make(cniPath, "image")
	if err != nil {
		return errors.Wrap(err, "Error making CNI image")
	}

	return nil
}

func createNodeImage() (err error, nodePath string) {
	logrus.Info("Building calicovpp/node image")
	nodePath = os.Getenv(nodePathEnvVar)
	if nodePath == "" {
		nodePath = defaultCalicoVPPPath
	}

	nodeBranch := os.Getenv(nodeBranchEnvVar)
	if nodeBranch != "" {
		if err = moveToBranch(nodePath, nodeBranch); err != nil {
			return errors.Wrapf(err, "Error checking out desired calico-vpp branch %s in path %s", nodeBranch, nodePath), ""
		}
	}

	err = make(nodePath, "image")
	if err != nil {
		return errors.Wrap(err, "Error making calico-vpp image"), ""
	}

	return nil, nodePath
}

func createVPPImage() (err error) {
	logrus.Info("Building calicovpp/vpp image")
	managerPath := os.Getenv(managerPathEnvVar)
	if managerPath == "" {
		managerPath = defaultManagerPath
	}

	managerBranch := os.Getenv(managerBranchEnvVar)
	if managerBranch != "" {
		if err = moveToBranch(managerPath, managerBranch); err != nil {
			return errors.Wrapf(err, "Error checking out desired vpp-manager branch %s in path %s", managerBranch, managerPath)
		}
	}

	err = make(managerPath, "image")
	if err != nil {
		return errors.Wrap(err, "Error making VPP image")
	}

	return nil
}

func make(path, target string) (err error) {
	c := exec.Command("make", "-C", path, target)
	output, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error running make: error: %v, make output: %s", err, output)
	}
	return nil
}

func moveToBranch(path, branch string) (err error) {
	c := exec.Command("git", "-C", path, "checkout", branch)
	output, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error checking out branch: error: %v, git output: %s", err, output)
	}
	return nil
}

func createCluster() (err error, vagrantPath string) {
	logrus.Info("Starting vagrant k8s cluster")
	vagrantPath = os.Getenv(vagrantPathEnvVar)
	if vagrantPath == "" {
		vagrantPath = defaultVagrantPath
	}

	vagrantBranch := os.Getenv(vagrantBranchEnvVar)
	if vagrantBranch != "" {
		if err = moveToBranch(vagrantPath, vagrantBranch); err != nil {
			return errors.Wrapf(err, "Error checking out desired vagrant branch %s in path %s", vagrantPath, vagrantBranch), ""
		}
	}

	cmd := fmt.Sprintf("cd %s; source calicovpp.env; make up", vagrantPath)
	c := exec.Command("bash", "-c", cmd)
	output, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error starting cluster: error: %v, output: %s", err, output), ""
	}
	return nil, vagrantPath
}

func pushImages(vagrantPath string) (err error) {
	logrus.Info("Deploying docker images")
	for _, image := range []string{cniImageName, nodeImageName, managerImageName} {
		if err := pushImage(vagrantPath, image); err != nil {
			return errors.Wrapf(err, "Failed to push image %s", image)
		}
	}
	return nil
}

func pushImage(vagrantPath, image string) (err error) {
	logrus.Infof("Deploying %s image", image)
	cmd := fmt.Sprintf("cd %s; source calicovpp.env; IMG=%s make load-image -j3", vagrantPath, image)
	c := exec.Command("bash", "-c", cmd)
	output, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error pushing image: error: %v, output: %s", err, output)
	}
	return nil
}

func deployCalicoVPP(nodePath string) (err error) {
	logrus.Info("Deploying calico-vpp yaml")
	yamlPath := filepath.Join(nodePath, calicovppYamlPath)
	c := exec.Command("kubectl", "apply", "-f", yamlPath)
	output, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error deploying yaml: error: %v, output: %s", err, output)
	}
	return nil
}

func cleanupCluster(nodePath string) (err error) {
	logrus.Info("Cleaning up")
	yamlPath := filepath.Join(nodePath, calicovppYamlPath)
	c := exec.Command("kubectl", "delete", "-f", yamlPath)
	output, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error deploying yaml: error: %v, output: %s", err, output)
	}
	return nil
}
