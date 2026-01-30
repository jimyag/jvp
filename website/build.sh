#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# JVP Website Build Script for Cloudflare Worker
# =============================================================================

# Tool versions
HUGO_VERSION="0.154.2"

# Timezone
export TZ="Asia/Shanghai"

# =============================================================================
# Install Hugo extended
# =============================================================================
echo "Installing Hugo ${HUGO_VERSION}..."
mkdir -p "${HOME}/.local/hugo"
curl -sL "https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_linux-amd64.tar.gz" | tar -xz -C "${HOME}/.local/hugo"
export PATH="${HOME}/.local/hugo:${PATH}"

# Verify Hugo version
echo "Hugo version: $(hugo version)"

# =============================================================================
# Build site
# =============================================================================
echo "Building site..."
hugo --gc --minify
