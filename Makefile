# Current Operator version

CONFIG_FILE=manifests/config.yaml
VERSION = $(shell yq e ".operator.version" $(CONFIG_FILE))
RELEASE = $(shell yq e ".operator.release" $(CONFIG_FILE))
BUNDLE_VERSION = $(shell yq e ".bundle.version" $(CONFIG_FILE))
BUNDLE_RELEASE = $(shell yq e ".bundle.release" $(CONFIG_FILE))
CHANNELS = $(shell yq e ".operator.channel" $(CONFIG_FILE))
DEFAULT_CHANNEL = $(CHANNELS)

# Default bundle image tag
BUNDLE_IMG ?= nvmesh-operator-bundle:$(BUNDLE_VERSION)-$(BUNDLE_RELEASE)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?= excelero/nvmesh-operator:$(VERSION)-$(RELEASE)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager manifests bundle

# Run tests
test:
	go test ./... -test.short

test-short-verbose:
	go test ./... -test.v -test.short -coverprofile coverage.out

show-coverage-report:
	go tool cover -html=coverage.out

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test-full: generate fmt vet manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/master/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

# go build
build: generate fmt vet
	go build ./...

# Build manager binary
manager: generate fmt vet
	go build -o bin/controller main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go $(args)

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
# make run args="-openshift -image-pull-always"
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen docs
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	cp config/crd/bases/nvmesh.excelero.com_nvmeshes.yaml manifests/bases/crd/nvmesh.crd.yaml
	cd manifests && ./build_manifests.py

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG} --build-arg VERSION=$(VERSION) --build-arg RELEASE=$(RELEASE)

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests
	# operator-sdk generate kustomize manifests -q
	# cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	# $(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./operator-hub/catalog_bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build: manifests
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# Build the bundle image and Custom Source index image
# and push both to DockerHub
.PHONY: bundle-dev-build
bundle-dev-build: manifests docker-build
	cd operator-hub && ./build_dev_catalog_images.sh

# Deploy the customn source in the cluster
.PHONY: bundle-dev-deploy
bundle-dev-deploy:
	kubectl apply -f operator-hub/dev/catalog_source.yaml

.PHONY: bundle-test
bundle-test:
	kubectl delete namespace test-operator || echo "namespace test-operator doesn't exists, proceeding with tests"
	kubectl create namespace test-operator
	kubectl create -f .secrets/openshift_scan_object.yaml -n test-operator
	date && kubectl apply -f operator-hub/dev/cr_sample.yaml -n test-operator
	sleep 1
	date && kubectl delete -f operator-hub/dev/cr_sample.yaml -n test-operator

.PHONY: list
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$'

.PHONY: debug-info
debug-info:
	echo $(VERSION)-$(RELEASE) $(DEFAULT_CHANNEL)

.PHONY: scorecard
scorecard:
	operator-sdk scorecard operator-hub/catalog_bundle/ -o text

.PHONY: docs
docs:
	cd docs/build_docs/ && ./gen_nvmesh_cr_docs.sh
