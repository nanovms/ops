#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

go vet -cpu 1 ./lepton/...
go vet -cpu 1 ./cmd/...
