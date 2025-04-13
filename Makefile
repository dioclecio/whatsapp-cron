.PHONY: build

IMAGE_NAME = whatsapp-cron

build:
	@read -p "Enter version tag for the image: " VERSION; \
	echo "Building image $(IMAGE_NAME):$$VERSION"; \
	podman build -t $(IMAGE_NAME):$$VERSION -t $(IMAGE_NAME):latest .