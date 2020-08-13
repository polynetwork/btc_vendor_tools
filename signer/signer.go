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
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/ontio/ontology-crypto/ec"
	sdk "github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/poly-go-sdk/client"
	"github.com/polynetwork/btc-vendor-tools/config"
	"github.com/polynetwork/btc-vendor-tools/db"
	"github.com/polynetwork/btc-vendor-tools/log"
	"github.com/polynetwork/btc-vendor-tools/utils"
	"time"
)

type Signer struct {
	txchan chan *utils.ToSignItem
	privk  *btcec.PrivateKey
	addr   *btcutil.AddressPubKey
	poly   *sdk.PolySdk
	acct   *sdk.Account
	redeem []byte
	vdb    *db.VendorDB
}

func NewSigner(privkFile string, pwd []byte, txchan chan *utils.ToSignItem, acct *sdk.Account, poly *sdk.PolySdk,
	redeem []byte, vdb *db.VendorDB) (*Signer, error) {
	btcAcct, err := utils.GetAccountByPassword(poly, privkFile, pwd)
	if err != nil {
		return nil, fmt.Errorf("[NewSigner] failed to get btc account: %v", err)
	}
	privkP256 := btcec.PrivateKey(*btcAcct.GetPrivateKey().(*ec.PrivateKey).PrivateKey)
	privk, _ := btcec.PrivKeyFromBytes(btcec.S256(), (&privkP256).Serialize())
	addr, err := btcutil.NewAddressPubKey(privk.PubKey().SerializeCompressed(), config.BtcNetParam)
	if err != nil {
		return nil, fmt.Errorf("[NewSigner] failed to new AddressPubKey: %v", err)
	}

	return &Signer{
		txchan: txchan,
		privk:  privk,
		addr:   addr,
		acct:   acct,
		poly:   poly,
		redeem: redeem,
		vdb:    vdb,
	}, nil
}

func (signer *Signer) Signing() {
	log.Infof("[Signer] start signing")
	key := utils.GetUtxoKey(signer.redeem)
	for {
		select {
		case item := <-signer.txchan:
			txHash := item.Mtx.TxHash()
			sigs, err := signer.getSigs(item)
			if err != nil {
				log.Errorf("[Signer] failed to sign (unsigned tx hash %s), not supposed to happen: "+
					"%v", txHash.String(), err)
				continue
			}
		RETRY:
			txid, err := signer.poly.Native.Ccm.BtcMultiSign(1, key, txHash[:], signer.addr.EncodeAddress(),
				sigs, signer.acct)
			if err != nil {
				switch err.(type) {
				case client.PostErr:
					log.Errorf("[Signer] post err and would retry after %d sec: %v", config.SleepTime, err)
					utils.Wait(config.SleepTime)
					goto RETRY
				default:
					log.Errorf("[Signer] account %s failed to invoke polygon: %v", signer.addr.EncodeAddress(), err)
				}
				continue
			}

			mtx := item.Mtx.Copy()
			for _, v := range mtx.TxIn {
				v.SignatureScript = nil
			}
			key := mtx.TxHash()
			if err = signer.vdb.PutSignedTx(key[:], &utils.SavedItem{
				Item:         item,
				TimeReceived: time.Now(),
				Done:         false,
			}); err != nil {
				log.Errorf("[Signer] failed to save item key:%s into db: %v", key.String(), err)
			}
			log.Infof("[Signer] signed for btc tx %s (db-key: %s) and send tx %s to polygon", txHash.String(),
				key.String(), txid.ToHexString())
		}
	}
}

func (signer *Signer) Sign(item *utils.ToSignItem) error {
	txHash := item.Mtx.TxHash()
	key := utils.GetUtxoKey(signer.redeem)
	sigs, err := signer.getSigs(item)
	if err != nil {
		log.Errorf("[Signer] failed to sign (unsigned tx hash %s), not supposed to happen: "+
			"%v", txHash.String(), err)
		return err
	}

RETRY:
	txid, err := signer.poly.Native.Ccm.BtcMultiSign(1, key, txHash[:], signer.addr.EncodeAddress(), sigs, signer.acct)
	if err != nil {
		switch err.(type) {
		case client.PostErr:
			log.Errorf("[Signer] post err and would retry after %d sec: %v", config.SleepTime, err)
			utils.Wait(config.SleepTime)
			goto RETRY
		default:
			log.Errorf("[Signer] failed to invoke polygon: %v", err)
			return err
		}
	}
	log.Infof("[Signer] signed for btc tx %s and send tx %s to polygon", txHash.String(), txid.ToHexString())
	return nil
}

func (signer *Signer) getSigs(item *utils.ToSignItem) ([][]byte, error) {
	sigs := make([][]byte, 0)
	pkScripts := make([][]byte, len(item.Mtx.TxIn))
	for i, in := range item.Mtx.TxIn {
		pkScripts[i] = in.SignatureScript
		item.Mtx.TxIn[i].SignatureScript = nil
	}
	var sig []byte
	var err error
	for i, pks := range pkScripts {
		switch c := txscript.GetScriptClass(pks); c {
		case txscript.MultiSigTy, txscript.ScriptHashTy:
			sig, err = txscript.RawTxInSignature(item.Mtx, i, signer.redeem, txscript.SigHashAll, signer.privk)
			if err != nil {
				return nil, fmt.Errorf("Failed to sign tx's No.%d input(%s): %v", i, c, err)
			}
		case txscript.WitnessV0ScriptHashTy:
			sh := txscript.NewTxSigHashes(item.Mtx)
			sig, err = txscript.RawTxInWitnessSignature(item.Mtx, sh, i, int64(item.Amts[i]), signer.redeem,
				txscript.SigHashAll, signer.privk)
			if err != nil {
				return nil, fmt.Errorf("Failed to sign tx's No.%d input(%s): %v", i, c, err)
			}
		default:
			return nil, fmt.Errorf("wrong type of input: %s", c)
		}
		sigs = append(sigs, sig)
	}

	return sigs, nil
}
