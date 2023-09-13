package kube

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/filariow/kim/tests/pkg/poll"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
	Cli *kubernetes.Clientset
	Dyn *dynamic.DynamicClient
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
		Cli: cli,
		Dyn: dyn,
	}, nil
}

func (k *Kubernetes) ParseResources(ctx context.Context, spec string) ([]unstructured.Unstructured, error) {
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

func (k *Kubernetes) BuildClientForResource(ctx context.Context, unstructuredObj unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	gr, err := restmapper.GetAPIGroupResources(k.Cli.Discovery())
	if err != nil {
		return nil, err
	}

	gvk := unstructuredObj.GroupVersionKind()
	mapper := restmapper.NewDiscoveryRESTMapper(gr)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	dri := k.Dyn.Resource(mapping.Resource)
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if unstructuredObj.GetNamespace() == "" {
			if ns, ok := ctx.Value(ContextNamespaceKey).(string); ok {
				unstructuredObj.SetNamespace(ns)
			}
			// TODO: take track of the resource in context for cleanup? fail?
		}
		return dri.Namespace(unstructuredObj.GetNamespace()), nil
	}

	return dri, nil
}

// steps
func (k *Kubernetes) ResourcesAreCreated(ctx context.Context, spec string) error {
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		dri, err := k.BuildClientForResource(ctx, u)
		if err != nil {
			return err
		}

		if _, err := dri.Create(ctx, &u, metav1.CreateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (k *Kubernetes) ResourcesAreUpdated(ctx context.Context, spec string) error {
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		dri, err := k.BuildClientForResource(ctx, u)
		if err != nil {
			return err
		}

		po, err := dri.Get(ctx, u.GetName(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		po.Object["spec"] = u.Object["spec"]
		if _, err := dri.Update(ctx, po, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (k *Kubernetes) ResourcesExist(ctx context.Context, spec string) error {
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		dri, err := k.BuildClientForResource(ctx, u)
		if err != nil {
			return err
		}

		if _, err := dri.Get(ctx, u.GetName(), metav1.GetOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (k *Kubernetes) ResourcesNotExist(ctx context.Context, spec string) error {
	uu, err := k.ParseResources(ctx, spec)
	if err != nil {
		return err
	}

	for _, u := range uu {
		dri, err := k.BuildClientForResource(ctx, u)
		if err != nil {
			return err
		}

		ctxd, cf := context.WithTimeout(ctx, 2*time.Minute)
		_, err = poll.DoR(ctxd, time.Second, func(ictx context.Context) (*unstructured.Unstructured, error) {
			lctx, lcf := context.WithTimeout(ictx, 1*time.Minute)
			defer lcf()

			if _, err := dri.Get(lctx, u.GetName(), metav1.GetOptions{}); err != nil {
				if kerrors.IsNotFound(err) {
					return nil, nil
				}
			}
			return nil, err
		})
		cf()

		if err != nil {
			ld, err := u.MarshalJSON()
			if err != nil {
				return fmt.Errorf(
					"resource exists: [ ApiVersion=%s, Kind=%s, Namespace=%s, Name=%s ]. Error marshaling as json: %w",
					u.GetAPIVersion(), u.GetKind(), u.GetNamespace(), u.GetName(), err)
			}
			return fmt.Errorf("resource exists: %s", ld)
		}
	}

	return nil
}

func (k *Kubernetes) CreateContextNamespace(ctx context.Context, namespace string) (context.Context, error) {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	if _, err := k.Cli.CoreV1().Namespaces().Create(ctx, &ns, metav1.CreateOptions{}); err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, ContextNamespaceKey, namespace), nil
}

func (k *Kubernetes) KIMIsDeployed(ctx context.Context) error {
	env := os.Environ()
	args := []string{"/usr/bin/make", "deploy", "wait-rollout", "-o", "install"}
	if vs, ok := ctx.Value(ContextNamespaceKey).(string); ok {
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
