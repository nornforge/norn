#!/bin/sh

# print the Go version
go version


# Install Bash Autocompletion
sudo apt-get update
sudo apt-get install -q -y bash-completion \
                           jq

go get -u github.com/spf13/cobra@latest
go install github.com/spf13/cobra-cli@latest
