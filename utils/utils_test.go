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
package utils

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestToSignItem_Serialize(t *testing.T) {
	mtx := wire.NewMsgTx(wire.TxVersion)
	mtx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(&chainhash.Hash{}, 10), nil, nil))
	v := &ToSignItem{
		Mtx:  mtx,
		Amts: []uint64{100},
	}
	raw, err := v.Serialize()
	assert.NoError(t, err)

	v1 := &ToSignItem{}
	err = v1.Deserialize(raw)
	assert.NoError(t, err)

	fmt.Println(v1.Mtx.TxIn[0].PreviousOutPoint.String())
}

func TestSavedItem_Serialize(t *testing.T) {
	mtx := wire.NewMsgTx(wire.TxVersion)
	mtx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(&chainhash.Hash{}, 10), nil, nil))
	v := &ToSignItem{
		Mtx:  mtx,
		Amts: []uint64{100},
	}

	s := &SavedItem{
		TimeReceived: time.Now(),
		Item:         v,
		Done:         false,
	}

	raw, err := s.Serialize()
	assert.NoError(t, err)

	s1 := &SavedItem{}
	err = s1.Deserialize(raw)
	assert.NoError(t, err)
	fmt.Println(s1.TimeReceived.String(), s1.Item.Mtx.TxIn[0].PreviousOutPoint.String())
}

func TestGetAccountByPassword(t *testing.T) {
	tx := "0100000001ce8c9ed816254a123be3fbca5a58436583116a32a1cbe11db0de68bdb4da491200000000fd5f02004730440220463bb76f43e867af12437173ce1f17187bda9e2e871dd9063bcd02c800419a28022079cfea2d4f26f4c93fb7592781e1c3f4996bb3509beebf757bbbbb9006103b8501483045022100b0922a8f61fedca065b8ca4985862cf9f92b271722c2902442f82394a7f36ddf0220262f8bd70d8f757fbcc7e447e5f1e892dfabe77e03b11eec57d6b8b0a5c0f7e70147304402200663f1745c3366cce2f7311f6ccf78ab334c94f9add7aab11454a0f19d679d7e022040a551059f353f139a8d928e0b1160f81a7316e97c88c8cbe2a806aaf1239182014830450221009b25652451fc0ec4beb4c7db3cc1e2e085fe2e28074faa9d6131710d8db6234f02206e78909eb2a937932ca3abfab8f654574421d442a63ddb876e36a5bc3d8604b601483045022100c9b1738b666e099e843adbe1922751f2a1210bd2a542adcf92760f4f424a73c802204331b6ac5233fab40bda41ed6d5a7528bf9594926ca1273cc837cb49569701c2014cf1552102dec9a415b6384ec0a9331d0cdf02020f0f1e5731c327b86e2b5a92455a289748210365b1066bcfa21987c3e207b92e309b95ca6bee5f1133cf04d6ed4ed265eafdbc21031104e387cd1a103c27fdc8a52d5c68dec25ddfb2f574fbdca405edfd8c5187de21031fdb4b44a9f20883aff505009ebc18702774c105cb04b1eecebcb294d404b1cb210387cda955196cc2b2fc0adbbbac1776f8de77b563c6d2a06a77d96457dc3d0d1f2102dd7767b6a7cc83693343ba721e0f5f4c7b4b8d85eeb7aec20d227625ec0f59d321034ad129efdab75061e8d4def08f5911495af2dae6d3e9a4b6e7aeb5186fa432fc57aeffffffff02ff120100000000001976a9145f35a2cc0318fbc17c4c479964734e7a9f8819d788aca04b000000000000220020216a09cb8ee51da1a91ea8942552d7936c886a10b507299003661816c0e9f18b00000000"
	raw, _ := hex.DecodeString(tx)
	mtx := wire.NewMsgTx(wire.TxVersion)

	_ = mtx.BtcDecode(bytes.NewBuffer(raw), wire.ProtocolVersion, wire.LatestEncoding)

	for _, v := range mtx.TxIn {
		v.SignatureScript = nil
	}

	fmt.Println(mtx.TxHash().String())
}
