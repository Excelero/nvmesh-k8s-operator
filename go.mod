module excelero.com/nvmesh-k8s-operator

go 1.12

require (
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/openshift/api v3.9.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/common v0.6.0
	k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v0.18.8
	k8s.io/utils v0.0.0-20200603063816-c1c6865ac451
	sigs.k8s.io/controller-runtime v0.6.2
)
