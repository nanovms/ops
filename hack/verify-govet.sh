#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

GOGC=20 go vet ./lepton/...
GOGC=20 go vet ./cmd/...
