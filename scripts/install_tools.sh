#!/usr/bin/env bash
set -euo pipefail

# Install required Go tools if not already present

# Check for go
if ! command -v go &> /dev/null; then
    echo "ERROR: go is not installed"
    exit 1
fi

# Check for ginkgo
if ! command -v ginkgo &> /dev/null; then
    echo "-----> Installing ginkgo"
    go install github.com/onsi/ginkgo/ginkgo@latest
fi

# Check for jq
if ! command -v jq &> /dev/null; then
    echo "ERROR: jq is not installed. Please install jq."
    exit 1
fi

echo "-----> Tools verified"
