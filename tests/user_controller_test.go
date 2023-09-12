package tests

import (
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

type UserFeature struct {
	*Kubernetes
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	k := NewKubernetesFromEnvOrDie()
	uf := UserFeature{Kubernetes: k}

	ctx.Step(`^Resource is created:$`, uf.resourcesAreCreated)
	ctx.Step(`^Resources are created:$`, uf.resourcesAreCreated)

	ctx.Step(`^Resource exists:$`, uf.resourcesExist)
	ctx.Step(`^Resources exist:$`, uf.resourcesExist)

	ctx.Step(`^Resource doesn't exist:$`, uf.resourcesNotExist)
	ctx.Step(`^Resources don't exist:$`, uf.resourcesNotExist)
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
			Output:   colors.Colored(os.Stdout),
		},
	}
	if sc := suite.Run(); sc != 0 {
		t.Fatalf("non-zero status returned (%d), failed to run feature tests", sc)
	}
}
