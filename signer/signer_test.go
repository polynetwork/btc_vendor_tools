/*
* Copyright (C) 2020 The poly network Authors
* This file is part of The poly network library.
*
* The poly network is free software: you can redistribute it and/or modify
* it under the terms of the GNU Lesser General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* The poly network is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
* GNU Lesser General Public License for more details.
* You should have received a copy of the GNU Lesser General Public License
* along with The poly network . If not, see <http://www.gnu.org/licenses/>.
*/
package signer

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	sdk "github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/poly-go-sdk/client"
	"github.com/polynetwork/poly/common"
	"github.com/polynetwork/poly/native/service/governance/side_chain_manager"
	common2 "github.com/polynetwork/poly/native/service/header_sync/common"
	utils2 "github.com/polynetwork/poly/native/service/utils"
	"github.com/polynetwork/btc-vendor-tools/config"
	"github.com/polynetwork/btc-vendor-tools/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var (
	privk  = "../privk"
	redeem = "552102dec9a415b6384ec0a9331d0cdf02020f0f1e5731c327b86e2b5a92455a289748210365b1066bcfa21987c3e207b92e309b95ca6bee5f1133cf04d6ed4ed265eafdbc21031104e387cd1a103c27fdc8a52d5c68dec25ddfb2f574fbdca405edfd8c5187de21031fdb4b44a9f20883aff505009ebc18702774c105cb04b1eecebcb294d404b1cb210387cda955196cc2b2fc0adbbbac1776f8de77b563c6d2a06a77d96457dc3d0d1f2102dd7767b6a7cc83693343ba721e0f5f4c7b4b8d85eeb7aec20d227625ec0f59d321034ad129efdab75061e8d4def08f5911495af2dae6d3e9a4b6e7aeb5186fa432fc57ae"
	swTx   = "01000000000101d102bf46072d5c36819d633e3e7685aa12ea870eeaa5ec1cce8165d324381b340100000000ffffffff02021b0000000000001976a91428d2e8cee08857f569e5a1b147c5d5e87339e08188ac2911000000000000220020216a09cb8ee51da1a91ea8942552d7936c886a10b507299003661816c0e9f18b0700473044022005ef849688c8f3612995f4b3eee91f06f0cd19d8c494c9518436cc5e74bf49de022036a2b2dd0101c9828e825f333c8b0f4a137455612b39e199846fb1f74dc231a401483045022100d634681163b3ac17fefa345298c995bf734ad5332dea43e262eb0b1f4a6a49c10220065283735f52f7c0d6b41f9f9f60c0ec0dfa07b3499607b0dee7b1501313eab90147304402206c3753c1e36860dc77d11a7b1ae6a54307fe306b6c6f69daaf150931d43c404d022060490dad039d1429e4dac03c96f0144f09fe90cafce448892afcca81e9aa4334014730440220281324bab36282a1b8a134f1ecff18f54386044b8eee199696fa33ff1022724e0220277d80e6bf9544d98036a5748cd034e51be4a936359c79db298d8cffb70a725101483045022100bb6bd929b3a2378fd79b6f16ed9f0314625e28eafc974718484490f1f4fc92e202200fe5b4f58a0a80d0c40ed69ed35e8452f7c2e0298f0b1143291e914f5cc934a601f1552102dec9a415b6384ec0a9331d0cdf02020f0f1e5731c327b86e2b5a92455a289748210365b1066bcfa21987c3e207b92e309b95ca6bee5f1133cf04d6ed4ed265eafdbc21031104e387cd1a103c27fdc8a52d5c68dec25ddfb2f574fbdca405edfd8c5187de21031fdb4b44a9f20883aff505009ebc18702774c105cb04b1eecebcb294d404b1cb210387cda955196cc2b2fc0adbbbac1776f8de77b563c6d2a06a77d96457dc3d0d1f2102dd7767b6a7cc83693343ba721e0f5f4c7b4b8d85eeb7aec20d227625ec0f59d321034ad129efdab75061e8d4def08f5911495af2dae6d3e9a4b6e7aeb5186fa432fc57ae00000000"
	amts   = []uint64{11651}
)

func TestNewSigner(t *testing.T) {
	config.BtcNetParam = &chaincfg.RegressionNetParams
	txchan := make(chan *utils.ToSignItem, 10)
	poly := sdk.NewPolySdk()
	poly.NewRpcClient().SetAddress(startMockPolyServer())
	acct, err := utils.GetAccountByPassword(poly, "../wallet.dat", []byte("1"))
	assert.NoError(t, err)

	rb, _ := hex.DecodeString(redeem)
	_, err = NewSigner(privk, "123", txchan, acct, poly, rb)
	assert.NoError(t, err)
}

func TestSigner_Signing(t *testing.T) {
	config.BtcNetParam = &chaincfg.RegressionNetParams
	txchan := make(chan *utils.ToSignItem, 10)
	poly := sdk.NewPolySdk()
	poly.NewRpcClient().SetAddress(startMockPolyServer())
	acct, err := utils.GetAccountByPassword(poly, "../wallet.dat", []byte("1"))
	assert.NoError(t, err)

	rb, _ := hex.DecodeString(redeem)
	signer, err := NewSigner(privk, "123", txchan, acct, poly, rb)
	assert.NoError(t, err)

	go signer.Signing()

	mtx := wire.NewMsgTx(wire.TxVersion)
	buf, _ := hex.DecodeString(swTx)
	mtx.BtcDecode(bytes.NewBuffer(buf), wire.ProtocolVersion, wire.LatestEncoding)
	mtx.TxIn[0].Witness = nil
	lock, _ := hex.DecodeString("0020216a09cb8ee51da1a91ea8942552d7936c886a10b507299003661816c0e9f18b")
	mtx.TxIn[0].SignatureScript = lock

	txchan <- &utils.ToSignItem{
		Mtx:  mtx,
		Amts: amts,
	}

	time.Sleep(2 * time.Second)
}

func TestSigner_getSigs(t *testing.T) {
	config.BtcNetParam = &chaincfg.RegressionNetParams
	txchan := make(chan *utils.ToSignItem, 10)
	rb, _ := hex.DecodeString(redeem)
	signer, err := NewSigner(privk, "123", txchan, nil, nil, rb)
	if err != nil {
		t.Fatal(err)
	}

	mtx := wire.NewMsgTx(wire.TxVersion)
	buf, _ := hex.DecodeString(swTx)
	mtx.BtcDecode(bytes.NewBuffer(buf), wire.ProtocolVersion, wire.LatestEncoding)
	witData := mtx.TxIn[0].Witness
	mtx.TxIn[0].Witness = nil

	lock, _ := hex.DecodeString("0020216a09cb8ee51da1a91ea8942552d7936c886a10b507299003661816c0e9f18b")
	mtx.TxIn[0].SignatureScript = lock
	sigs, err := signer.getSigs(&utils.ToSignItem{
		Amts: amts,
		Mtx:  mtx,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(sigs) != 1 || !bytes.Equal(sigs[0], witData[1]) {
		t.Fatal("not equal")
	}
}

func startMockPolyServer() string {
	ms := httptest.NewServer(http.HandlerFunc(handlePolyReq))
	return ms.URL
}

func handlePolyReq(w http.ResponseWriter, r *http.Request) {
	rb, _ := ioutil.ReadAll(r.Body)
	req := new(client.JsonRpcRequest)
	_ = json.Unmarshal(rb, req)

	switch req.Method {
	case client.RPC_GET_STORAGE:
		if req.Params[1].(string) ==
			hex.EncodeToString(append([]byte(side_chain_manager.SIDE_CHAIN), utils2.GetUint64Bytes(1)...)) {
			sc := &side_chain_manager.SideChain{
				Router:       0,
				BlocksToWait: 1,
				ChainId:      1,
				Name:         "BTC",
			}
			sink := common.NewZeroCopySink(nil)
			_ = sc.Serialization(sink)
			resp := map[string]interface{}{
				"error":  int64(0),
				"desc":   "SUCCESS",
				"result": common.ToHexString(sink.Bytes()),
			}
			rb, _ := json.Marshal(map[string]interface{}{
				"jsonrpc": "2.0",
				"error":   resp["error"],
				"desc":    resp["desc"],
				"result":  resp["result"],
				"id":      req.Id,
			})

			w.Write(rb)
		} else if req.Params[1].(string) ==
			hex.EncodeToString(append([]byte(common2.CURRENT_HEADER_HEIGHT), utils2.GetUint64Bytes(1)...)) {
			rawBh, _ := hex.DecodeString("0000002050c2f32c30615106cc58b01352a13e6f309d7e6f142ccbe58d37a709f81a3f4739825ad49375ac5ff5fc292df9ed518124035f4edcf9b48d0aaf49b29ef7770ef410415effff7f2000000000")
			bh := new(wire.BlockHeader)
			_ = bh.BtcDecode(bytes.NewBuffer(rawBh), wire.ProtocolVersion, wire.LatestEncoding)

			sh := &MockStoredHeader{}
			sh.Header = *bh
			sh.Height = 1804
			sh.totalWork = big.NewInt(0)

			sink := new(common.ZeroCopySink)
			sh.Serialization(sink)

			resp := map[string]interface{}{
				"error":  int64(0),
				"desc":   "SUCCESS",
				"result": common.ToHexString(sink.Bytes()),
			}
			rb, _ := json.Marshal(map[string]interface{}{
				"jsonrpc": "2.0",
				"error":   resp["error"],
				"desc":    resp["desc"],
				"result":  resp["result"],
				"id":      req.Id,
			})
			w.Write(rb)
		}
	case client.RPC_SEND_TRANSACTION:
		resp := map[string]interface{}{
			"error":  int64(0),
			"desc":   "SUCCESS",
			"result": "ea9822ea747b14af52e2eb7986d8e145960f0bfb2c0df1ce00d98fd5061e5dbc",
		}
		rb, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"error":   resp["error"],
			"desc":    resp["desc"],
			"result":  resp["result"],
			"id":      req.Id,
		})

		w.Write(rb)
	default:
		fmt.Fprint(w, "wrong method")
	}
}

type MockStoredHeader struct {
	Header    wire.BlockHeader
	Height    uint32
	totalWork *big.Int
}

func (this *MockStoredHeader) Serialization(sink *common.ZeroCopySink) {
	buf := bytes.NewBuffer(nil)
	this.Header.Serialize(buf)
	sink.WriteVarBytes(buf.Bytes())
	sink.WriteUint32(this.Height)
	biBytes := this.totalWork.Bytes()
	pad := make([]byte, 32-len(biBytes))
	//serializedBI := append(pad, biBytes...)
	sink.WriteVarBytes(append(pad, biBytes...))
}

func TestSigner_Sign(t *testing.T) {
	redeem := "552102dec9a415b6384ec0a9331d0cdf02020f0f1e5731c327b86e2b5a92455a289748210365b1066bcfa21987c3e207b92e309b95ca6bee5f1133cf04d6ed4ed265eafdbc21031104e387cd1a103c27fdc8a52d5c68dec25ddfb2f574fbdca405edfd8c5187de21031fdb4b44a9f20883aff505009ebc18702774c105cb04b1eecebcb294d404b1cb210387cda955196cc2b2fc0adbbbac1776f8de77b563c6d2a06a77d96457dc3d0d1f2102dd7767b6a7cc83693343ba721e0f5f4c7b4b8d85eeb7aec20d227625ec0f59d321034ad129efdab75061e8d4def08f5911495af2dae6d3e9a4b6e7aeb5186fa432fc57ae"
	rb, _ := hex.DecodeString(redeem)
	//hasher := sha256.New()
	//hasher.Write(rb)

	//addr, err := btcutil.NewAddressWitnessScriptHash(hasher.Sum(nil), &chaincfg.TestNet3Params)
	//if err != nil {
	//	t.Fatal(err)
	//}

	//fmt.Println(addr.EncodeAddress())
	fmt.Println(utils.GetUtxoKey(rb))
}
