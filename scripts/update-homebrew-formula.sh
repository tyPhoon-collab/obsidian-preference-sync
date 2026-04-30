#!/usr/bin/env sh
set -eu

tag="${1:-${GITHUB_REF_NAME:-}}"
sha256="${2:-}"

if [ -z "$tag" ]; then
  echo "usage: $0 <tag> [sha256]" >&2
  exit 1
fi

case "$tag" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) echo "tag must use vX.Y.Z format" >&2; exit 1 ;;
esac

if ! printf '%s\n' "$tag" | awk '/^v[0-9]+\.[0-9]+\.[0-9]+$/ { found=1 } END { exit found ? 0 : 1 }'; then
  echo "tag must use vX.Y.Z format" >&2
  exit 1
fi

archive_url="https://github.com/tyPhoon-collab/obsidian-preference-sync/archive/refs/tags/${tag}.tar.gz"

if [ -z "$sha256" ]; then
  archive_file="$(mktemp)"
  trap 'rm -f "$archive_file"' INT TERM EXIT
  curl -fsSL -o "$archive_file" "$archive_url"
  sha256="$(sha256sum "$archive_file" | awk '{ print $1 }')"
  rm -f "$archive_file"
  trap - INT TERM EXIT
fi

tap_dir="${HOMEBREW_TAP_DIR:-}"
cleanup_tap_dir=0
if [ -z "$tap_dir" ]; then
  if [ -z "${HOMEBREW_TAP_GITHUB_TOKEN:-}" ]; then
    echo "HOMEBREW_TAP_GITHUB_TOKEN is required when HOMEBREW_TAP_DIR is not set" >&2
    exit 1
  fi
  tap_dir="$(mktemp -d)"
  cleanup_tap_dir=1
  git clone "https://x-access-token:${HOMEBREW_TAP_GITHUB_TOKEN}@github.com/tyPhoon-collab/homebrew-tap.git" "$tap_dir"
fi

cleanup() {
  if [ "$cleanup_tap_dir" -eq 1 ]; then
    rm -rf "$tap_dir"
  fi
}
trap cleanup EXIT

cd "$tap_dir"
mkdir -p Formula

cat > Formula/obsidian-preference-sync.rb <<EOF
class ObsidianPreferenceSync < Formula
  desc "Synchronize selected Obsidian plugins, plugin settings, and app settings"
  homepage "https://github.com/tyPhoon-collab/obsidian-preference-sync"
  url "${archive_url}"
  sha256 "${sha256}"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build",
      "-mod=vendor",
      "-trimpath",
      "-ldflags", "-s -w",
      "-o", bin/"obsidian-preference-sync",
      "./cmd/obsidian-preference-sync"
  end

  test do
    output = shell_output("#{bin}/obsidian-preference-sync 2>&1", 2)
    assert_match "--vault is required", output
  end
end
EOF

if [ "${DRY_RUN:-0}" = "1" ]; then
  cat Formula/obsidian-preference-sync.rb
  exit 0
fi

git add Formula/obsidian-preference-sync.rb

if git diff --cached --quiet -- Formula/obsidian-preference-sync.rb; then
  echo "Homebrew formula is already up to date"
  exit 0
fi

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git commit -m "Update obsidian-preference-sync to ${tag}"
git push
