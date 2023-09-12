package tests

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
)

const ContextNamespaceKey string = "TestNamespace"

type Kubernetes struct {
	cli *kubernetes.Clientset
	dyn *dynamic.DynamicClient
}

func NewKubernetesFromEnvOrDie() *Kubernetes {
	k, err := NewKubernetesFromEnv()
	if err != nil {
		panic(err)
	}
	return k
}

func NewKubernetesFromEnv() (*Kubernetes, error) {
	cfg, err := GetRESTConfig()
	if err != nil {
		return nil, err
	}

	cli := kubernetes.NewForConfigOrDie(cfg)
	dyn := dynamic.NewForConfigOrDie(cfg)

	return &Kubernetes{
		cli: cli,
		dyn: dyn,
	}, nil
}

func (k *Kubernetes) parseResources(ctx context.Context, spec string) ([]unstructured.Unstructured, error) {
	uu := []unstructured.Unstructured{}
	decoder := yamlutil.NewYAMLOrJSONDecoder(strings.NewReader(spec), 100)
	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			break
		}

		obj, _, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, rawObj.Raw)
		}

		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return nil, err
		}

		unstructuredObj := unstructured.Unstructured{Object: unstructuredMap}
		uu = append(uu, unstructuredObj)
	}

	return uu, nil
}

func (k *Kubernetes) buildClientForResource(ctx context.Context, unstructuredObj unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	gr, err := restmapper.GetAPIGroupResources(k.cli.Discovery())
	if err != nil {
		return nil, err
	}

	gvk := unstructuredObj.GroupVersionKind()
	mapper := restmapper.NewDiscoveryRESTMapper(gr)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	dri := k.dyn.Resource(mapping.Resource)
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if unstructuredObj.GetNamespace() == "" {
			if ns, ok := getContextNamespace(ctx); ok {
				unstructuredObj.SetNamespace(ns)
			}
			// TODO: take track of the resource in context for cleanup? fail?
		}
		return dri.Namespace(unstructuredObj.GetNamespace()), nil
	}

	return dri, nil
}

// steps
func (k *Kubernetes) resourcesAreCreated(ctx context.Context, spec string) error {
	uu, err := k.parseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		dri, err := k.buildClientForResource(ctx, u)
		if err != nil {
			return err
		}

		if _, err := dri.Create(context.Background(), &u, metav1.CreateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (k *Kubernetes) resourcesExist(ctx context.Context, spec string) error {
	uu, err := k.parseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		dri, err := k.buildClientForResource(ctx, u)
		if err != nil {
			return err
		}

		if _, err := dri.Get(context.Background(), u.GetName(), metav1.GetOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (k *Kubernetes) resourcesNotExist(ctx context.Context, spec string) error {
	uu, err := k.parseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		dri, err := k.buildClientForResource(ctx, u)
		if err != nil {
			return err
		}

		if _, err := dri.Get(context.Background(), u.GetName(), metav1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return err
		} else {
			ld, err := u.MarshalJSON()
			if err != nil {
				return fmt.Errorf(
					"resource exists: [ ApiVersion=%s, Kind=%s, Namespace=%s, Name=%s ]. Error marshaling as json: %w",
					u.GetAPIVersion(), u.GetKind(), u.GetNamespace(), u.GetName(), err)
			}
			return fmt.Errorf("resource exists: %s", ld)
		}
	}

	return nil
}

func (k *Kubernetes) createContextNamespace(ctx context.Context, namespace string) (context.Context, error) {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	if _, err := k.cli.CoreV1().Namespaces().Create(ctx, &ns, metav1.CreateOptions{}); err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, ContextNamespaceKey, namespace), nil
}

func (k *Kubernetes) kimIsDeployed(ctx context.Context) error {
	env := os.Environ()
	args := []string{"/usr/bin/make", "deploy", "wait-rollout"}
	if vs, ok := getContextNamespace(ctx); ok {
		args = append(args, fmt.Sprintf("NAMESPACE=%s", vs))
	}

	var buf bytes.Buffer

	cmd := exec.Cmd{
		Path:   "/usr/bin/make",
		Args:   args,
		Dir:    "../",
		Env:    env,
		Stdout: &buf,
		Stderr: &buf,
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running cmd '%s': %w\n%s", cmd.String(), err, buf.String())
	}
	return nil
}

// utils
func getContextNamespace(ctx context.Context) (string, bool) {
	if vs, ok := ctx.Value(ContextNamespaceKey).(string); ok {
		return vs, ok
	}
	return "", false
}
