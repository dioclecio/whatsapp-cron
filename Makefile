.PHONY: build push login

IMAGE_NAME = whatsapp-cron
REPO = quay.io/dioclecio

build:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make build VERSION=<version_tag>"; \
		exit 1; \
	else \
		echo "Building image $(IMAGE_NAME):$(VERSION)"; \
		podman build -t $(IMAGE_NAME):$(VERSION) -t $(IMAGE_NAME):latest .; \
		echo $(VERSION) > .version; \
	fi

login:
	@echo "Fazendo login no repositório $(REPO)..."; \
	podman login quay.io  

push: login build
	@VERSION=$$(cat .version); \
	echo "Pushing image $(REPO)/$(IMAGE_NAME):$$VERSION"; \
	podman push $(IMAGE_NAME):$$VERSION $(REPO)/$(IMAGE_NAME):$$VERSION; \
	echo "Pushing image $(REPO)/$(IMAGE_NAME):latest"; \
	podman push $(IMAGE_NAME):latest $(REPO)/$(IMAGE_NAME):latest

# Adicionando suporte para passar a versão como argumento
.DEFAULT_GOAL := build
VERSION := $(word 2, $(MAKECMDGOALS))
ifeq ($(VERSION),)
	$(error Usage: make build VERSION=<version>)
endif