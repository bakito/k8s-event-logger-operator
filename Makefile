
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
test: tidy test-ci fmt test-ci

# Run tests
test-ci: generate mocks manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt
	go build -o bin/manager main.go

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/..." paths="./api/..." paths="./controllers/..." output:crd:artifacts:config=config/crd/bases
	cp config/crd/bases/*.yaml helm/crds/

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
	$(GORELEASER) --rm-dist

.PHONY: test-release
test-release: goreleaser
	$(GORELEASER) --skip-publish --snapshot --rm-dist

# generate mocks
.PHONY: mocks
mocks: mockgen
	$(MOCKGEN) -destination pkg/mocks/client/mock.go sigs.k8s.io/controller-runtime/pkg/client Client

	$(MOCKGEN) -destination pkg/mocks/logr/mock.go   github.com/go-logr/logr LogSink

.PHONY: lint-helm
lint-helm:
	helm lint helm/ --set webhook.enabled=true --set webhook.certManager.enabled=true

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
SEMVER ?= $(LOCALBIN)/semver
HELM_DOCS ?= $(LOCALBIN)/helm-docs
MOCKGEN ?= $(LOCALBIN)/mockgen
GORELEASER ?= $(LOCALBIN)/goreleaser
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.10.0
SEMVER_VERSION ?= v1.1.3
HELM_DOCS_VERSION ?= v1.11.0
MOCKGEN_VERSION ?= v1.6.0
GORELEASER_VERSION ?= latest
GOLANGCI_LINT_VERSION ?= latest

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: semver
semver: $(SEMVER) ## Download semver locally if necessary.
$(SEMVER): $(LOCALBIN)
	test -s $(LOCALBIN)/semver || GOBIN=$(LOCALBIN) go install github.com/bakito/semver@$(SEMVER_VERSION)

.PHONY: helm-docs
helm-docs: $(HELM_DOCS) ## Download helm-docs locally if necessary.
$(HELM_DOCS): $(LOCALBIN)
	test -s $(LOCALBIN)/helm-docs || GOBIN=$(LOCALBIN) go install github.com/norwoodj/helm-docs/cmd/helm-docs@$(HELM_DOCS_VERSION)

.PHONY: mockgen
mockgen: $(MOCKGEN) ## Download mockgen locally if necessary.
$(MOCKGEN): $(LOCALBIN)
	test -s $(LOCALBIN)/mockgen|| GOBIN=$(LOCALBIN) go install github.com/golang/mock/mockgen@$(MOCKGEN_VERSION)

.PHONY: goreleaser
goreleaser: $(GORELEASER) ## Download goreleaser locally if necessary.
$(GORELEASER): $(LOCALBIN)
	test -s $(LOCALBIN)/goreleaser|| GOBIN=$(LOCALBIN) go install github.com/goreleaser/goreleaser@$(GORELEASER_VERSION)

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $(LOCALBIN)/golangci-lint|| GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

docs: helm-docs update-docs
	@$(LOCALBIN)/helm-docs

update-docs: semver
	@version=$$($(LOCALBIN)/semver -next); \
	versionNum=$$($(LOCALBIN)/semver -next -numeric); \
	sed -i "s/^version:.*$$/version: $${versionNum}/"    ./helm/Chart.yaml; \
	sed -i "s/^appVersion:.*$$/appVersion: $${version}/" ./helm/Chart.yaml

helm-lint: docs
	helm lint ./helm
