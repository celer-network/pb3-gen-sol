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
  --sol_out=msg=Msg1,msg=Msg2,importpb=true:test/solidity/src/lib \
  test/test.proto test/a.proto test/b.proto
```

## Development Workflow

Regenerate the checked-in test fixtures:

```bash
bash test/generate_sol_pb.sh
```

Human-readable fixture inputs live under `test/*.textpb`. The regeneration script writes binary protobuf fixtures into `test/fixtures/bin/` for Foundry to consume.

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
cd test/solidity
forge fmt --check
forge build
forge test -vv
```

## Test Layout

- `test/main_integration_test.go`: committed end-to-end regression test. It builds the root `protoc-gen-sol` binary, runs `protoc` against `test.proto`, `a.proto`, and `b.proto`, and asserts the generated Solidity contains the expected runtime hardening and output-shape invariants.
- `generator/generator_test.go`: focused Go unit tests for generator-only behavior such as plugin parameter parsing and `soltype` option decoding and validation.
- `test/solidity/test/PbDecoding.t.sol`: fixture-backed Solidity decoder regression suite that checks successful decodes and malformed-input reverts.
- `test/*.textpb`: human-authored protobuf text fixtures.
- `test/fixtures/bin/*.pb`: generated binary protobuf fixtures used by the Foundry suite.

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
- Decoder behavior is intentionally stricter for malformed payloads: oversized `uint256` byte strings, wrong-length `address` and `bytes32` fields, truncated varints, and truncated length-delimited values now revert deterministically.
- Empty top-level messages and empty embedded messages are now accepted instead of failing due to the removed legacy `raw.length > 1` guard.
- Narrow integer overrides (`uint8`, `uint32`, `uint64` from `(soltype)` or native proto types) silently truncate on-wire values that exceed the target width. The decoder follows Solidity narrow-cast semantics; producers that emit out-of-range values for a narrow field would be observed truncated, not rejected. If your contract treats a narrow field as an authoritative bound, validate it explicitly after decode.
- Unknown fields with wire types `Fixed32` (4-byte) and `Fixed64` (8-byte) are skipped per the proto3 spec, even though the generator does not emit those types as decoded fields. This keeps older decoders forward-compatible with schemas that add `fixed32` / `fixed64` / `float` / `double` fields under tags the older decoder isn't aware of.

## Known Gaps

- No support for `int32` or `int64`.
- Nested message and enum definitions are still unsupported.
- Sparse or reordered proto enum numeric values are unsupported because Solidity enums must remain dense `0..N`.
- The generator only emits decoders today.
