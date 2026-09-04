FROM golang:1.26-alpine3.22

ARG APP_ROOT=/home/dev/codepipeline_monorepo_practice
ARG TASK_VERSION=v3.45.4

ENV TZ=Asia/Tokyo \
    LANG=ja_JP.UTF-8 \
    LANGUAGE=ja_JP:ja \
    LC_ALL=ja_JP.UTF-8 \
    MUSL_LOCPATH=/usr/share/i18n/locales/musl

RUN apk add --no-cache \
      aws-cli \
      bash \
      build-base \
      ca-certificates \
      curl \
      git \
      jq \
      musl-locales \
      musl-locales-lang \
      nodejs \
      npm \
      ripgrep \
      shadow \
      sudo \
      tzdata \
      vim \
      zip && \
    cp /usr/share/zoneinfo/${TZ} /etc/localtime && \
    echo "${TZ}" > /etc/timezone && \
    update-ca-certificates && \
    arch="$(apk --print-arch)" && \
    case "${arch}" in \
      x86_64) task_arch='amd64' ;; \
      aarch64) task_arch='arm64' ;; \
      *) echo "unsupported architecture: ${arch}" >&2; exit 1 ;; \
    esac && \
    curl -fsSL "https://github.com/go-task/task/releases/download/${TASK_VERSION}/task_linux_${task_arch}.tar.gz" \
      | tar -xz -C /usr/local/bin task && \
    addgroup -g 1000 dev && \
    adduser -D -u 1000 -G dev -s /bin/bash dev && \
    echo "dev ALL=(root) NOPASSWD:ALL" > /etc/sudoers.d/dev && \
    chmod 0440 /etc/sudoers.d/dev && \
    npm install --global aws-cdk

ENV GOPATH=/home/dev/go
ENV PATH="${GOPATH}/bin:${PATH}"

RUN GOBIN=/usr/local/bin go install golang.org/x/tools/gopls@latest && \
    GOBIN=/usr/local/bin go install github.com/go-delve/delve/cmd/dlv@latest && \
    GOBIN=/usr/local/bin go install honnef.co/go/tools/cmd/staticcheck@latest && \
    GOBIN=/usr/local/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.3 && \
    mkdir -p \
      "${APP_ROOT}" \
      /home/dev/.claude \
      /home/dev/.codex \
      /home/dev/go/pkg/mod \
      /home/dev/.vscode-server && \
    chown -R dev:dev \
      "${APP_ROOT}" \
      /home/dev/.claude \
      /home/dev/.codex \
      /home/dev/go \
      /home/dev/.vscode-server

COPY --chown=dev:dev .devcontainer/infra/go/.bashrc /home/dev/.bashrc
COPY --chown=dev:dev .devcontainer/infra/go/config.toml /home/dev/.codex/config.toml

WORKDIR ${APP_ROOT}
USER dev

# Codex CLI
ENV NPM_CONFIG_PREFIX=/home/dev/.local
RUN mkdir -p /home/dev/.local && \
    npm install --global @openai/codex@latest

# Claude Code
RUN curl -fsSL https://claude.ai/install.sh | bash

ENV PATH="/home/dev/.local/bin:${PATH}"
