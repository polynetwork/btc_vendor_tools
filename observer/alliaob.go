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
package observer

import (
	"bytes"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/wire"
	sdk "github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/poly-go-sdk/client"
	"github.com/polynetwork/poly-go-sdk/common"
	"github.com/polynetwork/btc-vendor-tools/config"
	"github.com/polynetwork/btc-vendor-tools/db"
	"github.com/polynetwork/btc-vendor-tools/log"
	httpcom "github.com/polynetwork/btc-vendor-tools/rest/http/common"
	"github.com/polynetwork/btc-vendor-tools/utils"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

type Observer struct {
	txchan            chan *utils.ToSignItem
	poly              *sdk.PolySdk
	loopWaitTime      int64
	WatchingKeyToSign string
	hashKey           string
	dbPath            string
	waitingCircle     uint32
	obCli             *ObCli
	startHeight       uint32
	vdb               *db.VendorDB
}

func NewObserver(poly *sdk.PolySdk, txchan chan *utils.ToSignItem, loopWaitTime int64, redeem []byte, watchingKeyToSign,
	dbPath, signerAddr string, circle, startHeight uint32, vdb *db.VendorDB) *Observer {
	return &Observer{
		poly:              poly,
		txchan:            txchan,
		WatchingKeyToSign: watchingKeyToSign,
		hashKey:           utils.GetUtxoKey(redeem),
		loopWaitTime:      loopWaitTime,
		dbPath:            dbPath,
		waitingCircle:     circle,
		obCli: func(txchan chan *utils.ToSignItem) *ObCli {
			if txchan == nil {
				return NewObCli(signerAddr)
			} else {
				return nil
			}
		}(txchan),
		startHeight: startHeight,
		vdb:         vdb,
	}
}

func (ob *Observer) Listen() {
	log.Infof("starting observing with hash-key %s", ob.hashKey)

	top := ob.getLastHeight()
	if ob.startHeight != 0 {
		top = ob.startHeight
	}
	log.Infof("[Observer] get start height %d from checkpoint or db, check once %d seconds", top, ob.loopWaitTime)
	tick := time.NewTicker(time.Second * time.Duration(ob.loopWaitTime))
	defer tick.Stop()

	lastRecorded := top
	for {
		select {
		case <-tick.C:
			toSign := 0
			newTop, err := ob.poly.GetCurrentBlockHeight()
			if err != nil {
				log.Errorf("[Observer] failed to get current height, retry after 10 sec: %v", err)
				utils.Wait(config.SleepTime)
				continue
			}

			if newTop-top <= 0 {
				continue
			}
			h := top + 1
			log.Tracef("[Observer] watch from %d to %d", h, newTop)
			for h <= newTop {
				events, err := ob.poly.GetSmartContractEventByBlock(h)
				if err != nil {
					switch err.(type) {
					case client.PostErr:
						log.Errorf("[Observer] GetSmartContractEventByBlock failed, retry after 10 sec: %v", err)
					default:
						log.Errorf("[Observer] not supposed to happen: %v", err)
					}
					utils.Wait(config.SleepTime)
					continue
				}
				signCnt := ob.checkEvents(events, h)
				toSign += signCnt
				h++
			}
			if toSign > 0 {
				log.Infof("[Observer] btc tx to sig: total %d transactions captured this time", toSign)
			}
			top = newTop
			if toSign > 0 || top-lastRecorded >= ob.waitingCircle {
				if err := ob.setLastHeight(top); err != nil {
					log.Errorf("[Observer] failed to set height: %v", err)
				}
				lastRecorded = top
			}
		}
	}
}

func (ob *Observer) checkEvents(events []*common.SmartContactEvent, h uint32) int {
	toSign := 0
	for _, e := range events {
		for _, n := range e.Notify {
			states, ok := n.States.([]interface{})
			if !ok {
				continue
			}

			mtx := wire.NewMsgTx(wire.TxVersion)
			if name, ok := states[0].(string); ok && name == utils.TO_SIGN_TX_KEY && states[1].(string) == ob.hashKey {
				txb, err := hex.DecodeString(states[2].(string))
				if err != nil {
					log.Errorf("[Observer] wrong hex-string of tx from chain, not supposed to happen: %v", err)
					continue
				}
				if err = mtx.BtcDecode(bytes.NewBuffer(txb), wire.ProtocolVersion, wire.LatestEncoding); err != nil {
					log.Errorf("[Observer] failed to decode btc transaction from chain, not supposed to happen: "+
						"%v", err)
					continue
				}
				amts := make([]uint64, 0)
				for _, v := range states[3].([]interface{}) {
					amts = append(amts, uint64(v.(float64)))
				}
				item := &utils.ToSignItem{
					Mtx:  mtx,
					Amts: amts,
				}
				if ob.txchan == nil {
				RETRY:
					if err := ob.obCli.SendToSign(item); err != nil {
						log.Errorf("[Observer] failed to call rpc: %v", err)
						utils.Wait(config.SleepTime)
						goto RETRY
					}
				} else {
					ob.txchan <- item
				}
				toSign++
				mtx = mtx.Copy()
				for _, v := range mtx.TxIn {
					v.SignatureScript = nil
				}
				txid := mtx.TxHash()
				log.Infof("[Observer] captured one tx (unsigned txid: %s) when height is %d", txid.String(), h)
			} else if ok && name == utils.SIGNED_TX_KEY && states[5].(string) == ob.hashKey {
				txb, err := hex.DecodeString(states[3].(string))
				if err != nil {
					log.Errorf("[Observer] wrong hex-string of tx from chain, not supposed to happen: %v", err)
					continue
				}
				if err = mtx.BtcDecode(bytes.NewBuffer(txb), wire.ProtocolVersion, wire.LatestEncoding); err != nil {
					log.Errorf("[Observer] failed to decode btc transaction from chain, not supposed to happen: "+
						"%v", err)
					continue
				}
				for _, in := range mtx.TxIn {
					in.SignatureScript = nil
				}
				txid := mtx.TxHash()
				if err = ob.vdb.SetTxDone(txid[:]); err != nil {
					log.Errorf("[Observer] failed to change tx %s status: %v", txid.String(), err)
					continue
				}
				log.Infof("[Observer] tx (unsigned tx key: %s) is signed", txid.String())
			}
		}
	}

	return toSign
}

func (ob *Observer) getLastHeight() uint32 {
	val, err := ioutil.ReadFile(path.Join(ob.dbPath, "last_height"))
	if err != nil {
		return 0
	}

	h := string(val)
	h = strings.TrimFunc(h, func(r rune) bool {
		if r == ' ' || r == '\n' {
			return true
		}
		return false
	})

	height, err := strconv.ParseUint(h, 10, 32)
	if err != nil {
		return 0
	}

	return uint32(height)
}

func (ob *Observer) setLastHeight(h uint32) error {
	if err := ioutil.WriteFile(path.Join(ob.dbPath, "last_height"), []byte(strconv.FormatUint(uint64(h), 10)),
		0644); err != nil {
		return err
	}
	return nil
}

type ObCli struct {
	addr string
	cli  *http.Client
}

func NewObCli(addr string) *ObCli {
	return &ObCli{
		cli: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   5,
				DisableKeepAlives:     false,
				IdleConnTimeout:       time.Second * 300,
				ResponseHeaderTimeout: time.Second * 300,
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: time.Second * 300,
		},
		addr: addr,
	}
}

func (cli *ObCli) sendRequest(addr string, data []byte) ([]byte, error) {
	resp, err := cli.cli.Post(addr, "application/json;charset=UTF-8",
		bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("rest post request:%s error:%s", data, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read rest response body error:%s", err)
	}
	return body, nil
}

func (cli *ObCli) SendToSign(item *utils.ToSignItem) error {
	rawItem, err := item.Serialize()
	if err != nil {
		return err
	}
	req, err := json.Marshal(httpcom.SignItemReq{
		Raw: hex.EncodeToString(rawItem),
	})
	if err != nil {
		return err
	}

	data, err := cli.sendRequest("http://"+cli.addr+"/api/v1/signtx", req)
	if err != nil {
		return err
	}

	var resp httpcom.Response
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal resp to json: %v", err)
	}
	if resp.Error != 0 || resp.Desc != "SUCCESS" {
		return fmt.Errorf("response shows failure: %s", resp.Desc)
	}

	return nil
}
