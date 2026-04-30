// protoc-gen-sol by Celer Network Team

/*
The code generator for the plugin for the Google protocol buffer compiler.
It generates Solidity code from the protocol buffer description files read by the
main routine.
*/
package generator

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
	plugin "google.golang.org/protobuf/types/pluginpb"
)

// TODO(template?): consider use text/template

// TODO(oneof): support Oneof field

// TODO(nested): support Nested msg/enum definition within message

// ExtName is the extension name to google.protobuf.FieldOptions
// its type must be string. Valid string values are keys in SolTypeMap
// (we don't use enum to avoid having solidity knowledge in chain.proto)
const ExtName = "soltype"

// SolVer is the compatible solidity/solc version in pragma solidity
const SolVer = ">=0.8.0;"

// string const for proto wire types
const WireVarint = "Varint"
const WireLendel = "Bytes"

// PassTypeMap is a map of proto types with native solidity support (aka. same type keyword in proto and solidity)
// to its wire type. keys are values from pbType2Str map below.
// currently we only support 2 wire types: varint and length-delimited.
var PassTypeMap = map[string]string{
	//"int32":  WireVarint,
	//"int64":  WireVarint,
	"uint32": WireVarint,
	"uint64": WireVarint,
	"bool":   WireVarint,
	"bytes":  WireLendel,
	"string": WireLendel,
}

// SolTypeMap is a map of solidity types as valid soltype option value
// to required proto primitive type.
// eg. a field can have (soltype) = "address payable", it must be defined as proto bytes
var SolTypeMap = map[string]string{
	"uint8":           "uint32",
	"address":         "bytes",
	"address payable": "bytes",
	"bytes32":         "bytes",
	"uint256":         "bytes",
	"uint":            "uint64",
	// solidity uint is an alias to uint256. but we add our own schema to it.
	// and only use uint256 for amount in wei. use uint for uint64 which could in theory save some gas.
	// eg. proto type uint64, soltype uint. Note without uint soltype it also works, only a bit more gas.
}

// map supported proto enum types to its string
var pbType2Str = map[descriptor.FieldDescriptorProto_Type]string{
	//descriptor.FieldDescriptorProto_TYPE_INT32:  "int32",
	//descriptor.FieldDescriptorProto_TYPE_INT64:  "int64",
	descriptor.FieldDescriptorProto_TYPE_UINT32: "uint32",
	descriptor.FieldDescriptorProto_TYPE_UINT64: "uint64",
	descriptor.FieldDescriptorProto_TYPE_BOOL:   "bool",
	descriptor.FieldDescriptorProto_TYPE_BYTES:  "bytes",
	descriptor.FieldDescriptorProto_TYPE_STRING: "string",
}

// type alias for easy typing
type fdes *descriptor.FileDescriptorProto
type msgdes *descriptor.DescriptorProto
type enumdes *descriptor.EnumDescriptorProto

// Generator is the type whose methods generate the output, stored in the associated response structure.
type Generator struct {
	*bytes.Buffer                               // cache .P() output
	Request       *plugin.CodeGeneratorRequest  // The input.
	Response      *plugin.CodeGeneratorResponse // The output.
	indent        string
	extnum        int32           // assigned field number eg. 1001 for ExtName
	importpb      bool            // whether to include library Pb in the generated .sol or import. if true, create import "Pb.sol" in header
	onlymsgs      map[string]bool // msg names specified by user in arg as whitelist, if not empty, only generate msg if it's in the list
}

// New creates a new generator and allocates the request and response protobufs.
func New() *Generator {
	g := new(Generator)
	g.Buffer = new(bytes.Buffer)
	g.Request = new(plugin.CodeGeneratorRequest)
	g.Response = new(plugin.CodeGeneratorResponse)
	g.onlymsgs = make(map[string]bool)
	return g
}

// In indents the output 4 spaces stop. per solidity style guide
func (g *Generator) In() { g.indent += "    " }

// Out unindents the output 4 spaces stop.
func (g *Generator) Out() {
	if len(g.indent) > 0 {
		g.indent = g.indent[4:]
	}
}

func (g *Generator) P(str ...interface{}) {
	g.WriteString(g.indent)
	for _, v := range str {
		g.printAtom(v)
	}
	g.WriteByte('\n')
}

func (g *Generator) printAtom(v interface{}) {
	switch v := v.(type) {
	case string:
		g.WriteString(v)
	case *string:
		g.WriteString(*v)
	case bool:
		fmt.Fprint(g, v)
	case *bool:
		fmt.Fprint(g, *v)
	case int:
		fmt.Fprint(g, v)
	case int32:
		fmt.Fprint(g, v)
	case *int32:
		fmt.Fprint(g, *v)
	case *int64:
		fmt.Fprint(g, *v)
	case float64:
		fmt.Fprint(g, v)
	case *float64:
		fmt.Fprint(g, *v)
	default:
		log.Print("unknown type in printer: ", v)
	}
}

func (g *Generator) ParseParams() {
	// only support 2 args for now:
	// msg=MsgA,msg=MsgB
	// importpb=true/false (false is default), if true, will generate import "Pb.sol" instead of having library Pb in the generated .sol
	// Note the param affects all .proto files
	parameter := g.Request.GetParameter()
	if len(parameter) == 0 {
		return
	}
	for _, p := range strings.Split(parameter, ",") {
		if p == "" {
			continue
		}
		key, value, ok := strings.Cut(p, "=")
		if !ok || key == "" || value == "" {
			Fail("invalid parameter", p)
		}
		switch key {
		case "msg":
			g.onlymsgs[value] = true
		case "importpb":
			if value == "true" {
				g.importpb = true
			} else if value != "false" {
				Fail("invalid importpb value", value)
			}
		default:
			Fail("Unknown params ", key, value)
		}
	}
}

// GenerateAllFiles generates the output for all the files we're outputting.
func (g *Generator) GenerateAllFiles() {
	for _, f := range g.Request.ProtoFile {
		if !inArray(*f.Name, g.Request.FileToGenerate) {
			// log.Println("Skip import file:", *f.Name)
			// We could build fname to package mapping here
			continue
		}
		g.Reset() // clear buffer
		g.indent = "" // and reset indent state, in case a prior generate() left it non-empty
		g.generate(f)
		outfn := getSolFile(f.GetPackage()) // file name for generated .sol file
		g.Response.File = append(g.Response.File, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(outfn),
			Content: proto.String(g.String()),
		})
	}
	if g.importpb {
		g.Response.File = append(g.Response.File, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String("Pb.sol"),
			Content: proto.String(standaloneRuntimeContent()),
		})
	}
}

// Fill the response protocol buffer with the generated output for all the files we're
// supposed to generate.
func (g *Generator) generate(f fdes) {
	if *f.Syntax != "proto3" {
		Fail("Only support proto3")
	}
	// find ExtName number if it's defined, -1 if not
	g.extnum = getExtNum(f, g.Request.ProtoFile)
	currentPkg := *f.Package
	knownPkgs := collectProtoPackages(g.Request.ProtoFile)
	g.generateHeader(f)
	g.In()
	g.P("using Pb for Pb.Buffer;  // so we can call Pb funcs on Buffer obj\n")

	// go over all top level enums
	for _, enum := range f.EnumType {
		g.generateEnum(enum)
	}

	// go over all top level messages
	for _, msg := range f.MessageType {
		if g.shouldOutput(*msg.Name) {
			g.generateMsg(msg, currentPkg, knownPkgs)
		}
	}
	g.Out()
	g.P("}") // close library
	if !g.importpb {
		g.appendRuntimeLibrary()
	}
}

// Generate the header, including package definition
func (g *Generator) generateHeader(f fdes) {
	g.P("// SPDX-License-Identifier: MIT")
	g.P("// Code generated by protoc-gen-sol. DO NOT EDIT.")
	g.P("// source: ", f.Name)
	g.P("pragma solidity ", SolVer)
	g.P()
	if g.importpb {
		g.P(`import "./Pb.sol";`)
	}
	for _, i := range f.Dependency {
		if i == "google/protobuf/descriptor.proto" {
			continue
		}
		// The output filename is derived from the dependency's declared
		// proto package, not from its file path. Looking the package up
		// here keeps the import line consistent with the file the other
		// proto's codegen actually emitted, even when the path uses `/`
		// segments (e.g. `foo/bar.proto`) and the package uses `.`
		// segments (e.g. `foo.bar`).
		depPkg := g.lookupDepPackage(i)
		g.P(`import "./`, getSolFile(depPkg), `";`)
	}
	g.P()
	g.P("/**")
	g.P(" * @title ", getSolLib(*f.Package))
	g.P(" * @notice **Auto-generated.** Solidity decoders for the protobuf messages defined in")
	g.P(" *  `", f.Name, "`. Do not hand-edit this file; change the proto schema and")
	g.P(" *  regenerate via `pb3-gen-sol`.")
	g.P(" */")
	g.P("library ", getSolLib(*f.Package), " {")
}

// lookupDepPackage returns the proto `package` declared by the dependency
// whose file path is `depPath`. Falls back to the legacy path-based name
// (`foo/bar.proto` → `foo/bar`) if the descriptor is not present in the
// request — e.g. running the plugin with a partial descriptor set. When a
// package is found, getSolFile/getSolLib will normalize multi-segment
// names like `foo.bar` to `PbFooBar` so the emitted import filename
// always matches the file the dependency's own codegen produces.
func (g *Generator) lookupDepPackage(depPath string) string {
	for _, pf := range g.Request.ProtoFile {
		if pf.GetName() == depPath {
			pkg := pf.GetPackage()
			if pkg != "" {
				return pkg
			}
			break
		}
	}
	return strings.TrimSuffix(depPath, ".proto")
}

func standaloneRuntimeContent() string {
	return runtimeSource()
}

func (g *Generator) appendRuntimeLibrary() {
	body := runtimeLibraryBody()
	g.WriteByte('\n')
	g.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		g.WriteByte('\n')
	}
}

func runtimeSource() string {
	return strings.TrimPrefix(ProtoSol, "\n")
}

func runtimeLibraryBody() string {
	source := runtimeSource()
	lines := strings.Split(source, "\n")
	index := 0
	if index < len(lines) && strings.HasPrefix(lines[index], "// SPDX-License-Identifier:") {
		index++
	}
	if index < len(lines) && strings.HasPrefix(lines[index], "// Code generated by protoc-gen-sol.") {
		index++
	}
	if index < len(lines) && strings.HasPrefix(lines[index], "pragma solidity ") {
		index++
	}
	for index < len(lines) && strings.TrimSpace(lines[index]) == "" {
		index++
	}
	if index > 0 {
		return strings.Join(lines[index:], "\n")
	}
	return source
}

func (g *Generator) generateEnum(e enumdes) {
	s := "enum " + *(e.Name) + " { "
	// assume enum definition is perfect (in order and no gaps)
	// TODO(enum): robust for disorders and gaps
	var values []string
	for i, v := range e.Value {
		values = append(values, *(v.Name))
		if int(*v.Number) != i {
			Fail("enum values must start from 0 and no skip numbers")
		}
	}
	s = s + strings.Join(values, ", ") + " }\n"
	g.P(s)

	g.P("// ", e.Name, "[] decode function")
	g.P("function ", e.Name, "s(uint256[] memory arr) internal pure returns (", e.Name, "[] memory t) {")
	g.In()
	g.P("t = new ", e.Name, "[](arr.length);")
	g.P("for (uint256 i = 0; i < t.length; i++) { t[i] = ", e.Name, "(arr[i]); }")
	g.Out()
	g.P("}\n")
}

// repeatedField captures the codegen state for one length-delimited repeated
// field. Every such field uses single-pass over-allocate + in-place shrink
// (no `cntTags` pre-pass).
//
// `typelessScratch` distinguishes the two scratch flavors:
//
//   - `false` (inline-primitive element types `bytes32` / `address` / `uint256`):
//     allocate the scratch as the actual element type's array. Each slot is
//     just a 32-byte zero word, no sub-allocation. Stores and the final
//     assignment are direct.
//   - `true` (reference element types `bytes` / `string` / embedded struct):
//     allocate the scratch as `uint256[]` to avoid per-slot zero-init of
//     fresh sub-allocations. Stores write the element pointer/value via
//     assembly `mstore`. At the end the scratch is aliased to the target
//     array type via assembly assignment, then assigned to the struct field.
type repeatedField struct {
	solName         string // struct field name in solidity (camelCase)
	elementType     string // element solidity type, no `[]` suffix
	tag             int32  // proto field number
	minWireSize     int    // minimum bytes per occurrence on the wire
	typelessScratch bool   // see doc above
}

// inlinePrimitiveSolTypes are element types whose `new T[](N)` does not
// trigger expensive per-element zero-initialization. Each slot is just a
// 32-byte zero word, so allocating directly as `T[] memory` is cheap.
var inlinePrimitiveSolTypes = map[string]bool{
	"bytes32":         true,
	"address":         true,
	"address payable": true,
	"uint256":         true,
}

func (g *Generator) generateMsg(m msgdes, currentPkg string, knownPkgs []string) {
	// map from tag(field number) to its decoder solidity code string
	tag2dec := make(map[int]string)
	// Pre-shifted dispatch key per known tag: `(tag << 3) | wire`. The
	// dispatch loop reads the raw key varint and compares against these
	// constants, which folds the per-field wire-type check into the same
	// EQ that selects the branch. A payload using a legal-but-wrong wire
	// type for a known tag falls through to the unknown-tag path and is
	// skipped per proto3 semantics — strictly safer than the pre-fix
	// behavior (which would have decoded with garbled data) and cheaper
	// than emitting a separate `require(wire == ...)` per branch.
	tag2key := make(map[int]int)

	g.P("struct ", m.Name, " {")
	g.In()
	var repeated []repeatedField
	for _, f := range m.Field {
		validatePackedOption(f)
		t := getSolType(f, g.extnum, currentPkg, knownPkgs)
		g.P(t, " ", toSolNaming(f.Name), ";", "   // tag: ", f.Number)
		tag2key[int(*f.Number)] = (int(*f.Number) << 3) | expectedWireNum(f)
		if isRepeated(f) && (getWiretype(*f.Type) == WireLendel) {
			elementType := strings.TrimSuffix(t, "[]")
			rf := repeatedField{
				solName:         toSolNaming(f.Name),
				elementType:     elementType,
				tag:             *f.Number,
				minWireSize:     minWireSize(getSolFieldSoltype(f, g.extnum)),
				typelessScratch: !inlinePrimitiveSolTypes[elementType],
			}
			repeated = append(repeated, rf)
			tag2dec[int(*f.Number)] = getSolDecodeStr(f, t, rf.typelessScratch)
		} else {
			tag2dec[int(*f.Number)] = getSolDecodeStr(f, t, false)
		}
	}
	g.Out()
	g.P("} ", "// end struct ", m.Name, "\n")

	// sorted tags
	stags := sortedTags(tag2dec)
	// generate decoder. we make decode function name the same as message to unify type cast
	// we use m for return struct name, saves us one g.P
	g.P("function ", getDecFname(*m.Name), "(bytes memory raw) internal pure returns (", m.Name, " memory m) {")
	g.In()
	g.P("Pb.Buffer memory buf = Pb.fromBytes(raw);\n")

	for _, rf := range repeated {
		if rf.typelessScratch {
			g.P(fmt.Sprintf("uint256[] memory _arr%d = new uint256[](raw.length / %d);", rf.tag, rf.minWireSize))
		} else {
			g.P(fmt.Sprintf("%s[] memory _arr%d = new %s[](raw.length / %d);", rf.elementType, rf.tag, rf.elementType, rf.minWireSize))
		}
		g.P(fmt.Sprintf("uint256 _cnt%d = 0;", rf.tag))
	}
	if len(repeated) > 0 {
		g.P()
	}
	// Dispatch reads the raw protobuf key varint (`(tag << 3) | wire`)
	// and compares it against pre-shifted constants. Folding tag+wire
	// into one EQ avoids the per-branch wire-type require and drops the
	// `decKey` split/cast. A known tag with the wrong wire falls through
	// to the unknown-tag path and is skipped per proto3 semantics.
	g.P("uint256 key;")
	g.P("while (buf.hasMore()) {")
	g.In()
	g.P("key = buf.decVarint();")
	if len(stags) == 0 {
		g.P("buf.skipValue(Pb.WireType(key & 7)); // skip value of unknown tag")
	} else {
		for index, k := range stags {
			if index == 0 {
				g.P("if (key == ", tag2key[k], ") { // tag ", k)
			} else {
				g.P("else if (key == ", tag2key[k], ") { // tag ", k)
			}
			g.In()
			g.P(strings.Replace(tag2dec[k], "{XXX_INDENT}", g.indent, -1))
			g.Out()
			g.P("}")
		}
		g.P("else { buf.skipValue(Pb.WireType(key & 7)); } // unknown tag or wrong wire")
	}
	g.Out()
	g.P("}")
	if len(repeated) > 0 {
		g.P()
		for _, rf := range repeated {
			if rf.typelessScratch {
				// Shrink the length and alias the typeless scratch to the
				// actual element-type array in one assembly block, then
				// assign to the struct field.
				g.P(fmt.Sprintf("%s[] memory _result%d;", rf.elementType, rf.tag))
				g.P(fmt.Sprintf(`assembly ("memory-safe") { mstore(_arr%d, _cnt%d) _result%d := _arr%d }`, rf.tag, rf.tag, rf.tag, rf.tag))
				g.P(fmt.Sprintf("m.%s = _result%d;", rf.solName, rf.tag))
			} else {
				g.P(fmt.Sprintf(`assembly ("memory-safe") { mstore(_arr%d, _cnt%d) }`, rf.tag, rf.tag))
				g.P(fmt.Sprintf("m.%s = _arr%d;", rf.solName, rf.tag))
			}
		}
	}
	g.Out()
	g.P("} ", "// end decoder ", m.Name, "\n")
	// TODO(oneof): check m.OneofDecl and generate struct members and funcs
}

// minWireSize returns the minimum number of bytes that one occurrence of a
// repeated length-delimited field can consume on the wire, used as the
// denominator for the over-allocation upper bound. The minimum encoding is
// 1 byte tag + 1 byte length-varint + payload-min, where payload-min is
// dictated by the soltype for fixed-width fields and 0 otherwise (empty
// bytes / strings / messages are valid encodings).
func minWireSize(soltype string) int {
	switch soltype {
	case "bytes32":
		return 1 + 1 + 32
	case "address", "address payable":
		return 1 + 1 + 20
	}
	return 1 + 1 + 0
}

// validatePackedOption rejects schemas that explicitly opt out of
// packed encoding for a repeated scalar or enum field via
// `[packed=false]`. The generator emits packed-only decode logic for
// these fields (combined-key dispatch matches LengthDelim), so an
// unpacked-on-the-wire payload would silently fall into the unknown-
// tag skip path and drop every element. Rejecting the schema up front
// avoids generating a decoder that would silently lose data.
//
// Length-delimited element types (bytes, string, message) are not
// affected — the `packed` option does not apply to them in the
// protobuf spec.
func validatePackedOption(f *descriptor.FieldDescriptorProto) {
	if !isRepeated(f) {
		return
	}
	if getWiretype(*f.Type) != WireVarint {
		return
	}
	if f.Options == nil || f.Options.Packed == nil {
		return
	}
	if !*f.Options.Packed {
		Fail("repeated scalar/enum field with [packed=false] is not supported", *f.Name)
	}
}

// expectedWireNum returns the protobuf wire-type number (0..5) that a
// known field's tag must arrive with on the wire.
//
//   - Repeated fields are always LengthDelim (2): packed scalars wrap
//     their elements in a single length-delimited payload, and repeated
//     bytes/string/message fields appear as one length-delimited
//     occurrence per element. (Unpacked repeated scalars are not
//     supported by this generator.)
//   - Non-repeated varint scalars (uint32, uint64, bool, enum, and the
//     `uint` soltype) decode with Varint (0).
//   - Non-repeated length-delimited fields (bytes, string, embedded
//     message, and the bytes-backed soltypes address / bytes32 / uint256)
//     decode with LengthDelim (2).
func expectedWireNum(f *descriptor.FieldDescriptorProto) int {
	if isRepeated(f) {
		return 2 // LengthDelim
	}
	if getWiretype(*f.Type) == WireVarint {
		return 0 // Varint
	}
	return 2 // LengthDelim
}

// getSolFieldSoltype returns the field's `(soltype)` extension value if
// present, otherwise empty. It is a thin wrapper over `getSolTypeOption`
// that also handles missing extensions.
func getSolFieldSoltype(field *descriptor.FieldDescriptorProto, extnum int32) string {
	if field.Options == nil || extnum == -1 {
		return ""
	}
	v, ok := getSolTypeOption(field.Options, extnum)
	if !ok {
		return ""
	}
	return v
}
func (g *Generator) shouldOutput(msgname string) bool {
	if len(g.onlymsgs) == 0 {
		return true
	}
	_, ok := g.onlymsgs[msgname]
	return ok
}

// helper functions below.

// whether s is in arr
func inArray(s string, arr []string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}

// return solidity code to decode this field. `typelessScratch` controls how
// the dispatch line for a repeated length-delimited field is shaped:
//
//   - `false` (inline-primitive element): write `_arr<tag>[_cnt<tag>] =
//     <decoder>;` and increment the counter `unchecked`. The scratch is the
//     real element-type array.
//   - `true` (reference element): decode into a typed temp local first, then
//     `mstore` the pointer/value into the typeless `uint256[]` scratch via
//     assembly. The scratch is later aliased back to the target array type
//     in `generateMsg`.
//
// Non-repeated fields are unaffected by `typelessScratch`.
func getSolDecodeStr(field *descriptor.FieldDescriptorProto, soltype string, typelessScratch bool) (code string) {
	// soltype could be uint256 or another message name
	soltype = strings.TrimSuffix(soltype, "[]") // remove [] for array, no-op if doesn't have it
	wire := getWiretype(*field.Type)
	// in proto3, repeated varints are default packed, we don't support option packed=false for now
	isPacked := isRepeated(field) && wire == WireVarint
	if isPacked {
		// buf.decPacked return uint[], use Pb.uintXXs to convert to uintXX[]
		if soltype == "uint" {
			code = fmt.Sprintf("m.%s = buf.decPacked();", toSolNaming(field.Name))
		} else if *field.Type == descriptor.FieldDescriptorProto_TYPE_ENUM {
			code = fmt.Sprintf("m.%s = %ss(buf.decPacked());", toSolNaming(field.Name), soltype)
		} else {
			code = fmt.Sprintf("m.%s = Pb.%ss(buf.decPacked());", toSolNaming(field.Name), soltype)
		}
		return
	}
	var decfun string
	if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
		decfun = fmt.Sprintf("%s(buf.decBytes())", getDecFname(soltype))
	} else if *field.Type == descriptor.FieldDescriptorProto_TYPE_ENUM {
		// ENUM needs an explicit cast from uint256.
		// Example: m.enum = EnumName(buf.decVarint());
		decfun = fmt.Sprintf("%s(buf.decVarint())", soltype)
	} else if wire == WireLendel && fixedWidthReader(soltype) != "" {
		// Bytes-backed soltype overrides skip the intermediate `bytes`
		// allocation by reading directly into the target type. See
		// `decAddress` / `decBytes32` / `decUint256` in the runtime.
		decfun = fixedWidthReader(soltype)
	} else if soltype == "bool" {
		decfun = "Pb._bool(buf.decVarint())"
	} else if (soltype == "uint" && wire == WireVarint) || (soltype == "bytes" && wire == WireLendel) {
		decfun = fmt.Sprintf("buf.dec%s()", wire)
	} else {
		// Native primitive cast: `uintN(buf.decVarint())`, `string(buf.decBytes())`, etc.
		decfun = fmt.Sprintf("%s(buf.dec%s())", soltype, wire)
	}

	if isRepeated(field) {
		// _arr<tag>/_cnt<tag> are declared at the top of the decoder body
		// (see generateMsg). The counter is bounded by the over-alloc upper
		// bound — unchecked is safe.
		if typelessScratch {
			// Decode into a typed temp local (Solidity needs the type so the
			// stack value carries the right pointer/value width), then mstore
			// into the uint256 scratch via assembly. shl(5, x) == mul(x, 32).
			elementType := strings.TrimSuffix(soltype, "[]")
			tempDecl := fmt.Sprintf("%s memory _v%d", elementType, *field.Number)
			code = fmt.Sprintf("%s = %s;\n", tempDecl, decfun)
			code += fmt.Sprintf("{XXX_INDENT}assembly (\"memory-safe\") { mstore(add(add(_arr%d, 32), shl(5, _cnt%d)), _v%d) }\n", *field.Number, *field.Number, *field.Number)
			code += fmt.Sprintf("{XXX_INDENT}unchecked { _cnt%d++; }", *field.Number)
		} else {
			code = fmt.Sprintf("_arr%d[_cnt%d] = %s;\n", *field.Number, *field.Number, decfun)
			code += fmt.Sprintf("{XXX_INDENT}unchecked { _cnt%d++; }", *field.Number)
		}
	} else {
		code = fmt.Sprintf("m.%s = %s;", toSolNaming(field.Name), decfun)
	}
	return
}

// fixedWidthReader returns the Pb runtime helper that decodes a length-
// delimited bytes-backed soltype directly into the target Solidity type,
// bypassing an intermediate `bytes` allocation. Empty result means there
// is no fixed-width reader for this soltype; the caller falls back to the
// generic `decBytes` path.
func fixedWidthReader(soltype string) string {
	switch soltype {
	case "address", "address payable":
		return "buf.decAddress()"
	case "bytes32":
		return "buf.decBytes32()"
	case "uint256":
		return "buf.decUint256()"
	}
	return ""
}

// getWiretype returns the proto wire-type string ("Varint" or "Bytes") for
// `fieldtype`. Packed-repeated handling is done by `getSolDecodeStr` (which
// switches to `decPacked` when a field is repeated and varint-typed); this
// function only reports the underlying scalar's wire type.
func getWiretype(fieldtype descriptor.FieldDescriptorProto_Type) string {
	if fieldtype == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
		return WireLendel
	} else if fieldtype == descriptor.FieldDescriptorProto_TYPE_ENUM {
		return WireVarint
	}
	s, ok := pbType2Str[fieldtype]
	if !ok {
		// fail here
		Fail("unsupported proto type", (&fieldtype).String())
	}
	wire, ok := PassTypeMap[s]
	if !ok {
		Fail("unsupported proto type", s)
	}
	return wire
}

// getSolType return solidity type as string
// if soltype option is set, uses that, otherwise use field.Type
// will also append [] if field is repeated
func getSolType(field *descriptor.FieldDescriptorProto, extnum int32, currentPkg string, knownPkgs []string) (s string) {
	// use solidity array for repeated field
	if isRepeated(field) {
		defer func() { s += "[]" }()
	}
	isMessage := *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE
	isEnum := *field.Type == descriptor.FieldDescriptorProto_TYPE_ENUM
	if isMessage || isEnum {
		return resolveProtoTypeName(*field.TypeName, currentPkg, knownPkgs)
	}
	// primitive types, check support and soltype option
	s, ok := pbType2Str[*field.Type]
	if !ok {
		Fail("unsupported proto type", (*field.Type).String())
	}

	if field.Options != nil && extnum != -1 {
		s2, ok := getSolTypeOption(field.Options, extnum)
		if ok {
			if s == SolTypeMap[s2] { // s matches s2 requirement
				s = s2
			} else {
				Fail("incompatible types", s, s2)
			}
		}
	}
	return
}

func resolveProtoTypeName(typeName string, currentPkg string, knownPkgs []string) string {
	qualified := strings.TrimPrefix(typeName, ".")
	for _, pkg := range knownPkgs {
		prefix := pkg + "."
		if !strings.HasPrefix(qualified, prefix) {
			continue
		}

		remainder := strings.TrimPrefix(qualified, prefix)
		if strings.Contains(remainder, ".") {
			Fail("unsupported name hierarchy", typeName, "nested types are not supported")
		}
		if pkg == currentPkg {
			return remainder
		}
		return getSolLib(pkg) + "." + remainder
	}

	Fail("unsupported name hierarchy", typeName, "could not resolve package name")
	return ""
}

func collectProtoPackages(files []*descriptor.FileDescriptorProto) []string {
	seen := make(map[string]struct{})
	packages := make([]string, 0, len(files))
	for _, file := range files {
		pkg := file.GetPackage()
		if pkg == "" {
			continue
		}
		if _, ok := seen[pkg]; ok {
			continue
		}
		seen[pkg] = struct{}{}
		packages = append(packages, pkg)
	}

	sort.Slice(packages, func(i, j int) bool {
		if len(packages[i]) == len(packages[j]) {
			return packages[i] > packages[j]
		}
		return len(packages[i]) > len(packages[j])
	})
	return packages
}

func getSolTypeOption(options *descriptor.FieldOptions, extnum int32) (string, bool) {
	unknown := options.ProtoReflect().GetUnknown()
	for len(unknown) > 0 {
		num, typ, n := protowire.ConsumeTag(unknown)
		if n < 0 {
			Error(protowire.ParseError(n), "parsing soltype option tag")
		}
		unknown = unknown[n:]

		if int32(num) == extnum {
			if typ != protowire.BytesType {
				Fail("soltype option has unsupported wire type")
			}
			value, n := protowire.ConsumeString(unknown)
			if n < 0 {
				Error(protowire.ParseError(n), "parsing soltype option value")
			}
			return value, true
		}

		n = protowire.ConsumeFieldValue(num, typ, unknown)
		if n < 0 {
			Error(protowire.ParseError(n), "skipping field option value")
		}
		unknown = unknown[n:]
	}

	return "", false
}

// get solidity library name from proto package name
// getSolLib("example") -> PbExample
func getSolLib(pkg string) string {
	if pkg == "" {
		Fail("empty package name")
	}
	libname := "Pb"
	cap := true
	for _, v := range pkg {
		if (v >= 'A' && v <= 'Z') || v >= '0' && v <= '9' {
			libname += string(v)
		} else if v >= 'a' && v <= 'z' {
			if cap {
				libname += strings.ToUpper(string(v))
				cap = false
			} else {
				libname += string(v)
			}
		} else if v == '_' || v == '.' || v == '-' {
			cap = true
		}
	}
	return libname
}

// get solidity library name from proto package name
// getSolFile("example") -> PbExample.sol
func getSolFile(pkg string) string {
	return getSolLib(pkg) + ".sol"
}

func getDecFname(name string) string {
	// if name has dot in it like pkg.Msg, we should return pkg.decMsg
	// otherwise just decMsg
	arr := strings.Split(name, ".")
	if len(arr) == 2 {
		return arr[0] + ".dec" + arr[1]
	}
	return "dec" + name
}

// sort by tag for stable map iteration order
func sortedTags(m map[int]string) (ret []int) {
	for k := range m {
		ret = append(ret, k)
	}
	sort.Ints(ret)
	return
}

// Iterate over defined extensions and if found ExtName, return its field number
// otherwise return -1
func getExtNum(current fdes, files []*descriptor.FileDescriptorProto) int32 {
	if extnum, ok := findExtNum(current.Extension); ok {
		return extnum
	}

	extnum := int32(-1)
	for _, file := range files {
		num, ok := findExtNum(file.Extension)
		if !ok {
			continue
		}
		if extnum == -1 {
			extnum = num
			continue
		}
		if extnum != num {
			Fail(
				"conflicting soltype extension numbers",
				strconv.FormatInt(int64(extnum), 10),
				strconv.FormatInt(int64(num), 10),
			)
		}
	}
	return extnum
}

func findExtNum(extensions []*descriptor.FieldDescriptorProto) (int32, bool) {
	for _, ext := range extensions {
		if ext.GetName() == ExtName {
			return ext.GetNumber(), true
		}
	}
	return -1, false
}

// Is this field repeated?
func isRepeated(field *descriptor.FieldDescriptorProto) bool {
	return field.Label != nil && *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED
}

// Error reports a problem, including an error, and exits the program.
func Error(err error, msgs ...string) {
	s := strings.Join(msgs, " ") + ":" + err.Error()
	log.Print("error: ", s)
	os.Exit(1)
}

// Fail reports a problem and exits the program.
func Fail(msgs ...string) {
	s := strings.Join(msgs, " ")
	log.Print("error: ", s)
	os.Exit(1)
}

// toSolNaming transforms proto's naming style to solidity's, e.g. var_name_one to varNameOne
func toSolNaming(name *string) string {
	var re = regexp.MustCompile(`_[a-z]`)
	s := re.ReplaceAllStringFunc(*name, func(m string) string { return strings.ToUpper(m[1:]) })
	return s
}

// ProtoSol is the embedded Solidity runtime library body used for both inline
// output and the standalone Pb.sol artifact.
//
//go:embed Pb.runtime.sol
var ProtoSol string
