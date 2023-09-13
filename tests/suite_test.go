package tests

import (
	"os"
	"path"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/filariow/kim/tests/pkg/kube"
	cp "github.com/otiai10/copy"
)

const TestFolderKey string = "TestFolder"

func TestFeatures(t *testing.T) {
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}

	if err := prepareDotTmpFolder(); err != nil {
		t.Fatalf("error preparing .tmp folder: %s", err.Error())
	}

	if err := os.Chdir(path.Join(".tmp", "base", "tests")); err != nil {
		t.Fatal(err)
	}

	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:      "pretty",
			Paths:       []string{"features"},
			TestingT:    t,
			Output:      colors.Colored(os.Stdout),
			Concurrency: 4,
		},
	}
	if sc := suite.Run(); sc != 0 {
		t.Fatalf("non-zero status returned (%d), failed to run feature tests", sc)
	}

	if err := os.RemoveAll(path.Join("..", "..", "..", ".tmp")); err != nil {
		t.Errorf("error cleaning up .tmp folder: %v", err)
	}
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	k := kube.NewKubernetesFromEnvOrDie()

	ctx.Step(`^Resource is created:$`, k.ResourcesAreCreated)
	ctx.Step(`^Resources are created:$`, k.ResourcesAreCreated)

	ctx.Step(`^Resource is updated:$`, k.ResourcesAreUpdated)
	ctx.Step(`^Resources are updated:$`, k.ResourcesAreUpdated)

	ctx.Step(`^Resource exists:$`, k.ResourcesExist)
	ctx.Step(`^Resources exist:$`, k.ResourcesExist)

	ctx.Step(`^Resource doesn't exist:$`, k.ResourcesNotExist)
	ctx.Step(`^Resources don't exist:$`, k.ResourcesNotExist)

	ctx.Step(`^Create context namespace "([\w]+[\w-]*)"$`, k.CreateContextNamespace)
	ctx.Step(`^KIM is deployed$`, k.KIMIsDeployed)

	// set and create the ContextNamespace
	ctx.Before(buildHookPrepareScenarioNamespace(k))

	// create temp folder for scenario
	ctx.Before(hookPrepareScenarioTestFolder)

	// delete the ContextNamespace if no errors occurred
	ctx.After(buildHookDestroyScenarioNamespace(k))

	// cleanup temp folder
	ctx.After(hookDestroyScenarioTestFolder)
}

func prepareDotTmpFolder() error {
	opts := cp.Options{
		OnDirExists: func(src, dest string) cp.DirExistsAction {
			return cp.Replace
		},
		PreserveOwner: true,
	}

	copyOverFunc := func(f ...string) func() error {
		return func() error {
			jf := path.Join(f...)
			return cp.Copy(jf, path.Join(".tmp", "base", jf), opts)
		}
	}
	todos := []func() error{
		func() error { return os.RemoveAll(".tmp/base") },
		copyOverFunc("bin", "controller-gen"),
		copyOverFunc("bin", "kustomize"),
		copyOverFunc("config"),
		copyOverFunc("tests", "features"),
		copyOverFunc("Makefile"),
	}
	for _, f := range todos {
		if err := f(); err != nil {
			return err
		}
	}

	return nil
}
