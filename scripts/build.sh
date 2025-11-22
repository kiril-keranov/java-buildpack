#!/usr/bin/env bash
set -euo pipefail

# Add GOPATH bin to PATH
export PATH="${PATH}:${HOME}/go/bin"

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
source ./scripts/install_tools.sh

ROOTDIR="$(pwd)"

# Find all CLI packages
IFS=" " read -r -a binaries <<< "$(find "${ROOTDIR}/src/java" -name cli -type d -print0 | xargs -0)"

# Read supported OSes from config.json
if [[ -f "${ROOTDIR}/config.json" ]]; then
    IFS=" " read -r -a oses <<< "$(jq -r -S '.oses[]' "${ROOTDIR}/config.json" | xargs)"
else
    # Default to linux if config.json doesn't exist yet
    oses=("linux")
fi

# Build for each OS
for os in "${oses[@]}"; do
    for path in "${binaries[@]}"; do
        name="$(basename "$(dirname "${path}")")"
        output="${ROOTDIR}/bin/${name}"
        
        if [[ "${os}" == "windows" ]]; then
            output="${output}.exe"
        fi
        
        echo "-----> Building ${name} for ${os}"
        CGO_ENABLED=0 GOOS="${os}" go build \
            -mod vendor \
            -ldflags="-s -w" \
            -o "${output}" \
            "${path}"
    done
done

echo "-----> Build complete"
