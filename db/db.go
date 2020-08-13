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
	"bytes"
	"container/list"
	"encoding/binary"
	"github.com/polynetwork/btc-vendor-tools/utils"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"sort"
	"sync"
)

const CACHE_SIZE = 100

var (
	tx_prefix       = []byte("tx")
	totalnum_prefix = []byte("total")
)

type VendorDB struct {
	lock    sync.RWMutex
	db      *leveldb.DB
	txCache *VendorCache
}

func NewVendorDB(path string) (*VendorDB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	v := &VendorDB{
		lock: sync.RWMutex{},
		db:   db,
		txCache: &VendorCache{
			l:    list.New(),
			size: CACHE_SIZE,
		},
	}
	v.txCache.totalCache = v.GetTotalTxNum()
	return v, nil
}

func (v *VendorDB) PutSignedTx(txHash []byte, item *utils.SavedItem) error {
	v.lock.Lock()
	defer v.lock.Unlock()

	if old := v.txCache.Get(txHash); old != nil {
		return nil
	}
	old, err := v.db.Get(append(tx_prefix, txHash...), nil)
	if old != nil {
		return nil
	}
	if err != leveldb.ErrNotFound {
		return nil
	}

	val, err := item.Serialize()
	if err != nil {
		return err
	}
	if err = v.db.Put(append(tx_prefix, txHash...), val, nil); err != nil {
		return err
	}
	v.txCache.Put(item)

	val = make([]byte, 8)
	binary.BigEndian.PutUint64(val, v.txCache.totalCache)
	if err = v.db.Put(totalnum_prefix, val, nil); err != nil {
		return err
	}

	return nil
}

func (v *VendorDB) GetSignedTx(txHash []byte) (*utils.SavedItem, error) {
	v.lock.RLock()
	defer v.lock.RUnlock()

	if res := v.txCache.Get(txHash); res != nil {
		return res, nil
	}
	val, err := v.db.Get(append(tx_prefix, txHash...), nil)
	if err != nil {
		return nil, err
	}
	item := &utils.SavedItem{}
	if err = item.Deserialize(val); err != nil {
		return nil, err
	}

	return item, nil
}

func (v *VendorDB) SetTxDone(txHash []byte) error {
	v.lock.Lock()
	defer v.lock.Unlock()

	if val := v.txCache.Get(txHash); val != nil {
		val.Done = true
	}
	val, err := v.db.Get(append(tx_prefix, txHash...), nil)
	if err != nil {
		return err
	}
	item := &utils.SavedItem{}
	if err = item.Deserialize(val); err != nil {
		return err
	}
	item.Done = true
	raw, _ := item.Serialize()
	if err = v.db.Put(append(tx_prefix, txHash...), raw, nil); err != nil {
		return err
	}
	return nil
}

func (v *VendorDB) GetTotalTxNum() uint64 {
	v.lock.RLock()
	defer v.lock.RUnlock()

	if v.txCache.totalCache > 0 {
		return v.txCache.totalCache
	}
	val, err := v.db.Get(totalnum_prefix, nil)
	if err != nil {
		return 0
	}
	res := binary.BigEndian.Uint64(val)
	return res
}

func (v *VendorDB) ReadFirstBatch() (utils.SavedItemArr, error) {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.txCache.Len() == CACHE_SIZE || uint64(v.txCache.Len()) == v.txCache.totalCache {
		return v.txCache.GetAll(), nil
	}

	arr := utils.SavedItemArr(make([]*utils.SavedItem, 0))
	iter := v.db.NewIterator(util.BytesPrefix(tx_prefix), nil)
	for iter.Next() {
		item := &utils.SavedItem{}
		if err := item.Deserialize(iter.Value()); err != nil {
			return nil, err
		}
		arr = append(arr, item)
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(arr))

	for i := v.txCache.Len(); i < CACHE_SIZE && i < len(arr); i++ {
		v.txCache.putToFront(arr[i])
	}
	if len(arr) > CACHE_SIZE {
		return arr[:CACHE_SIZE], nil
	}
	return arr, nil
}

func (v *VendorDB) Close() error {
	return v.db.Close()
}

type VendorCache struct {
	l          *list.List
	size       int
	totalCache uint64
}

func (cache *VendorCache) Put(item *utils.SavedItem) {
	cache.l.PushBack(item)
	if cache.l.Len() > cache.size {
		cache.l.Remove(cache.l.Front())
	}
	cache.totalCache++
}

func (cache *VendorCache) putToFront(item *utils.SavedItem) {
	cache.l.PushFront(item)
}

func (cache *VendorCache) Get(txHash []byte) *utils.SavedItem {
	for v := cache.l.Back(); v != nil; v = v.Prev() {
		vhash := v.Value.(*utils.SavedItem).Item.Mtx.TxHash()
		if bytes.Equal(vhash[:], txHash) {
			return v.Value.(*utils.SavedItem)
		}
	}
	return nil
}

func (cache *VendorCache) GetAll() utils.SavedItemArr {
	arr := make([]*utils.SavedItem, 0)
	for young := cache.l.Back(); young != nil; young = young.Prev() {
		arr = append(arr, young.Value.(*utils.SavedItem))
	}
	return arr
}

func (cache *VendorCache) Len() int {
	return cache.l.Len()
}
