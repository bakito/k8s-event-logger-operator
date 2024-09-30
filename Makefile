# Include toolbox tasks
include ./.toolbox.mk


# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: tidy fmt generate mocks manifests test-ci

# Run tests
test-ci:
	go test ./... -coverprofile cover.out.tmp
	@cat cover.out.tmp | grep -v "zz_generated.deepcopy.go" > cover.out # filter coverage of generated code
	@rm -f cover.out.tmp

# Build manager binary
manager: generate fmt
	go build -o bin/manager main.go

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/..." paths="./api/..." paths="./controllers/..." output:crd:artifacts:config=config/crd/bases
	cp config/crd/bases/*.yaml helm/crds/
	yq -i '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.imagePullSecrets.items.properties.name.description="Name of the referent."' helm/crds/eventlogger.bakito.ch_eventloggers.yaml

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: golangci-lint
	$(LOCALBIN)/golangci-lint run --fix

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
	$(GORELEASER) --clean

.PHONY: test-release
test-release: goreleaser
	$(GORELEASER) --skip-publish --snapshot --clean

# generate mocks
.PHONY: mocks
mocks: mockgen
	$(MOCKGEN) -destination pkg/mocks/client/mock.go sigs.k8s.io/controller-runtime/pkg/client Client

	$(MOCKGEN) -destination pkg/mocks/logr/mock.go   github.com/go-logr/logr LogSink

.PHONY: lint-helm
lint-helm:
	helm lint helm/ --set webhook.enabled=true --set webhook.certManager.enabled=true

docs: helm-docs update-docs
	@$(LOCALBIN)/helm-docs

update-docs: semver
	@version=$$($(LOCALBIN)/semver -next); \
	versionNum=$$($(LOCALBIN)/semver -next -numeric); \
	sed -i "s/^version:.*$$/version: $${versionNum}/"    ./helm/Chart.yaml; \
	sed -i "s/^appVersion:.*$$/appVersion: $${version}/" ./helm/Chart.yaml

helm-lint: docs
	helm lint ./helm