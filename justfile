set shell := ["sh", "-cu"]

default:
    just --list

fmt:
    gofmt -w cmd internal

test:
    go test ./...

build:
    go build -o bin/obsidian-preference-sync ./cmd/obsidian-preference-sync

check: fmt test build

release-check:
    GITHUB_REPOSITORY_OWNER=local GITHUB_REPOSITORY_NAME=obsidian-preference-sync mise exec -- goreleaser check

release-snapshot:
    GITHUB_REPOSITORY_OWNER=local GITHUB_REPOSITORY_NAME=obsidian-preference-sync mise exec -- goreleaser release --snapshot --clean

release version:
    #!/usr/bin/env sh
    set -eu

    version='{{ version }}'

    case "$version" in
      v[0-9]*.[0-9]*.[0-9]*) ;;
      *) echo "version must use vX.Y.Z format"; exit 1 ;;
    esac

    if ! printf '%s\n' "$version" | awk '/^v[0-9]+\.[0-9]+\.[0-9]+$/ { found=1 } END { exit found ? 0 : 1 }'; then
      echo "version must use vX.Y.Z format"
      exit 1
    fi

    git fetch --tags --prune-tags origin

    latest="$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | awk '/^v[0-9]+\.[0-9]+\.[0-9]+$/ { print; exit }')"
    if [ -n "$latest" ]; then
      echo "Latest tag: $latest"
      awk -v latest="$latest" -v target="$version" '
        function parse(version, parts) {
          sub(/^v/, "", version)
          split(version, parts, ".")
        }
        BEGIN {
          parse(latest, l)
          parse(target, n)
          for (i = 1; i <= 3; i++) {
            if (n[i] + 0 > l[i] + 0) exit 0
            if (n[i] + 0 < l[i] + 0) exit 1
          }
          exit 1
        }
      ' || { echo "$version must be greater than latest tag $latest"; exit 1; }
    else
      echo "Latest tag: none"
    fi

    printf 'Create and push tag %s? [y/N] ' "$version"
    read ans
    case "$ans" in
      y|Y|yes|YES) ;;
      *) echo "aborted"; exit 1 ;;
    esac

    git tag "$version"
    git push origin "$version"
