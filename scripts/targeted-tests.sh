#!/usr/bin/env bash
# Vet and test only the packages and modules affected by changed files; full run for build-level changes.
# Usage: scripts/targeted-tests.sh [--list] [changed-file ...]
# Without files, uses `git diff --name-only $BASE...HEAD` (BASE defaults to origin/main).
set -euo pipefail

list_only=0
if [ "${1:-}" = "--list" ]; then
  list_only=1
  shift
fi

cd "$(git rev-parse --show-toplevel)"

if [ "$#" -gt 0 ]; then
  changed=("$@")
else
  mapfile -d '' -t changed < <(git diff -z --name-only --no-renames "${BASE:-origin/main}...HEAD")
fi

if [ "${#changed[@]}" -eq 0 ]; then
  echo "no changed files; nothing to test"
  exit 0
fi

full_run() {
  echo "full run: $1"
  if [ "$list_only" -eq 1 ]; then
    echo "make vet wasm-vet test modules"
  else
    make vet wasm-vet test modules
  fi
  exit 0
}

module=$(go list -m)
direct=()
nested=()
grammar_changed=0
generate_changed=0
all_nested=0

module_dir_of() {
  local dir
  dir=$(dirname "$1")
  while [ "$dir" != "." ] && [ "$dir" != "/" ]; do
    if [ -f "$dir/go.mod" ]; then
      echo "$dir"
      return
    fi
    dir=$(dirname "$dir")
  done
  echo "."
}

add_nested() {
  local m
  for m in "${nested[@]:-}"; do
    [ "$m" = "$1" ] && return
  done
  nested+=("$1")
}

for f in "${changed[@]}"; do
  case "$f" in
    go.mod | go.sum | Makefile | .golangci.yml | scripts/targeted-tests.sh | .github/workflows/*)
      full_run "$f changed"
      ;;
    editors/vscode-xango/*)
      grammar_changed=1
      continue
      ;;
    *.xg)
      grammar_changed=1
      generate_changed=1
      ;;
    internal/transpile/* | cmd/xgc/*)
      generate_changed=1
      ;;
    *.md | LICENSE | NOTICE | COMMERCIAL.md | .gitignore | .github/* | scripts/* | docs/*)
      continue
      ;;
  esac
  mod_dir=$(module_dir_of "$f")
  if [ "$mod_dir" != "." ]; then
    add_nested "$mod_dir"
    continue
  fi
  case "$f" in
    *_test.go) ;;
    */*) ;;
    *.go) all_nested=1 ;;
  esac
  dir=$(dirname "$f")
  if [ ! -d "$dir" ]; then
    full_run "$f was deleted"
  fi
  if [ "$dir" = "." ]; then
    pkg="$module"
  else
    pkg="$module/$dir"
  fi
  direct+=("$pkg")
done

if [ "$all_nested" -eq 1 ]; then
  mapfile -t nested < <(git ls-files '*/go.mod' | xargs -n1 dirname | sort)
fi

affected_in_graph() {
  # stdin: "pkg imports..." lines from go list; $@: directly changed packages
  awk -v seeds="$*" '
    BEGIN { n = split(seeds, s, " "); for (i = 1; i <= n; i++) hit[s[i]] = 1 }
    { pkgs[NR] = $1; for (i = 2; i <= NF; i++) imp[NR, i] = $i; nf[NR] = NF; names[$1] = NR }
    END {
      do {
        changed = 0
        for (r = 1; r <= NR; r++) {
          if (hit[pkgs[r]]) continue
          for (i = 2; i <= nf[r]; i++) if (hit[imp[r, i]]) { hit[pkgs[r]] = 1; changed = 1; break }
        }
      } while (changed)
      for (p in hit) if (hit[p] && (p in names)) print p
    }' | sort
}

graph_fmt='{{.ImportPath}} {{join .Imports " "}} {{join .TestImports " "}} {{join .XTestImports " "}}'

native_pkgs=()
wasm_pkgs=()
if [ "${#direct[@]}" -gt 0 ]; then
  mapfile -t native_pkgs < <(go list -f "$graph_fmt" ./... | affected_in_graph "${direct[@]}")
  mapfile -t wasm_pkgs < <(GOOS=js GOARCH=wasm go list -f "$graph_fmt" ./... | affected_in_graph "${direct[@]}")
fi

echo "changed packages: ${direct[*]:-none}"
echo "native affected:  ${native_pkgs[*]:-none}"
echo "wasm affected:    ${wasm_pkgs[*]:-none}"
echo "nested modules:   ${nested[*]:-none}"
echo "regenerate .xg:   $([ "$generate_changed" -eq 1 ] && echo yes || echo no)"
echo "grammar suite:    $([ "$grammar_changed" -eq 1 ] && echo yes || echo no)"

if [ "$list_only" -eq 1 ]; then
  exit 0
fi

if [ "${#native_pkgs[@]}" -gt 0 ]; then
  go vet "${native_pkgs[@]}"
  go test "${native_pkgs[@]}"
fi
if [ "${#wasm_pkgs[@]}" -gt 0 ]; then
  GOOS=js GOARCH=wasm go vet "${wasm_pkgs[@]}"
fi
for m in "${nested[@]:-}"; do
  [ -n "$m" ] || continue
  echo "== $m"
  (
    cd "$m"
    go mod tidy -diff
    case "$m" in
      *gio*) if [ -n "${GIO_NATIVE:-}" ]; then go vet ./... && go test ./...; else GOOS=js GOARCH=wasm go vet ./...; fi ;;
      *) go vet ./... && go test ./... ;;
    esac
  )
done
if [ "$generate_changed" -eq 1 ]; then
  make generate generate-check
fi
if [ "$grammar_changed" -eq 1 ]; then
  make grammar-test-setup grammar-test
fi
