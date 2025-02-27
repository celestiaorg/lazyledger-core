# fuzz

Fuzzing for various packages in CometBFT using [go-fuzz](https://github.com/dvyukov/go-fuzz) library.

Inputs:

- mempool `CheckTx` (using kvstore in-process ABCI app)
- p2p `Addrbook#AddAddress`
- p2p `pex.Reactor#Receive`
- p2p `SecretConnection#Read` and `SecretConnection#Write`
- rpc jsonrpc server

## Directory structure

```
| test
|  |- corpus/
|  |- crashers/
|  |- init-corpus/
|  |- suppressions/
|  |- testdata/
|  |- <testname>.go
```

`/corpus` directory contains corpus data. The idea is to help the fuzzer to
understand what bytes sequences are semantically valid (e.g. if we're testing
PNG decoder, then we would put black-white PNG into corpus directory; with
blockchain reactor - we would put blockchain messages into corpus).

`/init-corpus` (if present) contains a script for generating corpus data.

`/testdata` directory may contain an additional data (like `addrbook.json`).

Upon running the fuzzer, `/crashers` and `/suppressions` dirs will be created,
along with <testname>.zip archive. `/crashers` will show any inputs, which have
lead to panics (plus a trace). `/suppressions` will show any suppressed inputs.

## Running

```sh
make fuzz-mempool
make fuzz-p2p-addrbook
make fuzz-p2p-pex
make fuzz-p2p-sc
make fuzz-rpc-server
```

Each command will create corpus data (if needed), generate a fuzz archive and
call `go-fuzz` executable.

Then watch out for the respective outputs in the fuzzer output to announce new
crashers which can be found in the directory `crashers`.

For example if we find

```sh
ls crashers/
61bde465f47c93254d64d643c3b2480e0a54666e
61bde465f47c93254d64d643c3b2480e0a54666e.output
61bde465f47c93254d64d643c3b2480e0a54666e.quoted
da39a3ee5e6b4b0d3255bfef95601890afd80709
da39a3ee5e6b4b0d3255bfef95601890afd80709.output
da39a3ee5e6b4b0d3255bfef95601890afd80709.quoted
```

the crashing bytes generated by the fuzzer will be in
`61bde465f47c93254d64d643c3b2480e0a54666e` the respective crash report in
`61bde465f47c93254d64d643c3b2480e0a54666e.output`

and the bug report can be created by retrieving the bytes in
`61bde465f47c93254d64d643c3b2480e0a54666e` and feeding those back into the
`Fuzz` function.
