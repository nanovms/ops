#!/bin/sh

GO111MODULE=on GOOS=linux go build
gsutil cp ops gs://cli/linux

GO111MODULE=on GOOS=darwin go build -ldflags "-w"
gsutil cp ops gs://cli/darwin

GO111MODULE=on GOOS=linux GOARCH=arm64 go build
gsutil cp ops gs://cli/linux/aarch64/

gsutil -D setacl public-read gs://cli/linux/ops
gsutil -D setacl public-read gs://cli/linux/aarch64/ops
gsutil -D setacl public-read gs://cli/darwin/ops
