package testing

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func Kubectl(envs []string, args ...string) (string, error) {

	k := exec.Command("kubectl", args...)
	k.Env = os.Environ()
	for _, e := range envs {
		k.Env = append(k.Env, e)
	}
	out, err := k.CombinedOutput()
	output := string(out)
	output = strings.TrimSuffix(output, "\n")

	if err != nil {
		return output, err
	}
	return output, nil
}

type ContainerEnv struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type Container struct {
	Name string         `json:"name"`
	Envs []ContainerEnv `json:"env"`
}

type PodStatus struct {
	Phase string `json:"phase"`
}
type PodSpec struct {
	Containers []Container `json:"containers"`
}
type Pod struct {
	Spec      PodSpec   `json:"spec"`
	PodStatus PodStatus `json:"status"`
}

func (p *Pod) IsRunning() bool {
	if p.PodStatus.Phase == "Running" {
		return true
	}
	return false
}

func KubePodStatus(podname string) (*Pod, error) {
	str, err := Kubectl([]string{}, "get", "pod", podname, "-o", "json")
	if err != nil {
		return nil, errors.Wrap(err, "Failed: "+string(str))
	}

	var p Pod
	err = json.Unmarshal([]byte(str), &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func KubeClean() error {
	str, err := Kubectl([]string{}, "delete", "pod", "--all")
	if err != nil {
		return errors.Wrap(err, "Failed: "+string(str))
	}
	str, err = Kubectl([]string{}, "delete", "mutatingwebhookconfiguration", "--all")
	if err != nil {
		return errors.Wrap(err, "Failed: "+string(str))
	}

	str, err = Kubectl([]string{}, "delete", "secrets", "--all")
	if err != nil {
		return errors.Wrap(err, "Failed: "+string(str))
	}
	str, err = Kubectl([]string{}, "delete", "svc", "--all")
	if err != nil {
		return errors.Wrap(err, "Failed: "+string(str))
	}
	return nil
}
func KubeApply(b []byte) error {
	tmpdir, err := ioutil.TempDir(os.TempDir(), "service")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir) // clean up
	apply := filepath.Join(tmpdir, "apply.yaml")

	err = ioutil.WriteFile(apply, b, os.ModePerm)
	if err != nil {
		return err
	}
	out, err := Kubectl([]string{}, "apply", "-f", apply)
	if err != nil {
		return errors.Wrap(err, "Failed: "+string(out))
	}
	return nil
}
