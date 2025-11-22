#!/usr/bin/env bash
set -euo pipefail

# Add GOPATH bin to PATH
export PATH="${PATH}:${HOME}/go/bin"

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
source ./scripts/install_tools.sh

echo "-----> Running unit tests"

# Run ginkgo tests
cd src/java
ginkgo -r -skipPackage=integration,brats

echo "-----> Unit tests complete"
