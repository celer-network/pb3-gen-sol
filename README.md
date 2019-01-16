# Overview

pb3-gen-sol is a proto3 to solidity library generator that supports proto3 native types and uses field option for solidity native types. Both the message and generated code are more efficient than other solutions. It also includes a library to decode protobuf wireformat. Currently it generates decoders, with encoder support planned.

## Usage
### .proto files
Example:
```protobuf
syntax = "proto3";
// package name is used for both generated .sol file name and library name
package mytest;

import "google/protobuf/descriptor.proto";
extend google.protobuf.FieldOptions {
    string soltype = 54321;  // must > 1001 and not conflict with other extensions
}

message MyMsg {
    bytes addr = 1 [ (soltype) = "address" ];
    uint32 num = 2 [ (soltype) = "uint8" ];
	bool has_moon = 3;  // for matching native types, no need for soltype option
}
```
Check .proto files unter test folder for more examples.

### Generate solidity library
Run

```bash
$ go build -o protoc-gen-sol
$ export PATH=$PWD:$PATH
$ protoc --sol_out=. [list of proto files]
```

proto files can have different package names, and generated .sol file name is proto package name.

## Params
- `msg`: only generate solidity struct and decode functions for msg name. Multiple can be specified.
- `importpb`: default false, if set to true, generated .sol file will import pb.sol instead of embed library pb in the file

Example:

```$ protoc --sol_out=msg=Msg1,msg=Msg2,msg=Msg3,importpb=true:test/solidity/contracts/lib/ test/test.proto```

## Kwown Issues
- No support for int32/int64
- Support embedded message/enum, but no nested message/enum definition
- Only support one layer proto package

# Contributors
- [stevenlcf](https://github.com/stevenlcf)