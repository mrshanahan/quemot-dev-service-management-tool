FROM golang:latest AS builder

RUN mkdir -p /app
COPY . /app/{{NAME}}
WORKDIR /app/{{NAME}}
RUN go build ./cmd/{{NAME}}.go

FROM ubuntu:latest
ARG GIT_SHA

# Need to install ca-certificates explicitly to get the LetsEncrypt root cert
RUN apt update
RUN apt install ca-certificates -y
RUN mkdir -p /app
COPY --from=builder /app/{{NAME}}/{{NAME}} /app/{{NAME}}

ENV {{ENVVAR_PREFIX}}_PORT={{API_PORT_DEFAULT}}

LABEL dev.quemot.{{NAME}}.image.sha=$GIT_SHA

ENTRYPOINT [ "/app/{{NAME}}" ]