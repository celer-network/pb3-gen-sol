#!/bin/bash

set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
repo_root="$(CDPATH= cd -- "$script_dir/.." && pwd)"
output_dir="$script_dir/solidity/src/lib"
fixture_output_dir="$script_dir/fixtures/bin"
plugin_path="$(mktemp "$script_dir/protoc-gen-sol.XXXXXX")"

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

trap 'rm -f "$plugin_path"' EXIT

mkdir -p "$output_dir" "$fixture_output_dir"
rm -f "$script_dir"/*.pb
rm -f "$fixture_output_dir"/*.pb
rm -f "$output_dir"/*.sol

protoc_include_dir="$(find_protoc_include)"
go build -o "$plugin_path" "$repo_root"

(
    cd "$script_dir"

    protoc \
        --plugin=protoc-gen-sol="$plugin_path" \
        --proto_path="$script_dir" \
        --proto_path="$protoc_include_dir" \
        --sol_out=importpb=true:"$output_dir" \
        test.proto a.proto b.proto

    for pathname in msg*.textpb; do
        filename=${pathname##*/}
        basename=${filename%.textpb}
        msg=${basename%%_*}
        msgno=${msg#msg}
        protoc --proto_path="$script_dir" --encode=mytest.Msg"$msgno" test.proto < "$basename".textpb > "$fixture_output_dir/$basename".pb || continue
    done

    protoc --proto_path="$script_dir" --encode=b.B b.proto < b.textpb > "$fixture_output_dir/b.pb"
)

if command -v forge >/dev/null 2>&1; then
    (
        cd "$script_dir/solidity"
        forge fmt src/lib/*.sol
    )
fi

echo "regenerated Solidity libraries into $output_dir and protobuf fixtures into $fixture_output_dir"
