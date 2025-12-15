.DEFAULT_GOAL := compile

{{ENVVAR_PREFIX}}_PORT ?= 1111

CMD_DIR = $(CURDIR)/cmd

compile:
	go build -o $(CMD_DIR)/{{NAME}} $(CMD_DIR)/{{NAME}}.go

build-image:
	docker build --build-arg GIT_SHA=$$(git rev-parse HEAD) -t {{DOCKER_IMAGE_NAME}} .

.PHONY: compile build-image
