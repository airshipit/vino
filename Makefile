# Image URL to use all building/pushing image targets
# IMG ?= controller:latest
CONTROLLER_IMG ?= quay.io/airshipit/vino
NODE_LABELER_IMG ?= quay.io/airshipit/nodelabeler

# Produce CRDs that work back to Kubernetes 1.16
CRD_OPTIONS ?= crd:crdVersions=v1

TOOLBINDIR          := tools/bin

API_REF_GEN_VERSION = v0.3.0
CONTROLLER_GEN_VERSION = v0.3.0

# linting
LINTER              := $(TOOLBINDIR)/golangci-lint
LINTER_CONFIG       := .golangci.yaml

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Docker proxy flags
DOCKER_PROXY_FLAGS  := --build-arg http_proxy=$(HTTP_PROXY)
DOCKER_PROXY_FLAGS  += --build-arg https_proxy=$(HTTPS_PROXY)
DOCKER_PROXY_FLAGS  += --build-arg NO_PROXY=$(NO_PROXY)

all: manager

# Run tests
test: generate fmt vet manifests lint api-docs
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kustomize build config/default | kubectl apply -f -
# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the controller docker image
# If DOCKER_PROXY_FLAGS values are empty, we are fine with that
docker-build-controller:
	docker build ${DOCKER_PROXY_FLAGS} . -t ${CONTROLLER_IMG}

# Build the nodelabeler docker image
# If DOCKER_PROXY_FLAGS values are empty, we are fine with that
docker-build-nodelabeler:
	docker build -f nodelabeler/Dockerfile . ${DOCKER_PROXY_FLAGS} -t ${NODE_LABELER_IMG}

# Push the controller docker image
docker-push-controller:
	docker push ${CONTROLLER_IMG}

# Push the nodelabeler docker image
docker-push-nodelabeler:
	docker push ${NODE_LABELER_IMG}

# Generate API reference documentation
api-docs: gen-crd-api-reference-docs
	$(API_REF_GEN) -api-dir=./pkg/api/v1 -config=./hack/api-docs/config.json -template-dir=./hack/api-docs/template -out-file=./docs/api/vino.md

API_REF_GEN=$(GOBIN)/gen-crd-api-reference-docs

# Find or download gen-crd-api-reference-docs
gen-crd-api-reference-docs:
	@{ \
	if ! which $(API_REF_GEN);\
	then\
		set -e ;\
		API_REF_GEN_TMP_DIR=$$(mktemp -d) ;\
		cd $$API_REF_GEN_TMP_DIR ;\
		go mod init tmp ;\
		go get github.com/ahmetb/gen-crd-api-reference-docs@$(API_REF_GEN_VERSION) ;\
		rm -rf $$API_REF_GEN_TMP_DIR ;\
	fi;\
	}

CONTROLLER_GEN:=$(GOBIN)/controller-gen

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
	@{ \
	if ! which $(CONTROLLER_GEN) || [ 'Version $(CONTROLLER_GEN_VERSION)' != "$$($(CONTROLLER_GEN) --version)" ];\
	then\
		set -e ;\
		CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
		cd $$CONTROLLER_GEN_TMP_DIR ;\
		go mod init tmp ;\
		go get sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION) ;\
		rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	fi;\
	}

.PHONY: lint
lint: $(LINTER)
	@echo "Performing linting step..."
	@./tools/whitespace_linter
	@./$(LINTER) run --config $(LINTER_CONFIG)
	@echo "Linting completed successfully"

$(LINTER): $(TOOLBINDIR)
	./tools/install_linter

$(TOOLBINDIR):
	mkdir -p $(TOOLBINDIR)

.PHONY: check-git-diff
check-git-diff:
	@./tools/git_diff_check