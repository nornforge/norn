#!/bin/sh

# print the Go version
go version


# Install Bash Autocompletion
sudo apt-get update
sudo apt-get install -q -y bash-completion \
                           jq

go get -u github.com/spf13/cobra@latest
go install github.com/spf13/cobra-cli@latest

# Go CI Lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v1.54.0
