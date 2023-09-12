package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	ctx.Step(`^Create context namespace "([\w]+[\w-]*)"$`, uf.createContextNamespace)
	ctx.Step(`^KIM is deployed$`, uf.kimIsDeployed)

	// set and create the ContextNamespace
	ctx.Before(
		func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
			if _, ok := getContextNamespace(ctx); !ok {
				ctx = context.WithValue(ctx, ContextNamespaceKey, fmt.Sprintf("kim-test-%s", sc.Id))
			}

			n, _ := getContextNamespace(ctx)
			ns := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: n,
				},
			}
			_, err := k.cli.CoreV1().Namespaces().Create(ctx, &ns, metav1.CreateOptions{})
			return ctx, err
		})

	// delete the ContextNamespace if no errors occurred
	ctx.After(
		func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
			if err != nil {
				return ctx, err
			}

			ns, ok := getContextNamespace(ctx)
			if !ok {
				return ctx, err
			}

			if errDel := k.cli.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{}); err != nil {
				if !kerrors.IsNotFound(errDel) {
					return ctx, errors.Join(errDel, err)
				}
			}

			return ctx, nil
		})
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
