.PHONY: check generate build-image push-image push-latest test

PKG := github.com/openshift/osde2e
E2E_PKG := $(PKG)/suites/e2e
SCALE_PKG := $(PKG)/suites/scale

DIR := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))

IMAGE_NAME := quay.io/app-sre/osde2e
IMAGE_TAG := $(shell git rev-parse --short=7 HEAD)

CONTAINER_ENGINE ?= docker

check: cmd/osde2e-docs
	go run $(PKG)/$< --check
	CGO_ENABLED=0 go test -v ./cmd/... ./pkg/...
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint && \
    golangci-lint run -c .golang-ci.yml ./... 

generate: docs/Options.md

build-image:
	$(CONTAINER_ENGINE) build -t "$(IMAGE_NAME):$(IMAGE_TAG)" .

push-image:
	@$(CONTAINER_ENGINE) --config=$(DOCKER_CONF) push "$(IMAGE_NAME):$(IMAGE_TAG)"

push-latest:
	$(CONTAINER_ENGINE) tag "$(IMAGE_NAME):$(IMAGE_TAG)" "$(IMAGE_NAME):latest"
	@$(CONTAINER_ENGINE) --config=$(DOCKER_CONF) push "$(IMAGE_NAME):latest"

test:
	go test $(E2E_PKG) -test.v -ginkgo.skip="$(GINKGO_SKIP)" -test.timeout 8h

test-scale:
	go test $(SCALE_PKG) -test.v -ginkgo.skip="$(GINKGO_SKIP)" -test.timeout 8h -test.run TestScale

test-docker:
	$(CONTAINER_ENGINE) run \
		-t \
		--rm \
		-e NO_DESTROY=$(NO_DESTROY) \
		-e CLUSTER_ID=$(CLUSTER_ID) \
		-e CLUSTER_NAME=$(CLUSTER_NAME) \
		-e CLEAN_RUNS=$(CLEAN_RUNS) \
		-e DRY_RUN=$(DRY_RUN) \
		-e MAJOR_TARGET=$(MAJOR_TARGET) \
		-e MINOR_TARGET=$(MINOR_TARGET) \
		-e CLUSTER_VERSION=$(CLUSTER_VERSION) \
		-e NO_DESTROY_DELAY=$(NO_DESTROY_DELAY) \
		-e GINKGO_SKIP="$(GINKGO_SKIP)" \
		-e UPGRADE_RELEASE_STREAM=$(UPGRADE_RELEASE_STREAM) \
		-e DEBUG_OSD=1 \
		-e OSD_ENV=$(OSD_ENV) \
		-e UHC_TOKEN=$(UHC_REFRESH_TOKEN) \
		-e AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) \
		-e AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) \
		$(IMAGE_NAME):$(IMAGE_TAG)

docs/Options.md: cmd/osde2e-docs pkg/config/config.go
	go run $(PKG)/$<
