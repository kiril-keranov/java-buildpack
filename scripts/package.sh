#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
source ./scripts/install_tools.sh

ROOTDIR="$(pwd)"
BUILDPACK_DIR="${ROOTDIR}"

# Parse arguments
CACHED=false
STACK="cflinuxfs4"

while [[ $# -gt 0 ]]; do
    case $1 in
        --cached)
            CACHED=true
            shift
            ;;
        --stack)
            STACK="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

VERSION=$(cat "${ROOTDIR}/VERSION" 2>/dev/null || echo "0.0.0")
OUTPUT_FILE="${ROOTDIR}/java_buildpack-${STACK}-v${VERSION}.zip"

if [[ "${CACHED}" == "true" ]]; then
    OUTPUT_FILE="${ROOTDIR}/java_buildpack-cached-${STACK}-v${VERSION}.zip"
fi

echo "-----> Building buildpack"
./scripts/build.sh

# Create temporary directory for packaging
TMP_DIR=$(mktemp -d)
trap "rm -rf ${TMP_DIR}" EXIT

echo "-----> Packaging buildpack to ${OUTPUT_FILE}"

# Copy buildpack files
cp -r "${BUILDPACK_DIR}/bin" "${TMP_DIR}/"
cp -r "${BUILDPACK_DIR}/config" "${TMP_DIR}/"
cp -r "${BUILDPACK_DIR}/defaults" "${TMP_DIR}/"
cp -r "${BUILDPACK_DIR}/resources" "${TMP_DIR}/"
cp "${BUILDPACK_DIR}/manifest.yml" "${TMP_DIR}/"
cp "${BUILDPACK_DIR}/VERSION" "${TMP_DIR}/"

# If cached, download dependencies
if [[ "${CACHED}" == "true" ]]; then
    echo "-----> Downloading dependencies for offline buildpack"
    mkdir -p "${TMP_DIR}/dependencies"
    
    # Parse manifest.yml and download dependencies
    # This would use a tool to download all dependencies listed in manifest.yml
    # For now, this is a placeholder
    echo "       (Dependency download not yet implemented)"
fi

# Create zip file
cd "${TMP_DIR}"
zip -r "${OUTPUT_FILE}" .

echo "-----> Buildpack packaged successfully: ${OUTPUT_FILE}"
