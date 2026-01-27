.DEFAULT_GOAL := compile

{{ENVVAR_PREFIX}}_PORT ?= 1111

CMD_DIR = $(CURDIR)/cmd

compile:
	go build -o $(CMD_DIR)/{{NAME}} $(CMD_DIR)/{{NAME}}.go

build-image:
	docker build --build-arg GIT_SHA=$$(git rev-parse HEAD) -t {{DOCKER_IMAGE_NAME}} .

build-dev:
	DISABLE_AUTH=1 docker compose -f ./docker-compose.yml build

run-dev:
	DISABLE_AUTH=1 docker compose -f ./docker-compose.yml up

.PHONY: compile build-image build-dev run-dev
