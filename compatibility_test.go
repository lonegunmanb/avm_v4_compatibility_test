package avm_compatibility_test

import (
	"github.com/ahmetb/go-linq/v3"
	"github.com/go-git/go-git/v5"
	"github.com/gruntwork-io/terratest/modules/terraform"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func GetAvmRepos() []string {
	return []string{
		"https://github.com/Azure/terraform-azurerm-avm-res-keyvault-vault",
	}
}

func TestCompatibility(t *testing.T) {
	repos := GetAvmRepos()
	if repo := os.Getenv("TARGET_REPO_URL"); repo != "" {
		repos = []string{repo}
	}
	linq.From(repos).Select(func(i interface{}) interface{} {
		return i.(string) + ".git"
	}).ToSlice(&repos)

	for _, repo := range repos {
		t.Run(repo, func(t *testing.T) {
			compatibilityTestForRepo(t, repo)
		})
	}
}

func compatibilityTestForRepo(t *testing.T, repo string) {
	path := cloneRepo(t, repo)
	defer os.RemoveAll(path)
	examples, err := os.ReadDir(filepath.Join(path, "examples"))
	require.NoError(t, err)
	for _, example := range examples {
		if example.IsDir() {
			t.Run(example.Name(), func(t *testing.T) {
				compatibilityTestForExample(t, path, example.Name())
			})
		}
	}
}

func compatibilityTestForExample(t *testing.T, path string, name string) {
	examplePath := filepath.Join(path, "examples", name)
	pwd, err := os.Getwd()
	require.NoError(t, err)
	tfOpt := &terraform.Options{
		TerraformDir: examplePath,
		Upgrade:      true,
	}
	_, err = terraform.InitE(t, tfOpt)
	n := err == nil
	require.True(t, n)
	require.NoError(t, err)
	mapotf(t, examplePath, "transform", "-r", "--tf-dir", examplePath, "--mptf-dir", filepath.Join(pwd, "mapotf"))
	_, err = terraform.InitE(t, tfOpt)
	require.NoError(t, err)
	_, err = terraform.ValidateE(t, tfOpt)
	assert.NoError(t, err)
	mapotf(t, examplePath, "reset", "-r", "--tf-dir", examplePath)
	_, err = terraform.InitE(t, tfOpt)
	require.NoError(t, err)
	_, err = terraform.ApplyE(t, tfOpt)
	require.NoError(t, err)
	defer terraform.Destroy(t, tfOpt)
	mapotf(t, examplePath, "transform", "-r", "--tf-dir", examplePath, "--mptf-dir", filepath.Join(pwd, "mapotf"))
	defer mapotf(t, examplePath, "reset", "-r", "--tf-dir", examplePath)
	_, err = terraform.InitE(t, tfOpt)
	require.NoError(t, err)
	tfOpt.PlanFilePath = filepath.Join(tfOpt.TerraformDir, "tf.plan")
	exitCode := terraform.PlanExitCode(t, tfOpt)
	plan := terraform.InitAndPlanAndShowWithStruct(t, tfOpt)
	changes := plan.ResourceChangesMap
	assert.True(t, exitCode == 0 || noChange(changes))
}

func noChange(changes map[string]*tfjson.ResourceChange) bool {
	if len(changes) == 0 {
		return true
	}
	return linq.From(changes).Select(func(i interface{}) interface{} {
		return i.(linq.KeyValue).Value
	}).All(func(i interface{}) bool {
		change := i.(*tfjson.ResourceChange).Change
		if change == nil {
			return true
		}
		if change.Actions == nil {
			return true
		}
		return change.Actions.NoOp()
	})
}

func cloneRepo(t *testing.T, repo string) string {
	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	_, err = git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL: repo,
	})
	require.NoError(t, err)
	return tmpDir
}

func mapotf(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("mapotf", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run())
}
