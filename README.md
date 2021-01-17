# ipfs-ds
Utility for working with the ipfs datastore

## Overview

Occasionally go-ipfs runs into issues where it might be useful for debugging or hotfixing purposes to directly
modify the datastore. `ipfs-ds` is a tool to help you do this.

## Installation

Clone this repo and run `go build` in the directory to build the binary,
or run `go install` to have the binary placed in your Go binary folder (e.g. `~/go/bin`).

## Examples

`ipfs-ds get --base=base58btc /local/filesroot` will output the bytes corresponding to the MFS root as a base58btc

`ipfs-ds put --value-encoded /local/filesroot zQmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn` will set the MFS root
to the bytes corresponding to the given base58btc data

Note: `QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn` is the UnixFSv1 empty directory. Therefore, running the above
command will reset your MFS root back to default in case it gets corrupted.

## A Note on Encodings

The reason for the optional base encodings is that it's possible that datastore keys or values might not be
representable in text. Therefore, we allow putting values into, and getting values out of, the CLI using multibase
encodings.
