SERVICE_NAME := $(shell grep "^module" go.mod | rev | cut -d "/" -f1 | rev)
GIT_REF := $(shell git rev-parse --short=7 HEAD)
VERSION ?= commit-$(GIT_REF)

# Setup GCP Registry Variables
GCR_PROJECT :=
REGISTRY := gcr.io/$(GCR_PROJECT)
IMAGE := $(REGISTRY)/$(SERVICE_NAME):$(VERSION)

.PHONY: test
test:
	@go test ./...

.PHONY: cloudbuild
cloudbuild:
	@gcloud builds submit . \
		--project=$(GCR_PROJECT) \
	 	--config=.cloudbuild.yaml \
	 	--substitutions="_IMAGE=$(IMAGE),_GITHUB_TOKEN=$(GITHUB_TOKEN),_VERSION=$(VERSION),_SERVICE_NAME=$(SERVICE_NAME)"
	 	--gcs-log-dir="gs://$(GCR_PROJECT)_cloudbuild/log" \

.PHONY: run
run:
	@go run .
