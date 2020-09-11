package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func exitOnErr(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %#v", msg, err)
	}
}

func GetConfig() *restclient.Config {
	usr, err := user.Current()
	exitOnErr("Failed to get current user", err)

	userKubeConfigPath := filepath.Join(usr.HomeDir, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", userKubeConfigPath)
	exitOnErr("Failed to Build client config", err)

	return config
}

func GetDynamicClient(config *restclient.Config) dynamic.Interface {
	client, err := dynamic.NewForConfig(config)
	exitOnErr("Failed to get dynamic client", err)

	return client
}

func GetStructuredClient(config *restclient.Config) client.Client {
	c, err := client.New(config, client.Options{})
	exitOnErr("Failed to get structured client", err)
	return c
}

// find the corresponding GVR (available in *meta.RESTMapping) for gvk
func findGVR(gvk *schema.GroupVersionKind, cfg *rest.Config) (*meta.RESTMapping, error) {

	// DiscoveryClient queries API server about the resources
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	return mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
}

func main() {
	config := GetConfig()

	// Structured Attempt
	serviceAccount := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "test-dynamic-client", Namespace: "default"},
	}
	sc := GetStructuredClient(config)

	sc.Delete(context.TODO(), &serviceAccount)
	err := sc.Create(context.TODO(), &serviceAccount)
	exitOnErr("Failed to create object", err)
	sc.Delete(context.TODO(), &serviceAccount)

	// Dynamic Attempt
	// decode YAML into unstructured.Unstructured
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	obj := &unstructured.Unstructured{}
	const saManifest = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-dynamic-client
  namespace: default
`
	_, gvk, err := dec.Decode([]byte(saManifest), nil, obj)
	exitOnErr("Failed to decode yaml into unstructured", err)

	// Get the common metadata, and show GVK
	fmt.Printf("%#v\n", gvk)

	// encode back to JSON
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	enc.Encode(obj)

	dc := GetDynamicClient(config)

	gvrMapping, err := findGVR(gvk, config)
	exitOnErr("Failed to find GroupVersionResource for object", err)

	res := dc.Resource(gvrMapping.Resource)
	_, err = res.Create(context.TODO(), obj, metav1.CreateOptions{})
	exitOnErr("Failed to create object", err)

	fmt.Println("Success!")
}
