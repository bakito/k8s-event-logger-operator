
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,crdVersions=v1beta1"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: generate mocks tidy fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && kustomize edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	cp config/crd/bases/*.yaml helm/crds/

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Run go fmt against code
fmt:
	go fmt ./...
	gofmt -s -w .


# Run go vet against code
vet:
	go vet ./...

# go mod tidy
tidy:
	go mod tidy

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

.PHONY: release
release: goreleaser
	@version=$$(semver); \
	git tag -s $$version -m"Release $$version"
	$(GORELEASER) --rm-dist

.PHONY: test-release
test-release: goreleaser
	$(GORELEASER) --skip-publish --snapshot --rm-dist

# generate mocks
.PHONY: mocks
mocks: mockgen
	$(MOCKGEN) -destination pkg/mocks/client/mock.go sigs.k8s.io/controller-runtime/pkg/client Client

	$(MOCKGEN) -destination pkg/mocks/logr/mock.go   github.com/go-logr/logr Logger

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

GORELEASER = $(shell pwd)/bin/goreleaser
goreleaser: ## Download goreleaser locally if necessary.
	$(call go-get-tool,$(GORELEASER),github.com/goreleaser/goreleaser)

MOCKGEN = $(shell pwd)/bin/mockgen
mockgen: ## Download mockgen locally if necessary.
	$(call go-get-tool,$(MOCKGEN),github.com/golang/mock/mockgen@v1.6.0)