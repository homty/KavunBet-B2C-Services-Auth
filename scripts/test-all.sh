#!/usr/bin/env bash
set -uo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repository_root="$(cd -- "$script_dir/.." && pwd)"

modules=(
    "pkg"
    "services/auth-service"
    "services/user-service"
    "services/wallet-service"
)

failed_modules=()

for module in "${modules[@]}"; do
    printf '\n==> Testing %s\n' "$module"

    if ! (
        cd "$repository_root/$module"
        go test -count=1 "$@" ./...
    ); then
        failed_modules+=("$module")
    fi
done

if ((${#failed_modules[@]} > 0)); then
    printf '\nTest failures:\n' >&2
    printf '  - %s\n' "${failed_modules[@]}" >&2
    exit 1
fi

printf '\nAll Go module tests passed.\n'
