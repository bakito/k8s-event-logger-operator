# Include toolbox tasks
include ./.toolbox.mk


# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd"

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

manifests: tb.controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(TB_CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./pkg/..." paths="./api/..." paths="./controllers/..." output:crd:artifacts:config=config/crd/bases
	cp config/crd/bases/*.yaml helm/crds/
	yq -i '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.imagePullSecrets.items.properties.name.description="Name of the referent."' helm/crds/eventlogger.bakito.ch_eventloggers.yaml

generate: tb.controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(TB_CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: tb.golangci-lint
	$(TB_GOLANGCI_LINT) run --fix

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
release: tb.goreleaser
	@version=$$($(TB_SEMVER)); \
	git tag -s $$version -m"Release $$version"
	$(TB_GORELEASER) --clean

.PHONY: test-release
test-release: tb.goreleaser
	$(TB_GORELEASER) --skip=publish --snapshot --clean

# generate mocks
.PHONY: mocks
mocks: tb.mockgen
	$(TB_MOCKGEN) -destination pkg/mocks/client/mock.go sigs.k8s.io/controller-runtime/pkg/client Client

	$(TB_MOCKGEN) -destination pkg/mocks/logr/mock.go   github.com/go-logr/logr LogSink

.PHONY: lint-helm
lint-helm:
	helm lint helm/ --set webhook.enabled=true --set webhook.certManager.enabled=true

docs: tb.helm-docs update-docs
	@$(TB_HELM_DOCS)

update-docs: tb.semver
	@version=$$($(TB_SEMVER) -next); \
	versionNum=$$($(TB_SEMVER) -next -numeric); \
	sed -i "s/^version:.*$$/version: $${versionNum}/"    ./helm/Chart.yaml; \
	sed -i "s/^appVersion:.*$$/appVersion: $${version}/" ./helm/Chart.yaml

helm-lint: docs
	helm lint ./helm