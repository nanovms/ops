#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

GOGC=1 go vet ./lepton/...
GOGC=1 go vet ./cmd/...
