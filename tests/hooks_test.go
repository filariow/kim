package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/cucumber/godog"
	"github.com/filariow/kim/tests/pkg/kube"
	cp "github.com/otiai10/copy"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func buildHookPrepareScenarioNamespace(k *kube.Kubernetes) func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	return func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		if _, ok := getContextNamespace(ctx); !ok {
			ctx = context.WithValue(ctx, kube.ContextNamespaceKey, fmt.Sprintf("kim-test-%s", sc.Id))
		}

		n, _ := getContextNamespace(ctx)
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: n,
			},
		}
		_, err := k.Cli.CoreV1().Namespaces().Create(ctx, &ns, metav1.CreateOptions{})
		return ctx, err
	}
}

func hookPrepareScenarioTestFolder(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	// prepare folder
	tf := path.Join("..", "..", sc.Id)
	if itf, err := os.Stat(tf); err != nil {
		if !os.IsNotExist(err) {
			return ctx, err
		}
	} else {
		if !itf.IsDir() {
			return ctx, fmt.Errorf("expected %s to be a temporary folder, found a file", tf)
		}
		if err := os.RemoveAll(tf); err != nil {
			return ctx, err
		}
	}

	if err := os.MkdirAll(tf, os.ModePerm); err != nil {
		return ctx, err
	}

	opts := cp.Options{
		OnDirExists: func(src, dest string) cp.DirExistsAction {
			return cp.Replace
		},
		PreserveOwner: true,
	}
	if err := cp.Copy(path.Join("..", "..", "base"), tf, opts); err != nil {
		return ctx, err
	}
	if err := os.Chdir(path.Join(tf, "tests")); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, TestFolderKey, tf), nil
}

func buildHookDestroyScenarioNamespace(k *kube.Kubernetes) func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	return func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		if err != nil {
			return ctx, err
		}

		ns, ok := getContextNamespace(ctx)
		if !ok {
			return ctx, err
		}

		if errDel := k.Cli.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{}); err != nil {
			if !kerrors.IsNotFound(errDel) {
				return ctx, errors.Join(errDel, err)
			}
		}

		return ctx, nil
	}
}

func hookDestroyScenarioTestFolder(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	if err != nil {
		return ctx, err
	}

	tf, ok := ctx.Value(TestFolderKey).(string)
	if !ok {
		return ctx, fmt.Errorf("can not find TestFolder in context: can not cleanup")
	}
	if err := os.RemoveAll(tf); err != nil {
		return ctx, fmt.Errorf("error cleaning up temp folder for test %s: %w", sc.Id, err)
	}
	return ctx, err
}

// utils
func getContextNamespace(ctx context.Context) (string, bool) {
	if vs, ok := ctx.Value(kube.ContextNamespaceKey).(string); ok {
		return vs, ok
	}
	return "", false
}
