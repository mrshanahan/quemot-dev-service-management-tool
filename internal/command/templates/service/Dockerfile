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
ENV {{ENVVAR_PREFIX}}_AUTH_PROVIDER_URL=http://localhost:8080/realms/{{NAME}}
ENV {{ENVVAR_PREFIX}}_REDIRECT_URL=http://localhost:{{API_PORT_DEFAULT}}/auth/callback

LABEL dev.quemot.{{NAME}}.image.sha=$GIT_SHA

ENTRYPOINT [ "/app/{{NAME}}" ]