#!/bin/bash

set -euo pipefail

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
repo_root="$(CDPATH= cd -- "$script_dir/.." && pwd)"
proto_unit_dir="$script_dir/proto/unit"
proto_bench_dir="$script_dir/proto/bench"
textpb_unit_dir="$script_dir/textpb/unit"
textpb_bench_dir="$script_dir/textpb/bench"
output_dir="$script_dir/lib"
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
rm -f "$fixture_output_dir"/*.pb
rm -f "$output_dir"/*.sol

protoc_include_dir="$(find_protoc_include)"
go build -o "$plugin_path" "$repo_root"

# Compile unit and bench protos in separate protoc invocations so that
# each compile only sees one `(soltype)` extension declaration.
# `test/proto/unit/test.proto` and `test/proto/bench/entity.proto` both
# define `extend google.protobuf.FieldOptions { string soltype = 1001; }`
# in different packages (`mytest.soltype` vs `entity.soltype`) but on the
# same field number 1001. Compiling them together makes protoc warn that
# extension number 1001 collides on `google.protobuf.FieldOptions`. The
# duplication is intentional: each schema is self-contained (`entity.proto`
# is copied verbatim from `agent-pay-contracts`), and at runtime no
# `FieldOptions` instance carries both extensions. Splitting the compile
# silences the cosmetic warning without coupling the schemas.
protoc \
    --plugin=protoc-gen-sol="$plugin_path" \
    --proto_path="$proto_unit_dir" \
    --proto_path="$protoc_include_dir" \
    --sol_out=importpb=true:"$output_dir" \
    test.proto a.proto b.proto

protoc \
    --plugin=protoc-gen-sol="$plugin_path" \
    --proto_path="$proto_bench_dir" \
    --proto_path="$protoc_include_dir" \
    --sol_out=importpb=true:"$output_dir" \
    chain.proto entity.proto

# Unit textpb fixtures: encode each msg<N>.textpb under mytest.Msg<N>; encode
# b.textpb under b.B.
for pathname in "$textpb_unit_dir"/msg*.textpb; do
    [[ -f "$pathname" ]] || continue
    filename=${pathname##*/}
    basename=${filename%.textpb}
    msg=${basename%%_*}
    msgno=${msg#msg}
    protoc --proto_path="$proto_unit_dir" --encode=mytest.Msg"$msgno" test.proto < "$pathname" > "$fixture_output_dir/$basename".pb || continue
done
protoc --proto_path="$proto_unit_dir" --encode=b.B b.proto < "$textpb_unit_dir/b.textpb" > "$fixture_output_dir/b.pb"

# Bench fixtures: each `bench_<name>.textpb` is encoded to the proto message
# listed in `bench_fixtures.manifest` (one `<basename> <fully.qualified.MessageName>`
# entry per line, comments allowed). Skip silently if the manifest or a
# textpb is missing — the bench tests load via vm.readFileBinary and will
# fail loudly on missing payloads.
if [[ -f "$script_dir/bench_fixtures.manifest" ]]; then
    while IFS=' ' read -r basename msgname; do
        [[ -z "$basename" || "$basename" == "#"* ]] && continue
        local_proto="${msgname%%.*}.proto"
        [[ -f "$textpb_bench_dir/$basename.textpb" ]] || continue
        protoc --proto_path="$proto_bench_dir" --encode="$msgname" "$local_proto" \
            < "$textpb_bench_dir/$basename.textpb" \
            > "$fixture_output_dir/$basename.pb"
    done < "$script_dir/bench_fixtures.manifest"
fi

if command -v forge >/dev/null 2>&1; then
    (
        cd "$script_dir"
        forge fmt lib/*.sol
    )
fi

echo "regenerated Solidity libraries into $output_dir and protobuf fixtures into $fixture_output_dir"
