#!/usr/bin/env bash
#
# Regenerate the `new/` half of the bench fixtures from the proto sources in
# bench/proto/ using the current pb3-gen-sol plugin. The `old/` half is a
# frozen snapshot of the hand-tuned agent-pay-contracts runtime and decoders;
# do not regenerate it from this script.
#
# Run after every change to the runtime template or the generator:
#
#     bash test/solidity/test/bench/regen.sh
#
# The renamed libraries (PbNew / PbChainNew / PbEntityNew) keep the new and
# old runtimes coexistent in the same Foundry compilation unit.

set -euo pipefail

bench_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
repo_root="$(CDPATH= cd -- "$bench_dir/../../../.." && pwd)"
proto_dir="$bench_dir/proto"
new_dir="$bench_dir/new"

find_protoc_include() {
    local protoc_path
    protoc_path="$(command -v protoc)"

    local candidates=(
        "$(dirname "$(dirname "$protoc_path")")/include"
        "/usr/local/include"
        "/opt/homebrew/include"
    )

    local candidate
    for candidate in "${candidates[@]}"; do
        if [[ -f "$candidate/google/protobuf/descriptor.proto" ]]; then
            printf '%s\n' "$candidate"
            return 0
        fi
    done

    echo "could not locate protobuf include directory" >&2
    return 1
}

plugin_path="$(mktemp "$bench_dir/protoc-gen-sol.XXXXXX")"
trap 'rm -f "$plugin_path"' EXIT

go build -o "$plugin_path" "$repo_root"

protoc_include_dir="$(find_protoc_include)"

rm -f "$new_dir"/*.sol
protoc \
    --plugin=protoc-gen-sol="$plugin_path" \
    --proto_path="$proto_dir" \
    --proto_path="$protoc_include_dir" \
    --sol_out=importpb=true:"$new_dir" \
    chain.proto entity.proto

# Rename libraries Pb/PbChain/PbEntity → *New so they don't clash with the
# checked-in `old/` snapshot or with `src/lib/Pb.sol`.
sed_inplace() {
    if sed --version >/dev/null 2>&1; then
        sed -i -e "$1" "$2"
    else
        sed -i '' -e "$1" "$2"
    fi
}

for f in "$new_dir/Pb.sol" "$new_dir/PbChain.sol" "$new_dir/PbEntity.sol"; do
    sed_inplace 's/library Pb {/library PbNew {/g' "$f"
    sed_inplace 's/library PbChain {/library PbChainNew {/g' "$f"
    sed_inplace 's/library PbEntity {/library PbEntityNew {/g' "$f"
    sed_inplace 's/Pb\.Buffer/PbNew.Buffer/g' "$f"
    sed_inplace 's/Pb\.WireType/PbNew.WireType/g' "$f"
    sed_inplace 's/Pb\.fromBytes/PbNew.fromBytes/g' "$f"
    sed_inplace 's/Pb\._/PbNew._/g' "$f"
    sed_inplace 's/Pb\.uint/PbNew.uint/g' "$f"
    sed_inplace 's/Pb\.bools/PbNew.bools/g' "$f"
    sed_inplace 's/using Pb for/using PbNew for/g' "$f"
done

if command -v forge >/dev/null 2>&1; then
    (
        cd "$bench_dir/../.."
        forge fmt test/bench/new/*.sol >/dev/null
    )
fi

echo "regenerated $new_dir/{Pb,PbChain,PbEntity}.sol from $proto_dir/*.proto"
