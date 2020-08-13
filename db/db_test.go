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
package db

import (
	"container/list"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/polynetwork/vendortool/utils"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

var (
	getTxArr = func(n int) []*utils.SavedItem {
		arr := make([]*utils.SavedItem, n)
		for i := 0; i < n; i++ {
			tx := wire.NewMsgTx(wire.TxVersion)
			tx.AddTxIn(wire.NewTxIn(wire.NewOutPoint(&chainhash.Hash{}, uint32(i)), nil, nil))
			arr[i] = &utils.SavedItem{
				Item: &utils.ToSignItem{
					Mtx: tx,
				},
				TimeReceived: time.Now(),
			}
			time.Sleep(10 * time.Millisecond)
		}
		return arr
	}
)

func TestNewVendorDB(t *testing.T) {
	_, err := NewVendorDB("./temp")
	defer os.RemoveAll("./temp")
	assert.NoError(t, err)
}

func TestVendorCache_Put(t *testing.T) {
	c := VendorCache{
		l:    list.New(),
		size: 10,
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	item := &utils.SavedItem{
		Item: &utils.ToSignItem{
			Mtx: tx,
		},
		TimeReceived: time.Now(),
	}
	raw, err := item.Serialize()
	assert.NoError(t, err)
	c.Put(item)
	txid := tx.TxHash()
	v := c.Get(txid[:])
	raw1, err := v.Serialize()
	assert.NoError(t, err)

	assert.Equal(t, raw, raw1)
}

func TestVendorCache_GetAll(t *testing.T) {
	c := VendorCache{
		l:    list.New(),
		size: 10,
	}
	arr := getTxArr(11)
	for _, v := range arr {
		c.Put(v)
	}
	assert.Equal(t, 10, c.Len())
	res := c.GetAll()
	for i, v := range res {
		assert.Equal(t, true, v.TimeReceived.Equal(arr[10-i].TimeReceived))
	}
}

func TestVendorDB_PutSignedTx(t *testing.T) {
	db, _ := NewVendorDB("./temp")
	defer os.RemoveAll("./temp")

	arr := getTxArr(201)

	txid1 := arr[0].Item.Mtx.TxHash()
	err := db.PutSignedTx(txid1[:], arr[0])
	assert.NoError(t, err)
	err = db.PutSignedTx(txid1[:], arr[0])
	assert.NoError(t, err)

	for i := 1; i < 201; i++ {
		txid := arr[i].Item.Mtx.TxHash()
		err = db.PutSignedTx(txid[:], arr[i])
		assert.NoError(t, err)
	}
	total := db.GetTotalTxNum()
	assert.Equal(t, uint64(201), total)

	for i := 0; i < 201; i++ {
		txid := arr[i].Item.Mtx.TxHash()
		item, err := db.GetSignedTx(txid[:])
		assert.NoError(t, err, fmt.Sprintf("%d: %s", i, txid.String()))
		assert.Equal(t, true, item.TimeReceived.Equal(arr[i].TimeReceived))
	}
}

func TestVendorDB_ReadFirstBatch(t *testing.T) {
	db, _ := NewVendorDB("./temp")
	defer os.RemoveAll("./temp")

	arr := getTxArr(200)
	for i := 0; i < 200; i++ {
		txid := arr[i].Item.Mtx.TxHash()
		err := db.PutSignedTx(txid[:], arr[i])
		assert.NoError(t, err)
	}
	total := db.GetTotalTxNum()
	assert.Equal(t, uint64(200), total)

	res, err := db.ReadFirstBatch()
	assert.NoError(t, err)
	for i, v := range res {
		assert.Equal(t, arr[199-i].Item.Mtx.TxIn[0].PreviousOutPoint.String(), v.Item.Mtx.TxIn[0].PreviousOutPoint.String())
	}

	db.Close()
	db, err = NewVendorDB("./temp1")
	defer os.RemoveAll("./temp1")
	assert.NoError(t, err)

	for i := 0; i < 10; i++ {
		txid := arr[i].Item.Mtx.TxHash()
		err := db.PutSignedTx(txid[:], arr[i])
		assert.NoError(t, err)
	}
	total = db.GetTotalTxNum()
	assert.Equal(t, uint64(10), total)

	res, err = db.ReadFirstBatch()
	assert.NoError(t, err)

	for i, v := range res {
		assert.Equal(t, arr[9-i].Item.Mtx.TxIn[0].PreviousOutPoint.String(), v.Item.Mtx.TxIn[0].PreviousOutPoint.String())
	}
}

func TestVendorDB_SetTxDone(t *testing.T) {
	db, _ := NewVendorDB("./temp")
	defer os.RemoveAll("./temp")

	arr := getTxArr(1)
	txid := arr[0].Item.Mtx.TxHash()
	err := db.PutSignedTx(txid[:], arr[0])
	assert.NoError(t, err)

	err = db.SetTxDone(txid[:])
	assert.NoError(t, err)

	res, _ := db.GetSignedTx(txid[:])
	assert.Equal(t, true, res.Done)
}
