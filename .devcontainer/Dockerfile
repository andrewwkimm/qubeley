FROM mcr.microsoft.com/devcontainers/go:2-1.24-bookworm

# Install Docker
RUN apt-get update && apt-get install -y docker.io && rm -rf /var/lib/apt/lists/*

# Install developer tools
RUN \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 \
    && go install honnef.co/go/tools/cmd/staticcheck@v0.5.1 \
    && go install golang.org/x/tools/cmd/godoc@v0.29.0