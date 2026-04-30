package generator

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
)

func TestParseParamsAcceptsKnownOptions(t *testing.T) {
	t.Parallel()

	g := New()
	parameter := "msg=Msg1,msg=Msg2,importpb=true"
	g.Request.Parameter = &parameter

	g.ParseParams()

	if !g.importpb {
		t.Fatal("expected importpb to be enabled")
	}
	if !g.onlymsgs["Msg1"] || !g.onlymsgs["Msg2"] {
		t.Fatalf("expected selected messages to be recorded, got %#v", g.onlymsgs)
	}
}

func TestGetSolTypeOptionReadsExtensionValue(t *testing.T) {
	t.Parallel()

	options := &descriptor.FieldOptions{}
	unknown := protowire.AppendTag(nil, 999, protowire.VarintType)
	unknown = protowire.AppendVarint(unknown, 7)
	unknown = appendSolTypeOption(unknown, 1001, "uint")
	options.ProtoReflect().SetUnknown(unknown)

	got, ok := getSolTypeOption(options, 1001)
	if !ok {
		t.Fatal("expected soltype option to be found")
	}
	if got != "uint" {
		t.Fatalf("expected soltype uint, got %q", got)
	}
}

func TestGetSolTypeAppliesCompatibleOverrides(t *testing.T) {
	t.Parallel()

	fieldTests := []struct {
		name  string
		field *descriptor.FieldDescriptorProto
		want  string
	}{
		{
			name:  "uint64 to uint",
			field: newPrimitiveField("ts", descriptor.FieldDescriptorProto_TYPE_UINT64, descriptor.FieldDescriptorProto_LABEL_OPTIONAL, newFieldOptions(1001, "uint")),
			want:  "uint",
		},
		{
			name:  "bytes to uint256",
			field: newPrimitiveField("amt", descriptor.FieldDescriptorProto_TYPE_BYTES, descriptor.FieldDescriptorProto_LABEL_OPTIONAL, newFieldOptions(1001, "uint256")),
			want:  "uint256",
		},
		{
			name:  "repeated uint64 to uint array",
			field: newPrimitiveField("tss", descriptor.FieldDescriptorProto_TYPE_UINT64, descriptor.FieldDescriptorProto_LABEL_REPEATED, newFieldOptions(1001, "uint")),
			want:  "uint[]",
		},
	}

	for _, tt := range fieldTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSolType(tt.field, 1001, "", nil); got != tt.want {
				t.Fatalf("expected Solidity type %q, got %q", tt.want, got)
			}
		})
	}
}

func TestGetSolTypeResolvesQualifiedNames(t *testing.T) {
	t.Parallel()

	knownPkgs := []string{"foo.bar", "other.pkg"}
	fieldTests := []struct {
		name       string
		currentPkg string
		field      *descriptor.FieldDescriptorProto
		want       string
	}{
		{
			name:       "same package message",
			currentPkg: "foo.bar",
			field:      newQualifiedField("msg", descriptor.FieldDescriptorProto_TYPE_MESSAGE, ".foo.bar.Msg"),
			want:       "Msg",
		},
		{
			name:       "cross package enum",
			currentPkg: "foo.bar",
			field:      newQualifiedField("enumValue", descriptor.FieldDescriptorProto_TYPE_ENUM, ".other.pkg.MyEnum"),
			want:       "PbOtherPkg.MyEnum",
		},
	}

	for _, tt := range fieldTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSolType(tt.field, -1, tt.currentPkg, knownPkgs); got != tt.want {
				t.Fatalf("expected Solidity type %q, got %q", tt.want, got)
			}
		})
	}
}

func TestImportPathDerivedFromDependencyPackage(t *testing.T) {
	t.Parallel()

	// A dependency whose proto path uses `/` separators while its package
	// uses `.` separators. The legacy code derived the import filename
	// from the path (yielding `PbFoobar.sol`), but the dependency's own
	// codegen emits `PbFooBar.sol`. The import line must match the
	// dependency's actual filename.
	depPath := "foo/bar.proto"
	depPkg := "foo.bar"
	depFile := &descriptor.FileDescriptorProto{
		Name:    proto.String(depPath),
		Package: proto.String(depPkg),
		Syntax:  proto.String("proto3"),
	}

	mainPath := "main.proto"
	mainPkg := "main"
	mainFile := &descriptor.FileDescriptorProto{
		Name:       proto.String(mainPath),
		Package:    proto.String(mainPkg),
		Syntax:     proto.String("proto3"),
		Dependency: []string{depPath},
	}

	g := New()
	g.Request.ProtoFile = []*descriptor.FileDescriptorProto{depFile, mainFile}
	g.generateHeader(mainFile)

	output := g.String()
	wantImport := `import "./PbFooBar.sol";`
	if !strings.Contains(output, wantImport) {
		t.Fatalf("expected import %q, got:\n%s", wantImport, output)
	}
	// Sanity: the legacy path-derived form must not appear.
	if strings.Contains(output, `PbFoobar`) {
		t.Fatalf("did not expect legacy path-derived filename, got:\n%s", output)
	}
}

func TestLookupDepPackageFallsBackToPath(t *testing.T) {
	t.Parallel()

	// Without a matching descriptor in the request, lookup falls back to
	// the path-with-extension-trimmed legacy behavior. Keeps the codegen
	// usable when invoked with a partial descriptor set.
	g := New()
	if got := g.lookupDepPackage("foo/bar.proto"); got != "foo/bar" {
		t.Fatalf("expected fallback %q, got %q", "foo/bar", got)
	}
}

func TestGetExtNumFallsBackToImportedDefinition(t *testing.T) {
	t.Parallel()

	extName := ExtName
	extNum := int32(1001)
	current := &descriptor.FileDescriptorProto{}
	imported := &descriptor.FileDescriptorProto{
		Extension: []*descriptor.FieldDescriptorProto{{
			Name:   &extName,
			Number: &extNum,
		}},
	}

	if got := getExtNum(current, []*descriptor.FileDescriptorProto{current, imported}); got != extNum {
		t.Fatalf("expected imported extension number %d, got %d", extNum, got)
	}
}

func TestGeneratorFatalPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		helperArg     string
		wantSubstring string
	}{
		{
			name:          "reject malformed parameter",
			helperArg:     "parse-invalid",
			wantSubstring: "invalid parameter",
		},
		{
			name:          "reject invalid importpb value",
			helperArg:     "parse-invalid-importpb",
			wantSubstring: "invalid importpb value",
		},
		{
			name:          "reject incompatible soltype override",
			helperArg:     "getsoltype-incompatible",
			wantSubstring: "incompatible types",
		},
		{
			name:          "reject nested type names",
			helperArg:     "getsoltype-nested",
			wantSubstring: "nested types are not supported",
		},
		{
			name:          "reject [packed=false] repeated scalar",
			helperArg:     "validate-packed-false",
			wantSubstring: "[packed=false] is not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := exec.Command(os.Args[0], "-test.run=TestGeneratorFatalHelperProcess", "--", tt.helperArg)
			cmd.Env = append(os.Environ(), "PB3_GEN_SOL_FATAL_HELPER=1")

			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected helper to fail, output: %s", output)
			}

			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("expected exit error, got %T (%v)", err, err)
			}
			if exitErr.ExitCode() != 1 {
				t.Fatalf("expected exit code 1, got %d, output: %s", exitErr.ExitCode(), output)
			}
			if !strings.Contains(string(output), tt.wantSubstring) {
				t.Fatalf("expected output to contain %q, got %s", tt.wantSubstring, output)
			}
		})
	}
}

func TestGeneratorFatalHelperProcess(t *testing.T) {
	if os.Getenv("PB3_GEN_SOL_FATAL_HELPER") != "1" {
		return
	}

	helperArg := os.Args[len(os.Args)-1]
	switch helperArg {
	case "parse-invalid":
		g := New()
		parameter := "msg"
		g.Request.Parameter = &parameter
		g.ParseParams()
	case "parse-invalid-importpb":
		g := New()
		parameter := "importpb=maybe"
		g.Request.Parameter = &parameter
		g.ParseParams()
	case "getsoltype-incompatible":
		field := newPrimitiveField("ts", descriptor.FieldDescriptorProto_TYPE_UINT64, descriptor.FieldDescriptorProto_LABEL_OPTIONAL, newFieldOptions(1001, "uint256"))
		_ = getSolType(field, 1001, "", nil)
	case "getsoltype-nested":
		field := newQualifiedField("nested", descriptor.FieldDescriptorProto_TYPE_MESSAGE, ".foo.bar.Outer.Inner")
		_ = getSolType(field, -1, "foo.bar", []string{"foo.bar"})
	case "validate-packed-false":
		packed := false
		field := newPrimitiveField(
			"nums",
			descriptor.FieldDescriptorProto_TYPE_UINT32,
			descriptor.FieldDescriptorProto_LABEL_REPEATED,
			&descriptor.FieldOptions{Packed: &packed},
		)
		validatePackedOption(field)
	default:
		os.Exit(2)
	}

	os.Exit(0)
}

func newFieldOptions(extnum int32, soltype string) *descriptor.FieldOptions {
	options := &descriptor.FieldOptions{}
	options.ProtoReflect().SetUnknown(appendSolTypeOption(nil, extnum, soltype))
	return options
}

func appendSolTypeOption(dst []byte, extnum int32, soltype string) []byte {
	dst = protowire.AppendTag(dst, protowire.Number(extnum), protowire.BytesType)
	return protowire.AppendString(dst, soltype)
}

func newPrimitiveField(
	name string,
	fieldType descriptor.FieldDescriptorProto_Type,
	label descriptor.FieldDescriptorProto_Label,
	options *descriptor.FieldOptions,
) *descriptor.FieldDescriptorProto {
	number := int32(1)
	return &descriptor.FieldDescriptorProto{
		Name:    &name,
		Number:  &number,
		Label:   &label,
		Type:    &fieldType,
		Options: options,
	}
}

func newQualifiedField(name string, fieldType descriptor.FieldDescriptorProto_Type, typeName string) *descriptor.FieldDescriptorProto {
	number := int32(1)
	label := descriptor.FieldDescriptorProto_LABEL_OPTIONAL
	return &descriptor.FieldDescriptorProto{
		Name:     &name,
		Number:   &number,
		Label:    &label,
		Type:     &fieldType,
		TypeName: &typeName,
	}
}
