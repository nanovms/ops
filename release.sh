#!/bin/sh

export VERSION="0.1.15"

GO111MODULE=on GOOS=linux go build -ldflags "-X github.com/nanovms/ops/lepton.Version=$VERSION"
gsutil cp ops gs://cli/linux

GO111MODULE=on GOOS=darwin go build -ldflags "-w -X github.com/nanovms/ops/lepton.Version=$VERSION"
gsutil cp ops gs://cli/darwin

GO111MODULE=on GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/nanovms/ops/lepton.Version=$VERSION"
gsutil cp ops gs://cli/linux/aarch64/

gsutil -D setacl public-read gs://cli/linux/ops
gsutil -D setacl public-read gs://cli/linux/aarch64/ops
gsutil -D setacl public-read gs://cli/darwin/ops
