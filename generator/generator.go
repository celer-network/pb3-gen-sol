// protoc-gen-sol by Celer Network Team

/*
	The code generator for the plugin for the Google protocol buffer compiler.
	It generates Solidity code from the protocol buffer description files read by the
	main routine.
*/
package generator

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

// TODO(template?): consider use text/template

// TODO(oneof): support Oneof field

// TODO(nested): support Nested msg/enum definition within message

// ExtName is the extension name to google.protobuf.FieldOptions
// its type must be string. Valid string values are keys in SolTypeMap
// (we don't use enum to avoid having solidity knowledge in chain.proto)
const ExtName = "soltype"

// SolVer is the compatible solidity/solc version in pragma solidity
const SolVer = "^0.5.0;"

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

// current proto package name
var curPkg string

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
		tmp := strings.Split(p, "=")
		key, value := tmp[0], tmp[1]
		switch key {
		case "msg":
			g.onlymsgs[value] = true
		case "importpb":
			if value == "true" {
				g.importpb = true
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
			Content: proto.String("pragma solidity " + SolVer + "\n" + ProtoSol),
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
	g.extnum = getExtNum(f)
	curPkg = *f.Package
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
			g.generateMsg(msg)
		}
	}
	g.Out()
	g.P("}") // close library
	if !g.importpb {
		g.P(ProtoSol)
	}
}

// Generate the header, including package definition
func (g *Generator) generateHeader(f fdes) {
	g.P("// Code generated by protoc-gen-sol. DO NOT EDIT.")
	g.P("// source: ", f.Name)
	g.P("pragma solidity ", SolVer)
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
	g.P("library ", getSolLib(*f.Package), " {")
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
	g.P("function ", e.Name, "s(uint[] memory arr) internal pure returns (", e.Name, "[] memory t) {")
	g.In()
	g.P("t = new ", e.Name, "[](arr.length);")
	g.P("for (uint i = 0; i < t.length; i++) { t[i] = ", e.Name, "(arr[i]); }")
	g.Out()
	g.P("}\n")
}

func (g *Generator) generateMsg(m msgdes) {
	// map from tag(field number) to its decoder solidity code string
	tag2dec := make(map[int]string)

	g.P("struct ", m.Name, " {")
	g.In()
	// because solidity doesn't support dynamic sized memory array
	// we need to count tag(field number) occurrences for repeated bytes or messages
	// then new the correct size array.
	// repeated uint doesn't need this because it's packed
	needNew := []string{"uint[] memory cnts = buf.cntTags({MAX_TAG});"}
	// go over fields and put decode string into tag2dec
	for _, f := range m.Field {
		t := getSolType(f, g.extnum)
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
	g.P("uint tag;")
	g.P("Pb.WireType wire;")
	g.P("while (buf.hasMore()) {")
	g.In()
	g.P("(tag, wire) = buf.decKey();")
	g.P("if (false) {} // solidity has no switch/case")
	// have to use if clause because solidity doesn't support switch
	for _, k := range stags {
		g.P("else if (tag == ", k, ") {")
		g.In()
		g.P(strings.Replace(tag2dec[k], "{XXX_INDENT}", g.indent, -1))
		g.Out()
		g.P("}")
	}
	g.P("else { buf.skipValue(wire); } // skip value of unknown tag")
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

	decfun := fmt.Sprintf("%s(buf.dec%s())", soltype, wire)

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
func getSolType(field *descriptor.FieldDescriptorProto, extnum int32) (s string) {
	// use solidity array for repeated field
	if isRepeated(field) {
		defer func() { s += "[]" }()
	}
	isMessage := *field.Type == descriptor.FieldDescriptorProto_TYPE_MESSAGE
	isEnum := *field.Type == descriptor.FieldDescriptorProto_TYPE_ENUM
	if isMessage || isEnum {
		// TypeName is fullyqualified name eg. .pkg1.pkg2.mymsg.submsg
		// only support 2 layers like .pkg.msg or .pkg.enum now
		// Need to update this to properly support proto import/namespace/nested definition
		fqn := strings.Split(*field.TypeName, ".")
		if len(fqn) != 3 {
			Fail("unsupported name hierarchy", *field.TypeName, "Expect .pkg.msg")
		}
		if fqn[1] == curPkg { // within same package, use only msg/enum name
			s = fqn[2]
		} else {
			s = getSolLib(fqn[1]) + "." + fqn[2]
		}
		return
	}
	// primitive types, check support and soltype option
	s, ok := pbType2Str[*field.Type]
	if !ok {
		Fail("unsupported proto type", (*field.Type).String())
	}

	if field.Options != nil && extnum != -1 {
		v, err := proto.GetExtension(field.Options, &proto.ExtensionDesc{Field: extnum})
		if err == nil {
			b := proto.NewBuffer(v.([]byte))
			b.DecodeVarint() // tag
			s2, err := b.DecodeStringBytes()
			if err == nil && s == SolTypeMap[s2] { // s matches s2 requirement
				s = s2
			} else {
				Fail("incompatible types", s, s2)
			}
		}
	}
	return
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
func getExtNum(f fdes) int32 {
	for _, ext := range f.Extension {
		if *ext.Name == ExtName {
			return *ext.Number
		}
	}
	return -1
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

// ProtoSol is the full proto library, appended in the end of generated .sol file
const ProtoSol = `
// runtime proto sol library
library Pb {
    enum WireType { Varint, Fixed64, LengthDelim, StartGroup, EndGroup, Fixed32 }
    struct Buffer {
        uint idx;  // the start index of next read. when idx=b.length, we're done
        bytes b;   // hold serialized proto msg, readonly
    }
    // create a new in-memory Buffer object from raw msg bytes
    function fromBytes(bytes memory raw) internal pure returns (Buffer memory buf) {
        require(raw.length > 1); // min length of a valid Pb msg is 2
        buf.b = raw;
        buf.idx = 0;
    }
    // whether there are unread bytes
    function hasMore(Buffer memory buf) internal pure returns (bool) {
        return buf.idx < buf.b.length;
    }
    // decode current field number and wiretype
    function decKey(Buffer memory buf) internal pure returns (uint tag, WireType wiretype) {
        uint v = decVarint(buf);
        tag = v / 8;
        wiretype = WireType(v & 7);
    }
    // count tag occurrences, return an array due to no memory map support
	// have to create array for (maxtag+1) size. cnts[tag] = occurrences
	// should keep buf.idx unchanged because this is only a count function
    function cntTags(Buffer memory buf, uint maxtag) internal pure returns (uint[] memory cnts) {
        uint originalIdx = buf.idx;
        cnts = new uint[](maxtag+1);  // protobuf's tags are from 1 rather than 0
        uint tag;
        WireType wire;
        while (hasMore(buf)) {
            (tag, wire) = decKey(buf);
            cnts[tag] += 1;
            skipValue(buf, wire);
        }
        buf.idx = originalIdx;
    }
    // read varint from current buf idx, move buf.idx to next read, return the int value
    function decVarint(Buffer memory buf) internal pure returns (uint v) {
        bytes10 tmp;  // proto int is at most 10 bytes (7 bits can be used per byte)
        bytes memory bb = buf.b;  // get buf.b mem addr to use in assembly
        v = buf.idx;  // use v to save one additional uint variable
        assembly {
            tmp := mload(add(add(bb, 32), v)) // load 10 bytes from buf.b[buf.idx] to tmp
        }
        uint b; // store current byte content
        v = 0; // reset to 0 for return value
        for (uint i=0; i<10; ++i) {
            assembly {
                b := byte(i, tmp)  // don't use tmp[i] because it does bound check and costs extra
            }
            v |= (b & 0x7F) << (i * 7);
            if (b & 0x80 == 0) {
                buf.idx += i + 1;
                return v;
            }
        }
        revert(); // i=10, invalid varint stream
    }
    // read length delimited field and return bytes
    function decBytes(Buffer memory buf) internal pure returns (bytes memory b) {
        uint len = decVarint(buf);
        uint end = buf.idx + len;
        require(end <= buf.b.length);  // avoid overflow
        b = new bytes(len);
        bytes memory bufB = buf.b;  // get buf.b mem addr to use in assembly
        uint bStart;
        uint bufBStart = buf.idx;
        assembly {
            bStart := add(b, 32)
            bufBStart := add(add(bufB, 32), bufBStart)
        }
        for (uint i=0; i<len; i+=32) {
            assembly{
                mstore(add(bStart, i), mload(add(bufBStart, i)))
            }
        }
        buf.idx = end;
    }
    // return packed ints
    function decPacked(Buffer memory buf) internal pure returns (uint[] memory t) {
        uint len = decVarint(buf);
        uint end = buf.idx + len;
        require(end <= buf.b.length);  // avoid overflow
        // array in memory must be init w/ known length
        // so we have to create a tmp array w/ max possible len first
        uint[] memory tmp = new uint[](len);
        uint i = 0; // count how many ints are there
        while (buf.idx < end) {
            tmp[i] = decVarint(buf);
            i++;
        }
        t = new uint[](i); // init t with correct length
        for (uint j=0; j<i; j++) {
            t[j] = tmp[j];
        }
        return t;
    }
    // move idx pass current value field, to beginning of next tag or msg end
    function skipValue(Buffer memory buf, WireType wire) internal pure {
        if (wire == WireType.Varint) { decVarint(buf); }
        else if (wire == WireType.LengthDelim) {
            uint len = decVarint(buf);
            buf.idx += len; // skip len bytes value data
            require(buf.idx <= buf.b.length);  // avoid overflow
        } else { revert(); }  // unsupported wiretype
    }

    // type conversion help utils
    function _bool(uint x) internal pure returns (bool v) {
        return x != 0;
    }
    function _uint256(bytes memory b) internal pure returns (uint256 v) {
        assembly { v := mload(add(b, 32)) }  // load all 32bytes to v
        v = v >> (8 * (32 - b.length));  // only first b.length is valid
	}
	function _address(bytes memory b) internal pure returns (address v) {
        v = _addressPayable(b);
    }
    function _addressPayable(bytes memory b) internal pure returns (address payable v) {
        require(b.length == 20);

        //load 32bytes then shift right 12 bytes
        assembly { v := div(mload(add(b, 32)), 0x1000000000000000000000000) }
    }
    function _bytes32(bytes memory b) internal pure returns (bytes32 v) {
        require(b.length == 32);

        assembly { v := mload(add(b, 32)) }
    }
    // uint[] to uint8[]
    function uint8s(uint[] memory arr) internal pure returns (uint8[] memory t) {
        t = new uint8[](arr.length);
        for (uint i = 0; i < t.length; i++) { t[i] = uint8(arr[i]); }
    }
    function uint32s(uint[] memory arr) internal pure returns (uint32[] memory t) {
        t = new uint32[](arr.length);
        for (uint i = 0; i < t.length; i++) { t[i] = uint32(arr[i]); }
    }
    function uint64s(uint[] memory arr) internal pure returns (uint64[] memory t) {
        t = new uint64[](arr.length);
        for (uint i = 0; i < t.length; i++) { t[i] = uint64(arr[i]); }
    }
    function bools(uint[] memory arr) internal pure returns (bool[] memory t) {
        t = new bool[](arr.length);
        for (uint i = 0; i < t.length; i++) { t[i] = arr[i]!=0; }
    }
}
`
