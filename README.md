# pb3-gen-sol

`pb3-gen-sol` is a `protoc` plugin that generates Solidity protobuf decoders for proto3 schemas. The generator targets Solidity `>=0.8.0`, emits a shared `Pb.sol` runtime, and supports the field-option-based Solidity type mappings used by Celer contracts.

## Features

- Proto3 decoder generation for top-level messages and enums.
- Cross-package reference support, including multi-segment proto package names.
- Separate `Pb.sol` runtime support library.
- Solidity native type overrides via `google.protobuf.FieldOptions` and `(soltype)`.
- Foundry-based Solidity regression suite, with binary `.pb` fixtures regenerated from checked-in human-readable `.textpb` sources.
- Go integration test that builds the plugin and runs `protoc` end to end.
- Go unit tests covering generator parameter parsing and `soltype` option handling.

## Why protobuf for Solidity?

If your off-chain stack already speaks protobuf — multiple producer
languages writing the same signed bytes, schema evolution managed via
tag numbers — `pb3-gen-sol` lets on-chain Solidity contracts consume
the same payloads without hand-rolling ABI structs or duplicating the
schema.

The trade-off vs hand-rolled Solidity ABI:

- **Language- and platform-neutral.** Any client (Go, Rust, TS, Java,
  etc.) can produce `.pb` bytes that decode on EVM (this repo), non-EVM
  chains (CosmWasm, Move, Solana programs with protobuf libraries), or
  any off-chain verifier — all from one schema.
- **Native schema evolution.** Adding fields with new tag numbers is
  safe; older decoders skip unknown tags per the proto3 spec. ABI has
  no native skip-unknown — adding a field shifts every offset and
  breaks every existing decoder.
- **Smaller wire size on sparse messages.** Proto3 elides zero-valued
  fields entirely; ABI always encodes every field.

ABI decode is cheaper in gas on most shapes. On L1 mainnet the
wire-size advantage roughly cancels the decode-gas penalty; on L2
rollups protobuf's smaller wire size wins on virtually every shape. See
[test/sol/bench/benchmark.md](test/sol/bench/benchmark.md) for the
full numbers and per-path breakdown.

## Supported Types

Native proto types:

- `uint32`
- `uint64`
- `bool`
- `bytes`
- `string`

Supported `(soltype)` overrides:

- `uint8`
- `uint`
- `uint256`
- `address`
- `address payable`
- `bytes32`

`uint` is the varint-backed path for protobuf `uint64` fields exposed as Solidity's native unsigned integer type. `uint256` is the bytes-backed path for true 256-bit values.

## Example Schema

```protobuf
syntax = "proto3";
package mytest;

import "google/protobuf/descriptor.proto";

extend google.protobuf.FieldOptions {
    string soltype = 54321;
}

message MyMsg {
    bytes addr = 1 [(soltype) = "address"];
    uint32 num = 2 [(soltype) = "uint8"];
    bool has_moon = 3;
}
```

More complete examples live under `test/`.

## Generate Solidity

Build the plugin and place it on your `PATH`:

```bash
go build -o protoc-gen-sol .
export PATH="$PWD:$PATH"
```

Then invoke `protoc`:

```bash
protoc --sol_out=importpb=true:out proto/example.proto
```

The generated file name is derived from the proto package name. With `importpb=true`, generated libraries import `Pb.sol`; otherwise the runtime is embedded into each generated file.

## Plugin Parameters

- `msg=<Name>`: only emit selected top-level message decoders. Repeat the parameter to whitelist multiple messages.
- `importpb=true|false`: emit `Pb.sol` as a separate artifact and import it from generated files.

Example:

```bash
protoc \
  --sol_out=msg=Msg1,msg=Msg2,importpb=true:test/lib \
  test/proto/unit/test.proto test/proto/unit/a.proto test/proto/unit/b.proto
```

## Development Workflow

Regenerate the checked-in test fixtures:

```bash
bash test/generate_sol_pb.sh
```

Human-readable fixture inputs live under `test/textpb/{unit,bench}/*.textpb`. The regeneration script encodes them via `protoc` and writes binary `.pb` outputs into `test/fixtures/bin/` for Foundry to consume.

Run Go validation:

```bash
go test ./...
```

Run only the committed end-to-end generator integration test:

```bash
go test ./test -run TestGeneratorFixtures
```

Run only the generator unit tests:

```bash
go test ./generator
```

Run Solidity validation:

```bash
cd test
forge fmt --check
forge build
forge test -vv
```

## Test Layout

```
test/
├── main_integration_test.go      Go integration test (builds plugin, invokes protoc, asserts output shape)
├── generate_sol_pb.sh            regenerates Solidity decoders + binary protobuf fixtures
├── bench_fixtures.manifest       maps each bench textpb basename → its proto message type
├── foundry.toml                  Foundry project config (this is the Foundry root)
├── proto/
│   ├── unit/                     test.proto, a.proto, b.proto — schemas used by the unit tests
│   └── bench/                    chain.proto, entity.proto — AgentPay schemas used by the bench
├── textpb/
│   ├── unit/                     msg*.textpb, b.textpb — fixture sources for the unit tests
│   └── bench/                    bench_*.textpb — fixture sources for the bench
├── fixtures/bin/                 *.pb regenerated from textpb (gitignored)
├── lib/                          generated Solidity decoder libraries (Foundry src)
└── sol/                          Foundry test root
    ├── PbDecoding.t.sol          decoder correctness + malformed-input revert tests
    ├── utils/TestBase.sol        minimal hand-rolled assertEq helpers (no forge-std dep)
    └── bench/                    decode-gas benchmark vs abi.encode/decode
        ├── Bench.t.sol           paired pb_decX / abi_decX tests
        └── benchmark.md          benchmark numbers + interpretation
```

`generator/generator_test.go` covers plugin parameter parsing, `soltype` option decoding, and other generator-only invariants.

The integration test is intentionally committed because it is the cheapest end-to-end check that the plugin still builds, `protoc` still invokes it correctly, and the generated Solidity still carries the expected modernization and hardening changes.

## CI

GitHub Actions is the authoritative CI. The workflow runs:

- Go tests, including the generator integration test.
- Fixture regeneration plus generated-output drift detection.
- Foundry formatting, build, and test checks.

The Forge job also installs Go and `protoc` because it regenerates Solidity and protobuf fixtures before running the Solidity checks.

## Compatibility Notes

- Valid protobuf payloads remain wire-compatible with the previous generator for the supported schema features.
- Generated Solidity now targets `pragma solidity >=0.8.0;` and uses the modern `Pb.sol` shared-runtime layout when `importpb=true` is used.
- Decoder behavior is intentionally stricter for malformed payloads: oversized `uint256` byte strings, wrong-length `address` and `bytes32` fields, truncated varints, oversized varints exceeding the protobuf `uint64` range, and truncated length-delimited values now revert deterministically.
- Per-field wire-type validation is folded into dispatch: each known tag matches only when both its tag number *and* its expected protobuf wire type appear together in the encoded key. A known tag arriving with the wrong wire type does not match any branch and is skipped per proto3 unknown-field semantics, leaving the field at its default zero value rather than decoding garbled data.
- Empty top-level messages and empty embedded messages are now accepted instead of failing due to the removed legacy `raw.length > 1` guard.
- Narrow integer overrides (`uint8`, `uint32`, `uint64` from `(soltype)` or native proto types) silently truncate on-wire values that exceed the target width. The decoder follows Solidity narrow-cast semantics; producers that emit out-of-range values for a narrow field would be observed truncated, not rejected. If your contract treats a narrow field as an authoritative bound, validate it explicitly after decode.
- Unknown fields with wire types `Fixed32` (4-byte) and `Fixed64` (8-byte) are skipped per the proto3 spec, even though the generator does not emit those types as decoded fields. This keeps older decoders forward-compatible with schemas that add `fixed32` / `fixed64` / `float` / `double` fields under tags the older decoder isn't aware of.

## Known Gaps

- No support for `int32` or `int64`.
- Nested message and enum definitions are still unsupported.
- Sparse or reordered proto enum numeric values are unsupported because Solidity enums must remain dense `0..N`.
- The generator only emits decoders today.
