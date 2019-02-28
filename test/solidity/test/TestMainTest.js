var fs = require('fs');
var path = require('path');
var protobuf = require("protobufjs");
const Web3 = require('web3');
const web3 = new Web3(new Web3.providers.HttpProvider('http://localhost:8545'));

const TestMain = artifacts.require('TestMain');

contract('TestMain', async accounts => {
    let testMain;

    before(async () => {
        testMain = await TestMain.new();
    });

    it('should decode msg1 (only native types) correctly', async () => {
        const buf = fs.readFileSync(path.join(__dirname, "../../msg1.pb"));
        const raw = '0x' + buf.toString('hex');

        const receipt = await testMain.testMsg1(raw);

        assert.equal(receipt.logs[0].event, 'Msg1Part1');
        assert.equal(receipt.logs[0].args.f1.toString(), '1');
        assert.equal(receipt.logs[0].args.f2.toString(), '12');
        assert.equal(receipt.logs[0].args.f3.toString(), 'true');
        assert.equal(receipt.logs[0].args.f4.toString(), '0x01020304');
        assert.equal(receipt.logs[0].args.f5.toString(), 'abc');
        assert.equal(receipt.logs[0].args.f6.toString(), [1, 2, 3]);
        assert.equal(receipt.logs[0].args.f7.toString(), [11, 12, 13]);

        assert.equal(receipt.logs[1].event, 'Msg1Part2');
        assert.equal(receipt.logs[1].args.f8.toString(), [true, false, true]);
        assert.equal(receipt.logs[1].args.f9_0.toString(), '0x0102');
        assert.equal(receipt.logs[1].args.f9_1.toString(), '0x0304');
        assert.equal(receipt.logs[1].args.f10_0.toString(), 'ab');
        assert.equal(receipt.logs[1].args.f10_1.toString(), 'cd');
    });

    it('should decode msg1 (only native types) with large numbers correctly', async () => {
        const buf = fs.readFileSync(path.join(__dirname, "../../msg1_large_number.pb"));
        const raw = '0x' + buf.toString('hex');

        const receipt = await testMain.testMsg1(raw);

        assert.equal(receipt.logs[0].event, 'Msg1Part1');
        assert.equal(receipt.logs[0].args.f1.toString(), '4294967295');
        assert.equal(receipt.logs[0].args.f2.toString(), '18446744073709551615');
        assert.equal(receipt.logs[0].args.f6.toString(), '4294967295,4294967294,4294967293');
        assert.equal(receipt.logs[0].args.f7.toString(), '18446744073709551615,18446744073709551614,18446744073709551613');
    });

    it('should decode msg2 (using soltype) correctly', async () => {
        const buf = fs.readFileSync(path.join(__dirname, "../../msg2.pb"));
        const raw = '0x' + buf.toString('hex');

        const receipt = await testMain.testMsg2(raw);

        assert.equal(receipt.logs[0].event, 'Msg2Part1');
        assert.equal(receipt.logs[0].args.num.toString(), '12');
        assert.equal(receipt.logs[0].args.addr.toString().toLowerCase(), '0x0002030405060708090a0b0c0d0e0f1011121314');
        assert.equal(receipt.logs[0].args.addrPayable.toString().toLowerCase(), '0x0102030405060708090a0b0c0d0e0f1011121314');
        assert.equal(receipt.logs[0].args.amt.toString(), '117967114');
        assert.equal(receipt.logs[0].args.h.toString().toLowerCase(), '0x0101010101010101010101010101010101010101010101010101010101010101');
        assert.equal(receipt.logs[0].args.nums.toString(), [12, 34, 56]);
        const expectedAddrs1 = ['0x0002030405060708090a0b0c0d0e0f1011121314','0x0b0c0d0e0f10111213140102030405060708090a','0xff02030405060708090a0b0c0d0e0f1011121314'];
        assert.equal(receipt.logs[0].args.addrs.toString().toLowerCase(), expectedAddrs1);
        const expectedAddrs2 = ['0x0102030405060708090a0b0c0d0e0f1011121314','0x0b0c0d0e0f10111213140102030405060708090a','0xff02030405060708090a0b0c0d0e0f1011121314'];
        assert.equal(receipt.logs[0].args.addrPayables.toString().toLowerCase(), expectedAddrs2);

        assert.equal(receipt.logs[1].event, 'Msg2Part2');
        assert.equal(receipt.logs[1].args.amts.toString(), [1, 257, 65793]);
        const expectedHs = ['0x0101010101010101010101010101010101010101010101010101010101010101','0x0202020202020202020202020202020202020202020202020202020202020202','0x0303030303030303030303030303030303030303030303030303030303030303'];
        assert.equal(receipt.logs[1].args.hs.toString().toLowerCase(), expectedHs);
        assert.equal(receipt.logs[1].args.ts.toString(), '123456');
        assert.equal(receipt.logs[1].args.tss.toString(), [12345678, 87654321]);
    });

    it('should decode msg2 (using soltype) with large numbers correctly', async () => {
        const buf = fs.readFileSync(path.join(__dirname, "../../msg2_large_number.pb"));
        const raw = '0x' + buf.toString('hex');

        const receipt = await testMain.testMsg2(raw);

        assert.equal(receipt.logs[1].args.ts.toString(), '18446744073709551615');
        assert.equal(receipt.logs[1].args.tss.toString(), '18446744073709551615,18446744073709551614,18446744073709551613');
    });

    it('should not decode msg2 with wrong addr len successfully', async () => {
        const buf = fs.readFileSync(path.join(__dirname, "../../msg2_wrong_addr.pb"));
        const raw = '0x' + buf.toString('hex');        

        let err = null;

        try {
            await testMain.testMsg2(raw);
        } catch (error) {
            err = error;
        }
        assert.isOk(err instanceof Error);
    });

    it('should not decode msg2 with wrong bytes32 len successfully', async () => {
        const buf = fs.readFileSync(path.join(__dirname, "../../msg2_wrong_bytes32.pb"));
        const raw = '0x' + buf.toString('hex');

        let err = null;

        try {
            await testMain.testMsg2(raw);
        } catch (error) {
            err = error;
        }
        assert.isOk(err instanceof Error);
    });

    it('should decode msg3 (embedded msgs) correctly', async () => {
        const buf = fs.readFileSync(path.join(__dirname, "../../msg3.pb"));
        const raw = '0x' + buf.toString('hex');

        const receipt = await testMain.testMsg3(raw);

        let expectedAddrs, expectedHs;

        // check events of m1
        assert.equal(receipt.logs[0].event, 'Msg1Part1');
        assert.equal(receipt.logs[0].args.f1.toString(), '1');
        assert.equal(receipt.logs[0].args.f2.toString(), '12');
        assert.equal(receipt.logs[0].args.f3.toString(), 'true');
        assert.equal(receipt.logs[0].args.f4.toString(), '0x01020304');
        assert.equal(receipt.logs[0].args.f5.toString(), 'abc');
        assert.equal(receipt.logs[0].args.f6.toString(), [1, 2, 3]);
        assert.equal(receipt.logs[0].args.f7.toString(), [11, 12, 13]);

        assert.equal(receipt.logs[1].event, 'Msg1Part2');
        assert.equal(receipt.logs[1].args.f8.toString(), [true, false, true]);
        assert.equal(receipt.logs[1].args.f9_0.toString(), '0x0102');
        assert.equal(receipt.logs[1].args.f9_1.toString(), '0x0304');
        assert.equal(receipt.logs[1].args.f10_0.toString(), 'ab');
        assert.equal(receipt.logs[1].args.f10_1.toString(), 'cd');

        // check events of m2
        assert.equal(receipt.logs[2].event, 'Msg2Part1');
        assert.equal(receipt.logs[2].args.num.toString(), '12');
        assert.equal(receipt.logs[2].args.addr.toString().toLowerCase(), '0x0002030405060708090a0b0c0d0e0f1011121314');
        assert.equal(receipt.logs[2].args.addrPayable.toString().toLowerCase(), '0x0102030405060708090a0b0c0d0e0f1011121314');
        assert.equal(receipt.logs[2].args.amt.toString(), '117967114');
        assert.equal(receipt.logs[2].args.h.toString().toLowerCase(), '0x0101010101010101010101010101010101010101010101010101010101010101');
        assert.equal(receipt.logs[2].args.nums.toString(), [12, 34, 56]);
        expectedAddrs1 = ['0x0002030405060708090a0b0c0d0e0f1011121314','0x0b0c0d0e0f10111213140102030405060708090a','0xff02030405060708090a0b0c0d0e0f1011121314'];
        expectedAddrs2 = ['0x0102030405060708090a0b0c0d0e0f1011121314','0x0b0c0d0e0f10111213140102030405060708090a','0xff02030405060708090a0b0c0d0e0f1011121314'];
        assert.equal(receipt.logs[2].args.addrs.toString().toLowerCase(), expectedAddrs1);
        assert.equal(receipt.logs[2].args.addrPayables.toString().toLowerCase(), expectedAddrs2);

        assert.equal(receipt.logs[3].event, 'Msg2Part2');
        assert.equal(receipt.logs[3].args.amts.toString(), [1, 257, 65793]);
        expectedHs = ['0x0101010101010101010101010101010101010101010101010101010101010101','0x0202020202020202020202020202020202020202020202020202020202020202','0x0303030303030303030303030303030303030303030303030303030303030303'];
        assert.equal(receipt.logs[3].args.hs.toString().toLowerCase(), expectedHs);
        assert.equal(receipt.logs[3].args.ts.toString(), '12345678');
        assert.equal(receipt.logs[3].args.tss.toString(), [12345678, 12345678]);

        // check events of m1s[0]
        assert.equal(receipt.logs[4].event, 'Msg1Part1');
        assert.equal(receipt.logs[4].args.f1.toString(), '2');
        assert.equal(receipt.logs[4].args.f2.toString(), '1234567');
        assert.equal(receipt.logs[4].args.f3.toString(), 'false');
        assert.equal(receipt.logs[4].args.f4.toString(), '0x01020304');
        assert.equal(receipt.logs[4].args.f5.toString(), 'abc');
        assert.equal(receipt.logs[4].args.f6.toString(), [1, 2, 3]);
        assert.equal(receipt.logs[4].args.f7.toString(), [11, 12, 13]);

        assert.equal(receipt.logs[5].event, 'Msg1Part2');
        assert.equal(receipt.logs[5].args.f8.toString(), [true, false, true]);
        assert.equal(receipt.logs[5].args.f9_0.toString(), '0x0102');
        assert.equal(receipt.logs[5].args.f9_1.toString(), '0x0304');
        assert.equal(receipt.logs[5].args.f10_0.toString(), 'ab');
        assert.equal(receipt.logs[5].args.f10_1.toString(), 'cd');

        // check events of m1s[1]
        assert.equal(receipt.logs[6].event, 'Msg1Part1');
        assert.equal(receipt.logs[6].args.f1.toString(), '3');
        assert.equal(receipt.logs[6].args.f2.toString(), '1234567');
        assert.equal(receipt.logs[6].args.f3.toString(), 'false');
        assert.equal(receipt.logs[6].args.f4.toString(), '0x01020304');
        assert.equal(receipt.logs[6].args.f5.toString(), 'abc');
        assert.equal(receipt.logs[6].args.f6.toString(), [1, 2, 3]);
        assert.equal(receipt.logs[6].args.f7.toString(), [11, 12, 13]);

        assert.equal(receipt.logs[7].event, 'Msg1Part2');
        assert.equal(receipt.logs[7].args.f8.toString(), [true, false, true]);
        assert.equal(receipt.logs[7].args.f9_0.toString(), '0x0102');
        assert.equal(receipt.logs[7].args.f9_1.toString(), '0x0304');
        assert.equal(receipt.logs[7].args.f10_0.toString(), 'ab');
        assert.equal(receipt.logs[7].args.f10_1.toString(), 'cd');

        // check events of m2s[0]
        assert.equal(receipt.logs[8].event, 'Msg2Part1');
        assert.equal(receipt.logs[8].args.num.toString(), '111');
        assert.equal(receipt.logs[8].args.addr.toString().toLowerCase(), '0x000c0d0e0f10111213140102030405060708090a');
        assert.equal(receipt.logs[8].args.addrPayable.toString().toLowerCase(), '0x0b0c0d0e0f10111213140102030405060708090a');
        assert.equal(receipt.logs[8].args.amt.toString(), '117967114');
        assert.equal(receipt.logs[8].args.h.toString().toLowerCase(), '0x0202020202020202020202020202020202020202020202020202020202020202');
        assert.equal(receipt.logs[8].args.nums.toString(), [12, 34, 56]);
        expectedAddrs1 = ['0x0002030405060708090a0b0c0d0e0f1011121314','0x0b0c0d0e0f10111213140102030405060708090a','0xff02030405060708090a0b0c0d0e0f1011121314'];
        expectedAddrs2 = ['0x0102030405060708090a0b0c0d0e0f1011121314','0x0b0c0d0e0f10111213140102030405060708090a','0xff02030405060708090a0b0c0d0e0f1011121314'];
        assert.equal(receipt.logs[8].args.addrs.toString().toLowerCase(), expectedAddrs1);
        assert.equal(receipt.logs[8].args.addrPayables.toString().toLowerCase(), expectedAddrs2);

        assert.equal(receipt.logs[9].event, 'Msg2Part2');
        assert.equal(receipt.logs[9].args.amts.toString(), [1, 257, 65793]);
        expectedHs = ['0x0101010101010101010101010101010101010101010101010101010101010101','0x0202020202020202020202020202020202020202020202020202020202020202','0x0303030303030303030303030303030303030303030303030303030303030303'];
        assert.equal(receipt.logs[9].args.hs.toString().toLowerCase(), expectedHs);
        assert.equal(receipt.logs[9].args.ts.toString(), '12345');
        assert.equal(receipt.logs[9].args.tss.toString(), [123456, 1234567]);

        // check events of m2s[1]
        assert.equal(receipt.logs[10].event, 'Msg2Part1');
        assert.equal(receipt.logs[10].args.num.toString(), '222');
        assert.equal(receipt.logs[10].args.addr.toString().toLowerCase(), '0x0002030405060708090a0b0c0d0e0f1011121314');
        assert.equal(receipt.logs[10].args.addrPayable.toString().toLowerCase(), '0xff02030405060708090a0b0c0d0e0f1011121314');
        assert.equal(receipt.logs[10].args.amt.toString(), '117967114');
        assert.equal(receipt.logs[10].args.h.toString().toLowerCase(), '0x0303030303030303030303030303030303030303030303030303030303030303');
        assert.equal(receipt.logs[10].args.nums.toString(), [12, 34, 56]);
        expectedAddrs1 = ['0x0002030405060708090a0b0c0d0e0f1011121314','0x0b0c0d0e0f10111213140102030405060708090a','0xff02030405060708090a0b0c0d0e0f1011121314'];
        expectedAddrs2 = ['0x0102030405060708090a0b0c0d0e0f1011121314','0x0b0c0d0e0f10111213140102030405060708090a','0xff02030405060708090a0b0c0d0e0f1011121314'];
        assert.equal(receipt.logs[10].args.addrs.toString().toLowerCase(), expectedAddrs1);
        assert.equal(receipt.logs[10].args.addrPayables.toString().toLowerCase(), expectedAddrs2);

        assert.equal(receipt.logs[11].event, 'Msg2Part2');
        assert.equal(receipt.logs[11].args.amts.toString(), [1, 257, 65793]);
        expectedHs = ['0x0101010101010101010101010101010101010101010101010101010101010101','0x0202020202020202020202020202020202020202020202020202020202020202','0x0303030303030303030303030303030303030303030303030303030303030303'];
        assert.equal(receipt.logs[11].args.hs.toString().toLowerCase(), expectedHs);
        assert.equal(receipt.logs[11].args.ts.toString(), '54321');
        assert.equal(receipt.logs[11].args.tss.toString(), [654321, 7654321]);
    });

    it('should decode msg4 (enum) correctly', async () => {
        const buf = fs.readFileSync(path.join(__dirname, "../../msg4.pb"));
        const raw = '0x' + buf.toString('hex');

        const receipt = await testMain.testMsg4(raw);

        assert.equal(receipt.logs[0].event, 'Msg4Info');
        assert.equal(receipt.logs[0].args.enum1.toString(), '1');
        assert.equal(receipt.logs[0].args.enum2.toString(), '2');
        assert.equal(receipt.logs[0].args.enums.toString(), [0, 1, 2]);
    });

    it('should decode import correctly', async () => {
        const buf = fs.readFileSync(path.join(__dirname, "../../b.pb"));
        const raw = '0x' + buf.toString('hex');

        const receipt = await testMain.testImport(raw);

        assert.equal(receipt.logs[0].event, 'DecodedB');
        assert.equal(receipt.logs[0].args.i.toString(), '1');
        assert.equal(receipt.logs[0].args.alistlen.toString(), '2');
        assert.equal(receipt.logs[0].args.e.toString(), '0');
        assert.equal(receipt.logs[0].args.elistlen.toString(), '2');
    });
});
