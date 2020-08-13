<h1 align="center">BTC Vendor Tool</h1>

## Introduction

Vendor Tool can sign a bitcoin cross-chain backing transaction created by Poly smartcontract. This transaction will unlock BTC from vendor's multisig address to user's address. Vendor Tool scans Poly chain and sign every unlock transaction created by Poly automaticly.

## Build From Source

### Prerequisites

- [Golang](https://golang.org/doc/install) version 1.14 or later

### Build

```shell
git clone https://github.com/polynetwork/btc-vendor-tools.git
cd btc-vendor-tools
go build -o vendortool cmd/run.go
```

After building the source code successfully,  you should see the executable program `vendortool`. 

## Usage

### Configuration

Before running vendortool, you have to config it right.

```
{
	"PolyJsonRpcAddress": "http://poly_rpc:20336", // Poly RPC address
	"WalletFile": "/path/to/wallet.dat", // poly wallet file
	"WalletPwd": "", // poly wallet password. if not set, you're supposed to input it starting vendortool
	"BtcWalletPwd": "", // password for bitcoin wallet file encrypted from btc private key
	"PolyObLoopWaitTime": 2, // interval for scanning poly
	"BtcPrivkFile": "/path/to/btcprivk", // bitcoin wallet file encrypted from btc private key
	"WatchingKeyToSign": "makeBtcTx",// key word no need to change
	"ConfigBitcoinNet": "test", // bitcoin net type
	"ConfigDBPath": "./db", // DB path
	"RestPort": 50071, // restful service port
	"SleepTime": 10, // sleep when some situation happen
	"CircleToSaveHeight": 300, // save a snapshot height every n heights.
	"Redeem": "552102dec...432fc57ae", // vendor multisig redeem script
	"SignerAddr": "",
	"ObServerAddr": "",
	"PolyStartHeight": 1, // start scanning from this height
	"WebServerPort": "8080" // web service for create a vendor (still in dev)
}
```

### Start Relayer

Run as follow:

```
./vendortool --web=0 --config=./conf.json
```

You can create a vendor by run:

```
./vendortool --config=./conf.json
```

And visit http://localhost:8080 to create a vendor. This function is still in develop.

