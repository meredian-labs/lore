#!/usr/bin/env bash
# update-tap.sh — push a fresh Homebrew formula to meredian-labs/homebrew-tap
# for the latest git tag. Run via `make tap` after `make release`.
set -euo pipefail

REPO="meredian-labs/lore"
TAP_REPO="meredian-labs/homebrew-tap"
TAG=$(git describe --tags --abbrev=0 2>/dev/null)

if [[ -z "$TAG" ]]; then
  echo "error: no git tag found — tag a release first (e.g. git tag v0.2.0)" >&2
  exit 1
fi

VERSION="${TAG#v}"  # strip leading 'v'

echo "Updating tap formula for $TAG..."

# Fetch checksums from the GitHub release.
CHECKSUMS=$(gh release download "$TAG" --repo "$REPO" \
  --pattern "checksums.txt" --output - 2>/dev/null)

sha_for() {
  echo "$CHECKSUMS" | grep "$1" | awk '{print $1}'
}

SHA_DARWIN_ARM64=$(sha_for "lore_darwin_arm64.tar.gz")
SHA_DARWIN_AMD64=$(sha_for "lore_darwin_amd64.tar.gz")
SHA_LINUX_ARM64=$(sha_for  "lore_linux_arm64.tar.gz")
SHA_LINUX_AMD64=$(sha_for  "lore_linux_amd64.tar.gz")

for var in SHA_DARWIN_ARM64 SHA_DARWIN_AMD64 SHA_LINUX_ARM64 SHA_LINUX_AMD64; do
  if [[ -z "${!var}" ]]; then
    echo "error: could not find checksum for $var — is $TAG published?" >&2
    exit 1
  fi
done

BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"

FORMULA=$(cat <<RUBY
# typed: false
# frozen_string_literal: true

# This file is maintained by scripts/update-tap.sh — do not edit manually.
class Lore < Formula
  desc "Local-first engineering memory system"
  homepage "https://github.com/${REPO}"
  version "${VERSION}"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "${BASE_URL}/lore_darwin_arm64.tar.gz"
      sha256 "${SHA_DARWIN_ARM64}"
    else
      url "${BASE_URL}/lore_darwin_amd64.tar.gz"
      sha256 "${SHA_DARWIN_AMD64}"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "${BASE_URL}/lore_linux_arm64.tar.gz"
      sha256 "${SHA_LINUX_ARM64}"
    else
      url "${BASE_URL}/lore_linux_amd64.tar.gz"
      sha256 "${SHA_LINUX_AMD64}"
    end
  end

  def install
    bin.install "lore"
    bin.install_symlink "lore" => "glh"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/lore --version")
  end
end
RUBY
)

# Get current file SHA (needed for GitHub API PUT).
FILE_SHA=$(gh api "repos/${TAP_REPO}/contents/lore.rb" --jq '.sha' 2>/dev/null || echo "")

ENCODED=$(echo "$FORMULA" | base64)

if [[ -n "$FILE_SHA" ]]; then
  gh api "repos/${TAP_REPO}/contents/lore.rb" \
    --method PUT \
    --field message="lore ${TAG}" \
    --field content="$ENCODED" \
    --field sha="$FILE_SHA" \
    --jq '.commit.html_url' | xargs echo "Formula updated:"
else
  gh api "repos/${TAP_REPO}/contents/lore.rb" \
    --method PUT \
    --field message="lore ${TAG}" \
    --field content="$ENCODED" \
    --jq '.commit.html_url' | xargs echo "Formula created:"
fi

echo ""
echo "Install with:"
echo "  brew tap meredian-labs/tap"
echo "  brew install lore"
