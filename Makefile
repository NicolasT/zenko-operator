DOCKER_TAG = docker.io/nicolast/zenko-operator:0.0.0

OPERATOR_HELM_CHART = charts/zenko
OPERATOR_API_VERSION = zenko.io/v1alpha1
OPERATOR_KIND = Zenko

DOCKER = $(shell command -v docker)

GO = $(shell command -v go)

bin/zenko-operator: cmd/zenko-operator/main.go
	@CGO_ENABLED=0 $(GO) build -o $@ $<

docker-build:
	@$(DOCKER) build \
		-t $(DOCKER_TAG) \
		--build-arg HELM_CHART="$(OPERATOR_HELM_CHART)" \
		--build-arg API_VERSION="$(OPERATOR_API_VERSION)" \
		--build-arg KIND="$(OPERATOR_KIND)" \
		.
.PHONY: docker-build

docker-push:
	@$(DOCKER) push "$(DOCKER_TAG)"
.PHONY: docker-push
