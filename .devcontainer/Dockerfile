FROM mcr.microsoft.com/devcontainers/go

# Install development tools
RUN \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
    && go install honnef.co/go/tools/cmd/staticcheck@latest \
    && go install golang.org/x/tools/cmd/godoc@latest