#!/bin/bash

# remove old sol and pb files
rm -f *.pb
rm -f solidity/contracts/lib/*.sol

# generate new sol files
export PATH="$TRAVIS_BUILD_DIR:$PATH"
protoc --sol_out=importpb=true:solidity/contracts/lib/ test.proto a.proto b.proto

# generate new pb files
for pathname in *.textpb; do
    filename=${pathname##*/}
    basename=${filename%.textpb}
    msg=${basename%%_*}
    msgno=${msg#msg}
    cat "$basename".textpb|protoc  --encode=mytest.Msg"$msgno" test.proto > "$basename".pb || continue
done

# generate binary msg for b to test import
protoc --encode=b.B b.proto < b.pbtxt > b.pb
