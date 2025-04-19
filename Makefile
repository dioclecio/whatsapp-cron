.PHONY: build push login

IMAGE_NAME = whatsapp-cron
REPO = quay.io/dioclecio

build:
	@read -p "Enter version tag for the image: " VERSION; \
	echo "Building image $(IMAGE_NAME):$$VERSION"; \
	podman build -t $(IMAGE_NAME):$$VERSION -t $(IMAGE_NAME):latest .; \
	echo $$VERSION > .version

login:
	@echo "Fazendo login no repositório $(REPO)..."; \
	podman login quay.io  

push: login build
	@VERSION=$$(cat .version); \
	echo "Pushing image $(REPO)/$(IMAGE_NAME):$$VERSION"; \
	podman push $(IMAGE_NAME):$$VERSION $(REPO)/$(IMAGE_NAME):$$VERSION; \
	echo "Pushing image $(REPO)/$(IMAGE_NAME):latest"; \
	podman push $(IMAGE_NAME):latest $(REPO)/$(IMAGE_NAME):latest