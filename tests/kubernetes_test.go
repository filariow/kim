package tests

import (
	"context"
	"fmt"
	"strings"

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
			unstructuredObj.SetNamespace("default")
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
