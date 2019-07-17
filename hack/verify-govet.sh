#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

GO111MODULE=on
export GO111MODULE

go vet ./lepton/...
go vet ./cmd/...
