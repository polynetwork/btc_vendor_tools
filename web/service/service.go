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
package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/ontio/ontology-crypto/ec"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology-crypto/signature"
	"github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/poly/account"
	"github.com/polynetwork/poly/common"
	"github.com/polynetwork/poly/core/types"
	"github.com/polynetwork/poly/native/service/utils"
	"github.com/polynetwork/btc-vendor-tools/config"
	"github.com/polynetwork/btc-vendor-tools/db"
	utils2 "github.com/polynetwork/btc-vendor-tools/utils"
	"strconv"
	"strings"
)

type BtcService struct {
	Poly *poly_go_sdk.PolySdk
	Acc  *poly_go_sdk.Account
}

func NewBtcService(poly *poly_go_sdk.PolySdk) *BtcService {
	return &BtcService{
		Poly: poly,
	}
}

func (s *BtcService) GenePrivkAndAddr(pwd, path string) (*btcutil.WIF, *btcutil.AddressPubKey, error) {
	priv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, err
	}
	wif, err := btcutil.NewWIF(priv, config.BtcNetParam, true)
	if err != nil {
		return nil, nil, err
	}
	addr, err := btcutil.NewAddressPubKey(wif.PrivKey.PubKey().SerializeCompressed(), config.BtcNetParam)
	if err != nil {
		return nil, nil, err
	}

	pri, err := keypair.GetP256KeyPairFromWIF([]byte(wif.String()))
	if err != nil {
		return nil, nil, err
	}

	wallet, err := account.Open(path)
	if err != nil {
		return nil, nil, err
	}

	pub := pri.Public()
	a := types.AddressFromPubKey(pub)
	b58addr := a.ToBase58()
	k, err := keypair.EncryptPrivateKey(pri, b58addr, []byte(pwd))
	if err != nil {
		return nil, nil, err
	}

	var accMeta account.AccountMetadata
	accMeta.Address = k.Address
	accMeta.KeyType = k.Alg
	accMeta.EncAlg = k.EncAlg
	accMeta.Hash = k.Hash
	accMeta.Key = k.Key
	accMeta.Curve = k.Param["curve"]
	accMeta.Salt = k.Salt
	accMeta.Label = ""
	accMeta.PubKey = hex.EncodeToString(keypair.SerializePublicKey(pub))
	accMeta.SigSch = signature.SHA256withECDSA.Name()

	err = wallet.ImportAccount(&accMeta)
	if err != nil {
		return nil, nil, err
	}
	pwd = ""

	return wif, addr, nil
}

func (s *BtcService) GeneRedeem(pubkStr, reqStr string) ([]byte, []byte, *btcutil.AddressScriptHash,
	*btcutil.AddressWitnessScriptHash, error) {
	rn, err := strconv.ParseUint(reqStr, 10, 64)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	pubkStr = strings.TrimSpace(pubkStr)
	pubkStrArr := strings.Split(pubkStr, ",")
	if uint64(len(pubkStrArr)) < rn {
		return nil, nil, nil, nil, fmt.Errorf("require number %d is bigger than length of pubkeys %d", rn, len(pubkStrArr))
	}
	arr := make([]*btcutil.AddressPubKey, len(pubkStrArr))
	for i, s := range pubkStrArr {
		raw, err := hex.DecodeString(s)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("no.%d pubkey decode to hex error: %v", i, err)
		}
		p, err := btcutil.NewAddressPubKey(raw, config.BtcNetParam)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("no.%d new an AddressPubKey object error: %v", i, err)
		}
		arr[i] = p
	}
	r, err := txscript.MultiSigScript(arr, int(rn))
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to get a multisig-script: %v", err)
	}
	rk := btcutil.Hash160(r)
	p2sh, err := btcutil.NewAddressScriptHash(r, config.BtcNetParam)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to new an AddressScriptHash object error: %v", err)
	}
	hasher := sha256.New()
	hasher.Write(r)
	p2wsh, err := btcutil.NewAddressWitnessScriptHash(hasher.Sum(nil), config.BtcNetParam)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to new AddressWitnessScriptHash object error: %v", err)
	}
	return r, rk, p2sh, p2wsh, nil
}

func (s *BtcService) SignContract(pf, pwd, contract, redeem string, chainId, ver uint64) ([]byte, error) {
	btcAcct, err := utils2.GetAccountByPassword(s.Poly, pf, []byte(pwd))
	if err != nil {
		return nil, fmt.Errorf("[NewSigner] failed to get btc account: %v", err)
	}
	privkP256 := btcec.PrivateKey(*btcAcct.GetPrivateKey().(*ec.PrivateKey).PrivateKey)
	privk, _ := btcec.PrivKeyFromBytes(btcec.S256(), (&privkP256).Serialize())

	contract = strings.Replace(contract, "0x", "", 1)
	c, err := hex.DecodeString(contract)
	if err != nil {
		return nil, err
	}
	r, err := hex.DecodeString(redeem)
	if err != nil {
		return nil, fmt.Errorf("failed to decode redeem: %v", err)
	}

	fromId := utils.GetUint64Bytes(1)
	toId := utils.GetUint64Bytes(chainId)
	rawVer := utils.GetUint64Bytes(ver)

	h := btcutil.Hash160(append(append(append(append(r, fromId...), c...), toId...), rawVer...))
	sig, err := privk.Sign(h)
	if err != nil {
		return nil, err
	}

	return sig.Serialize(), nil
}

func (s *BtcService) SetContract(contract, redeem, sigs string, ver, chainId uint64) (common.Uint256, error) {
	contract = strings.Replace(contract, "0x", "", 1)
	c, err := hex.DecodeString(contract)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	r, err := hex.DecodeString(redeem)
	if err != nil {
		return common.UINT256_EMPTY, fmt.Errorf("failed to decode redeem: %v", err)
	}

	strArr := strings.Split(sigs, ",")
	sigArr := make([][]byte, len(strArr))
	for i, s := range strArr {
		raw, err := hex.DecodeString(s)
		if err != nil {
			return common.UINT256_EMPTY, fmt.Errorf("no.%d sig is failed decode into hex: %v", i, err)
		}
		sigArr[i] = raw
	}

	txHash, err := s.Poly.Native.Scm.RegisterRedeem(1, chainId, r, c, ver, sigArr, s.Acc)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return txHash, nil
}

func (s *BtcService) SignParam(fr, mc, ver uint64, redeem, pf, pwd string) ([]byte, error) {
	btcAcct, err := utils2.GetAccountByPassword(s.Poly, pf, []byte(pwd))
	if err != nil {
		return nil, fmt.Errorf("[NewSigner] failed to get btc account: %v", err)
	}
	privkP256 := btcec.PrivateKey(*btcAcct.GetPrivateKey().(*ec.PrivateKey).PrivateKey)
	privk, _ := btcec.PrivKeyFromBytes(btcec.S256(), (&privkP256).Serialize())

	r, err := hex.DecodeString(redeem)
	if err != nil {
		return nil, fmt.Errorf("failed to decode redeem: %v", err)
	}
	fromId := utils.GetUint64Bytes(1)
	rawFr := utils.GetUint64Bytes(fr)
	rawMc := utils.GetUint64Bytes(mc)
	rawVer := utils.GetUint64Bytes(ver)

	hash := btcutil.Hash160(append(append(append(append(r, fromId...), rawFr...), rawMc...), rawVer...))

	sig, err := privk.Sign(hash)
	if err != nil {
		return nil, err
	}

	return sig.Serialize(), nil
}

func (s *BtcService) SetParam(fr, mc, ver uint64, redeem, sigs string) (common.Uint256, error) {
	r, err := hex.DecodeString(redeem)
	if err != nil {
		return common.UINT256_EMPTY, fmt.Errorf("failed to decode redeem: %v", err)
	}

	strArr := strings.Split(sigs, ",")
	sigArr := make([][]byte, len(strArr))
	for i, s := range strArr {
		raw, err := hex.DecodeString(s)
		if err != nil {
			return common.UINT256_EMPTY, fmt.Errorf("no.%d sig is failed decode into hex: %v", i, err)
		}
		sigArr[i] = raw
	}

	txHash, err := s.Poly.Native.Scm.SetBtcTxParam(r, 1, fr, mc, ver, sigArr, s.Acc)
	if err != nil {
		return common.UINT256_EMPTY, err
	}
	return txHash, nil
}

type DbService struct {
	vdb *db.VendorDB
}

func NewDbService(vdb *db.VendorDB) *DbService {
	return &DbService{
		vdb: vdb,
	}
}

func (s *DbService) GetTxArrToShow() utils2.SavedItemArr {
	return nil
}
