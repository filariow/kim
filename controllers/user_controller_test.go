package controllers

import (
	"context"
	"strings"

	"github.com/cucumber/godog"
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

type UserFeature struct {
	cli     kubernetes.Clientset
	dynamic dynamic.DynamicClient
}

func (uf *UserFeature) resourcesAreCreated(spec string) error {
	decoder := yamlutil.NewYAMLOrJSONDecoder(strings.NewReader(spec), 100)
	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			break
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return err
		}

		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		gr, err := restmapper.GetAPIGroupResources(uf.cli.Discovery())
		if err != nil {
			return err
		}

		mapper := restmapper.NewDiscoveryRESTMapper(gr)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		var dri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace("default")
			}
			dri = uf.dynamic.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = uf.dynamic.Resource(mapping.Resource)
		}

		if _, err := dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	uf := UserFeature{}

	ctx.Step(`^resource [is|are] created:$`, uf.resourcesAreCreated)
}
