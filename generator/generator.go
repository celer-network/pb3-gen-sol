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

// When we want to support multiple .proto and imports, need preprocess to get all definition relationships
func (g *Generator) Preprocess() {
	// init text template?
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
		if i != "google/protobuf/descriptor.proto" {
			// require proto file name and package name are same
			g.P(`import "./`, getSolFile(strings.TrimSuffix(i, ".proto")), `";`)
		}
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

func (g *Generator) generateMsg(m msgdes, currentPkg string, knownPkgs []string) {
	// map from tag(field number) to its decoder solidity code string
	tag2dec := make(map[int]string)

	g.P("struct ", m.Name, " {")
	g.In()
	// because solidity doesn't support dynamic sized memory array
	// we need to count tag(field number) occurrences for repeated bytes or messages
	// then new the correct size array.
	// repeated uint doesn't need this because it's packed
	needNew := []string{"uint256[] memory cnts = buf.cntTags({MAX_TAG});"}
	// go over fields and put decode string into tag2dec
	for _, f := range m.Field {
		t := getSolType(f, g.extnum, currentPkg, knownPkgs)
		g.P(t, " ", toSolNaming(f.Name), ";", "   // tag: ", f.Number)
		tag2dec[int(*f.Number)] = getSolDecodeStr(f, t)
		if isRepeated(f) && (getWiretype(*f.Type) == WireLendel) {
			needNew = append(needNew, fmt.Sprintf("m.%s = new %s(cnts[%d]);", toSolNaming(f.Name), t, *f.Number))
			needNew = append(needNew, fmt.Sprintf("cnts[%d] = 0;  // reset counter for later use", *f.Number))
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
	if len(needNew) > 1 { // some fields need to new
		g.P(strings.Replace(needNew[0], "{MAX_TAG}", strconv.Itoa(stags[len(stags)-1]), 1)) // replace placeholder w/ actual max tag number
		for _, s := range needNew[1:] {
			g.P(s)
		}
		g.P()
	}
	g.P("uint256 tag;")
	g.P("Pb.WireType wire;")
	g.P("while (buf.hasMore()) {")
	g.In()
	g.P("(tag, wire) = buf.decKey();")
	if len(stags) == 0 {
		g.P("buf.skipValue(wire); // skip value of unknown tag")
	} else {
		for index, k := range stags {
			if index == 0 {
				g.P("if (tag == ", k, ") {")
			} else {
				g.P("else if (tag == ", k, ") {")
			}
			g.In()
			g.P(strings.Replace(tag2dec[k], "{XXX_INDENT}", g.indent, -1))
			g.Out()
			g.P("}")
		}
		g.P("else { buf.skipValue(wire); } // skip value of unknown tag")
	}
	g.Out()
	g.P("}")
	g.Out()
	g.P("} ", "// end decoder ", m.Name, "\n")
	// TODO(oneof): check m.OneofDecl and generate struct members and funcs
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

// return solidity code to decode this field
func getSolDecodeStr(field *descriptor.FieldDescriptorProto, soltype string) (code string) {
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
	// additional optimization can be done to only cast if soltype != decXXX native types
	if *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
		soltype = getDecFname(soltype) // use decMsg for msg decoder
	} else if *field.Type == descriptor.FieldDescriptorProto_TYPE_ENUM {
		// ENUM only needs an explicit conversion, so it doesn't need to change soltype at all
		// Example: m.enum = EnumName(buf.decVarint());
	} else {
		_, ok := SolTypeMap[soltype]
		if soltype == "address payable" {
			soltype = "Pb._addressPayable" // for address payable
		} else if (ok && wire == WireLendel) || soltype == "bool" {
			soltype = "Pb._" + soltype // if sol type like uint256, need special conv func in Pb library
		}
	}

	decodeCall := fmt.Sprintf("buf.dec%s()", wire)
	decfun := decodeCall
	if !((soltype == "uint" && wire == WireVarint) || (soltype == "bytes" && wire == WireLendel)) {
		decfun = fmt.Sprintf("%s(%s)", soltype, decodeCall)
	}

	if isRepeated(field) {
		code = fmt.Sprintf("m.%s[cnts[%d]] = %s;\n", toSolNaming(field.Name), *field.Number, decfun)
		code += fmt.Sprintf("{XXX_INDENT}cnts[%d]++;", *field.Number)
	} else {
		code = fmt.Sprintf("m.%s = %s;", toSolNaming(field.Name), decfun)
	}
	return
}

// wiretype string, WireVarint or WireLendel
// packed ints is handled by getPbDecFunc
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
