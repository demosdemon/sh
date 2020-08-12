#!/usr/bin/env bash

set -eux

go get -u ./...
go mod tidy
go mod vendor -v

go build -o ./bin/gosh ./cmd/gosh
go build -o ./bin/shfmt ./cmd/shfmt

git add -f go.{mod,sum} vendor
git commit -m 'maint: update vendored packages' go.{mod,sum} vendor
