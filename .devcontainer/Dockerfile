FROM mcr.microsoft.com/devcontainers/go:2-1.24-bookworm

RUN \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 \
    && go install honnef.co/go/tools/cmd/staticcheck@v0.5.1 \
    && go install golang.org/x/tools/cmd/godoc@v0.29.0