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
package controller

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/gin-gonic/gin"
	"github.com/polynetwork/poly/common"
	"github.com/polynetwork/poly/native/service/cross_chain_manager/btc"
	"github.com/polynetwork/poly/native/service/governance/side_chain_manager"
	utils2 "github.com/polynetwork/poly/native/service/utils"
	"github.com/polynetwork/vendortool/config"
	"github.com/polynetwork/vendortool/log"
	"github.com/polynetwork/vendortool/utils"
	"github.com/polynetwork/vendortool/web/service"
	"net/http"
	"strconv"
)

type Controller struct {
	Bs   *service.BtcService
	Ds   *service.DbService
	Conf *config.Config
}

func (c *Controller) HandleGenePrivk(ctx *gin.Context) {
	pwd, ok := ctx.GetPostForm("pwd")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no pwd"})
		log.Error("[Web] wrong form, pubkeys not found")
		return
	}
	pwd2, ok := ctx.GetPostForm("pwd2")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no pwd confirmation"})
		log.Error("[Web] wrong form, pubkeys not found")
		return
	}
	if pwd != pwd2 {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "password is not equal"})
		return
	}
	path, ok := ctx.GetPostForm("path")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no path"})
		log.Error("[Web] wrong form, path not found")
		return
	}

	wif, addr, err := c.Bs.GenePrivkAndAddr(pwd, path)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
			"data": fmt.Sprintf("call function to generate private key failed: %v", err.Error()),
		})
		log.Errorf("[Web] failed to GenePrivkAndAddr: %v", err)
		return
	}
	ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
		"data": fmt.Sprintf("your private key is %s\nyour pubkey is %s\nyour addr is %s\n",
			wif.String(), hex.EncodeToString(wif.PrivKey.PubKey().SerializeCompressed()), addr.EncodeAddress()),
	})
	c.Conf.BtcWalletPwd = pwd
	c.Conf.BtcPrivkFile = path
}

func (c *Controller) HandleGeneRedeem(ctx *gin.Context) {
	pubks, ok := ctx.GetPostForm("pubks")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no pubkeys"})
		log.Error("[Web] wrong form, pubkeys not found")
		return
	}

	req, ok := ctx.GetPostForm("req")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no net type"})
		log.Error("[Web] wrong form, require number not found")
		return
	}

	r, rk, p2sh, p2wsh, err := c.Bs.GeneRedeem(pubks, req)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
			"data": fmt.Sprintf("call function to generate redeem failed: %v", err),
		})
		log.Error("[Web] failed to GeneRedeem: %v", err)
		return
	}
	c.Conf.Redeem = hex.EncodeToString(r)

	ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
		"data": fmt.Sprintf("your multisig redeem script is %s\nyour redeem hash is %s\n"+
			"your p2wsh addr is %s\nyour p2sh addr is %s\n", hex.EncodeToString(r),
			hex.EncodeToString(rk), p2wsh.EncodeAddress(), p2sh.EncodeAddress()),
	})
}

func (c *Controller) HandleSignContract(ctx *gin.Context) {
	pf, ok := ctx.GetPostForm("privk_file")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no private key file set"})
		log.Error("[Web] wrong form, private key file not found")
		return
	}
	pwd, ok := ctx.GetPostForm("pwd")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no password for private key file set"})
		log.Error("[Web] wrong form, password for private key file not found")
		return
	}

	contract, ok := ctx.GetPostForm("contract")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no smart contract set"})
		log.Error("[Web] wrong form, private key not found")
		return
	}
	chainid, ok := ctx.GetPostForm("chainid")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no target chain id set"})
		log.Error("[Web] wrong form, target chain id not found")
		return
	}

	ver, ok := ctx.GetPostForm("ver")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no contract version set"})
		log.Error("[Web] wrong form, contract version not found")
		return
	}

	id, err := strconv.ParseUint(chainid, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	v, err := strconv.ParseUint(ver, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}

	sig, err := c.Bs.SignContract(pf, pwd, contract, c.Conf.Redeem, id, v)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
			"data": fmt.Sprintf("failed to sign for contract %s: %v", contract, err),
		})
		log.Errorf("[Web] failed to sign for contract %s: %v", contract, err)
		return
	}
	ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
		"data": fmt.Sprintf("your sig is %s", hex.EncodeToString(sig)),
	})
}

func (c *Controller) HandleFuncSignContract(ctx *gin.Context) {
	pf, ok := ctx.GetPostForm("privk_file")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "sign_contract.tmpl", gin.H{"data": "no private key file set"})
		log.Error("[Web] wrong form, private key file not found")
		return
	}
	pwd, ok := ctx.GetPostForm("pwd")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "sign_contract.tmpl", gin.H{"data": "no password for private key file set"})
		log.Error("[Web] wrong form, password for private key file not found")
		return
	}

	contract, ok := ctx.GetPostForm("contract")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "sign_contract.tmpl", gin.H{"data": "no smart contract set"})
		log.Error("[Web] wrong form, private key not found")
		return
	}
	chainid, ok := ctx.GetPostForm("chainid")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "sign_contract.tmpl", gin.H{"data": "no target chain id set"})
		log.Error("[Web] wrong form, target chain id not found")
		return
	}

	ver, ok := ctx.GetPostForm("ver")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "sign_contract.tmpl", gin.H{"data": "no contract version set"})
		log.Error("[Web] wrong form, contract version not found")
		return
	}

	id, err := strconv.ParseUint(chainid, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "sign_contract.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	v, err := strconv.ParseUint(ver, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "sign_contract.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}

	sig, err := c.Bs.SignContract(pf, pwd, contract, c.Conf.Redeem, id, v)
	if err != nil {
		ctx.HTML(http.StatusOK, "sign_contract.tmpl", gin.H{
			"data": fmt.Sprintf("failed to sign for contract %s: %v", contract, err),
		})
		log.Errorf("[Web] failed to sign for contract %s: %v", contract, err)
		return
	}
	ctx.HTML(http.StatusOK, "sign_contract.tmpl", gin.H{
		"data": fmt.Sprintf("your sig is %s", hex.EncodeToString(sig)),
	})
}

func (c *Controller) HandleSetContract(ctx *gin.Context) {
	contract, ok := ctx.GetPostForm("contract")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no smart contract set"})
		log.Error("[Web] wrong form, private key not found")
		return
	}
	ver, ok := ctx.GetPostForm("ver")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no contract version set"})
		log.Error("[Web] wrong form, contract version not found")
		return
	}
	chainid, ok := ctx.GetPostForm("chainid")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no target chain id set"})
		log.Error("[Web] wrong form, target chain id not found")
		return
	}
	sigs, ok := ctx.GetPostForm("sigs")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no signatures set"})
		log.Error("[Web] wrong form, signatures not found")
		return
	}

	id, err := strconv.ParseUint(chainid, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	v, err := strconv.ParseUint(ver, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	hash, err := c.Bs.SetContract(contract, c.Conf.Redeem, sigs, v, id)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
			"data": fmt.Sprintf("failed to sign contract %s(ver %d): %v", contract, v, err),
		})
		log.Errorf("[Web] failed to sign for contract %s(ver %d): %v", contract, v, err)
		return
	}
	ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
		"data": fmt.Sprintf("send tx successfully, your tx hash is %s", hash.ToHexString()),
	})
}

func (c *Controller) HandleFuncSetContract(ctx *gin.Context) {
	wallet, ok := ctx.GetPostForm("wallet")
	if ok && wallet != "" {
		c.Conf.WalletFile = wallet
	}
	opwd, ok := ctx.GetPostForm("opwd")
	if ok && opwd != "" {
		c.Conf.WalletPwd = opwd
	}
	if err := c.getORCPwd(); err != nil {
		ctx.HTML(http.StatusOK, "set_contract.tmpl", gin.H{
			"data": fmt.Sprintf("get account failed: %v", err),
		})
		log.Errorf("[Web] failed to get account: %v", err)
		return
	}

	contract, ok := ctx.GetPostForm("contract")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "set_contract.tmpl", gin.H{"data": "no smart contract set"})
		log.Error("[Web] wrong form, private key not found")
		return
	}
	ver, ok := ctx.GetPostForm("ver")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "set_contract.tmpl", gin.H{"data": "no contract version set"})
		log.Error("[Web] wrong form, contract version not found")
		return
	}
	chainid, ok := ctx.GetPostForm("chainid")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "set_contract.tmpl", gin.H{"data": "no target chain id set"})
		log.Error("[Web] wrong form, target chain id not found")
		return
	}
	sigs, ok := ctx.GetPostForm("sigs")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "set_contract.tmpl", gin.H{"data": "no signatures set"})
		log.Error("[Web] wrong form, signatures not found")
		return
	}

	id, err := strconv.ParseUint(chainid, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "set_contract.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	v, err := strconv.ParseUint(ver, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "set_contract.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	hash, err := c.Bs.SetContract(contract, c.Conf.Redeem, sigs, v, id)
	if err != nil {
		ctx.HTML(http.StatusOK, "set_contract.tmpl", gin.H{
			"data": fmt.Sprintf("failed to sign contract %s(ver %d): %v", contract, v, err),
		})
		log.Errorf("[Web] failed to sign for contract %s(ver %d): %v", contract, v, err)
		return
	}
	ctx.HTML(http.StatusOK, "set_contract.tmpl", gin.H{
		"data": fmt.Sprintf("send tx successfully, your tx hash is %s", hash.ToHexString()),
	})
}

func (c *Controller) HandleSignParam(ctx *gin.Context) {
	pf, ok := ctx.GetPostForm("privk_file")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no private key file set"})
		log.Error("[Web] wrong form, private key file not found")
		return
	}
	pwd, ok := ctx.GetPostForm("pwd")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no password for private key file set"})
		log.Error("[Web] wrong form, password for private key file not found")
		return
	}

	ver, ok := ctx.GetPostForm("ver")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no contract version set"})
		log.Error("[Web] wrong form, contract version not found")
		return
	}

	fr, ok := ctx.GetPostForm("fr")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no fee rate set"})
		log.Error("[Web] wrong form, fee rate not found")
		return
	}

	mc, ok := ctx.GetPostForm("mc")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no min change set"})
		log.Error("[Web] wrong form, contract version not found")
		return
	}

	v, err := strconv.ParseUint(ver, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	feeRate, err := strconv.ParseUint(fr, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	minChange, err := strconv.ParseUint(mc, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}

	sig, err := c.Bs.SignParam(feeRate, minChange, v, c.Conf.Redeem, pf, pwd)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
			"data": fmt.Sprintf("failed to sign param(fee rate: %d sat/byte, min change: %d sat, ver %d): %v",
				feeRate, minChange, v, err),
		})
		log.Errorf("[Web] failed to sign param(fee rate: %d sat/byte, min change: %d sat, ver %d): %v",
			feeRate, minChange, v, err)
		return
	}

	ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
		"data": fmt.Sprintf("your sig is %s", hex.EncodeToString(sig)),
	})
}

func (c *Controller) HandleFuncSignParam(ctx *gin.Context) {
	pf, ok := ctx.GetPostForm("privk_file")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "sign_param.tmpl", gin.H{"data": "no private key file set"})
		log.Error("[Web] wrong form, private key file not found")
		return
	}
	pwd, ok := ctx.GetPostForm("pwd")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "sign_param.tmpl", gin.H{"data": "no password for private key file set"})
		log.Error("[Web] wrong form, password for private key file not found")
		return
	}

	ver, ok := ctx.GetPostForm("ver")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "sign_param.tmpl", gin.H{"data": "no contract version set"})
		log.Error("[Web] wrong form, contract version not found")
		return
	}

	fr, ok := ctx.GetPostForm("fr")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "sign_param.tmpl", gin.H{"data": "no fee rate set"})
		log.Error("[Web] wrong form, fee rate not found")
		return
	}

	mc, ok := ctx.GetPostForm("mc")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "sign_param.tmpl", gin.H{"data": "no min change set"})
		log.Error("[Web] wrong form, contract version not found")
		return
	}

	v, err := strconv.ParseUint(ver, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "sign_param.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	feeRate, err := strconv.ParseUint(fr, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "sign_param.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	minChange, err := strconv.ParseUint(mc, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "sign_param.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}

	sig, err := c.Bs.SignParam(feeRate, minChange, v, c.Conf.Redeem, pf, pwd)
	if err != nil {
		ctx.HTML(http.StatusOK, "sign_param.tmpl", gin.H{
			"data": fmt.Sprintf("failed to sign param(fee rate: %d sat/byte, min change: %d sat, ver %d): %v",
				feeRate, minChange, v, err),
		})
		log.Errorf("[Web] failed to sign param(fee rate: %d sat/byte, min change: %d sat, ver %d): %v",
			feeRate, minChange, v, err)
		return
	}

	ctx.HTML(http.StatusOK, "sign_param.tmpl", gin.H{
		"data": fmt.Sprintf("your sig is %s", hex.EncodeToString(sig)),
	})
}

func (c *Controller) HandleSetParam(ctx *gin.Context) {
	ver, ok := ctx.GetPostForm("ver")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no contract version set"})
		log.Error("[Web] wrong form, contract version not found")
		return
	}

	fr, ok := ctx.GetPostForm("fr")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no fee rate set"})
		log.Error("[Web] wrong form, fee rate not found")
		return
	}

	mc, ok := ctx.GetPostForm("mc")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no min change set"})
		log.Error("[Web] wrong form, contract version not found")
		return
	}

	sigs, ok := ctx.GetPostForm("sigs")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "index.tmpl", gin.H{"data": "no signatures set"})
		log.Error("[Web] wrong form, signatures not found")
		return
	}

	v, err := strconv.ParseUint(ver, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	feeRate, err := strconv.ParseUint(fr, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	minChange, err := strconv.ParseUint(mc, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}

	txHash, err := c.Bs.SetParam(feeRate, minChange, v, c.Conf.Redeem, sigs)
	if err != nil {
		ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
			"data": fmt.Sprintf("failed to set param(fee rate: %d sat/byte, min change: %d sat, ver %d): %v",
				feeRate, minChange, v, err),
		})
		log.Errorf("[Web] failed to set param(fee rate: %d sat/byte, min change: %d sat, ver %d): %v",
			feeRate, minChange, v, err)
		return
	}
	ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
		"data": fmt.Sprintf("send tx successfully, your tx hash is %s", txHash.ToHexString()),
	})
}

func (c *Controller) HandleFuncSetParam(ctx *gin.Context) {
	wallet, ok := ctx.GetPostForm("wallet")
	if ok && wallet != "" {
		c.Conf.WalletFile = wallet
	}
	opwd, ok := ctx.GetPostForm("opwd")
	if ok && opwd != "" {
		c.Conf.WalletPwd = opwd
	}
	if err := c.getORCPwd(); err != nil {
		ctx.HTML(http.StatusOK, "set_contract.tmpl", gin.H{
			"data": fmt.Sprintf("get account failed: %v", err),
		})
		log.Errorf("[Web] failed to get account: %v", err)
		return
	}

	ver, ok := ctx.GetPostForm("ver")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "set_param.tmpl", gin.H{"data": "no contract version set"})
		log.Error("[Web] wrong form, contract version not found")
		return
	}

	fr, ok := ctx.GetPostForm("fr")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "set_param.tmpl", gin.H{"data": "no fee rate set"})
		log.Error("[Web] wrong form, fee rate not found")
		return
	}

	mc, ok := ctx.GetPostForm("mc")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "set_param.tmpl", gin.H{"data": "no min change set"})
		log.Error("[Web] wrong form, contract version not found")
		return
	}

	sigs, ok := ctx.GetPostForm("sigs")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "set_param.tmpl", gin.H{"data": "no signatures set"})
		log.Error("[Web] wrong form, signatures not found")
		return
	}

	v, err := strconv.ParseUint(ver, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "set_param.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	feeRate, err := strconv.ParseUint(fr, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "set_param.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}
	minChange, err := strconv.ParseUint(mc, 10, 64)
	if err != nil {
		ctx.HTML(http.StatusOK, "set_param.tmpl", gin.H{"data": fmt.Sprintf("wrong number format: %v", err)})
		log.Errorf("[Web] wrong number format: %v", err)
		return
	}

	txHash, err := c.Bs.SetParam(feeRate, minChange, v, c.Conf.Redeem, sigs)
	if err != nil {
		ctx.HTML(http.StatusOK, "set_param.tmpl", gin.H{
			"data": fmt.Sprintf("failed to set param(fee rate: %d sat/byte, min change: %d sat, ver %d): %v",
				feeRate, minChange, v, err),
		})
		log.Errorf("[Web] failed to set param(fee rate: %d sat/byte, min change: %d sat, ver %d): %v",
			feeRate, minChange, v, err)
		return
	}
	ctx.HTML(http.StatusOK, "set_param.tmpl", gin.H{
		"data": fmt.Sprintf("send tx successfully, your tx hash is %s", txHash.ToHexString()),
	})
}

func (c *Controller) HandleStatus(ctx *gin.Context) {
	rpc, ok := ctx.GetPostForm("rpc")
	if ok && rpc != "" {
		log.Infof("[Web] change rpc address from %s to %s", c.Conf.PolyJsonRpcAddress, rpc)
		c.Conf.PolyJsonRpcAddress = rpc
	}
	if c.Bs.Poly.GetRpcClient() == nil {
		_ = utils.SetUpPoly(c.Bs.Poly, c.Conf.PolyJsonRpcAddress)
	}

	ow, ok := ctx.GetPostForm("owallet")
	if ok && rpc != "" {
		log.Infof("[Web] change polygon wallet from %s to %s", c.Conf.WalletFile, ow)
		c.Conf.WalletFile = ow
	}

	opwd, ok := ctx.GetPostForm("opwd")
	if ok && opwd != "" {
		log.Infof("[Web] change polygon password from %s to %s", c.Conf.WalletPwd, opwd)
		c.Conf.WalletPwd = opwd
	}

	bwallet, ok := ctx.GetPostForm("bwallet")
	if ok && bwallet != "" {
		log.Infof("[Web] change btc wallet file from %s to %s", c.Conf.BtcPrivkFile, bwallet)
		c.Conf.BtcPrivkFile = bwallet
	}

	bpwd, ok := ctx.GetPostForm("bpwd")
	if ok && bpwd != "" {
		log.Infof("[Web] change btc password from %s to %s", c.Conf.BtcWalletPwd, bpwd)
		c.Conf.BtcWalletPwd = bpwd
	}

	nt, ok := ctx.GetPostForm("nt")
	if ok && nt != "" {
		switch nt {
		case "regtest":
			config.BtcNetParam = &chaincfg.RegressionNetParams
		case "test":
			config.BtcNetParam = &chaincfg.TestNet3Params
		default:
			config.BtcNetParam = &chaincfg.MainNetParams
		}
		log.Infof("[Web] change btc net from %s to %s", c.Conf.ConfigBitcoinNet, nt)
		c.Conf.ConfigBitcoinNet = nt
	}

	wt, ok := ctx.GetPostForm("wait_time")
	if ok && wt != "" {
		time, err := strconv.ParseInt(wt, 10, 64)
		if err == nil {
			log.Infof("[Web] change wait time from %s to %s", c.Conf.PolyObLoopWaitTime, time)
			c.Conf.PolyObLoopWaitTime = time
		}
	}

	db, ok := ctx.GetPostForm("db")
	if ok && db != "" {
		log.Infof("[Web] change db path from %s to %s", c.Conf.ConfigDBPath, db)
		c.Conf.ConfigDBPath = db
	}

	rdm, ok := ctx.GetPostForm("redeem")
	if ok && rdm != "" {
		log.Infof("[Web] change redeem from %s to %s", c.Conf.Redeem, rdm)
		c.Conf.Redeem = rdm
	}

	if err := c.getORCPwd(); err != nil {
		ctx.HTML(http.StatusOK, "conf.tmpl", gin.H{
			"data": fmt.Sprintf("get account failed: %v", err),
		})
		log.Errorf("[Web] failed to get account: %v", err)
		return
	}

	// TODO: collect status
	raw, err := hex.DecodeString(c.Conf.Redeem)
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "status.tmpl", gin.H{"error": fmt.Sprintf("wrong redeem: %v", err)})
		log.Errorf("[Web] wrong redeem: %v", err)
		return
	}
	rk := btcutil.Hash160(raw)
	store, err := c.Bs.Poly.GetStorage(utils2.CrossChainManagerContractAddress.ToHexString(),
		append(append([]byte(btc.UTXOS), utils2.GetUint64Bytes(1)...), []byte(hex.EncodeToString(rk))...))
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "status.tmpl", gin.H{"error": fmt.Sprintf("get utxo failed: %v", err)})
		log.Errorf("failed to get utxos from chain: %v", err)
		return
	}
	utxos := &btc.Utxos{
		Utxos: make([]*btc.Utxo, 0),
	}
	if store != nil {
		if err = utxos.Deserialization(common.NewZeroCopySource(store)); err != nil {
			ctx.HTML(http.StatusBadRequest, "status.tmpl", gin.H{"error": fmt.Sprintf("failed to deserialize utxos: %v", err)})
			log.Errorf("failed to deserialize utxos: %v", err)
			return
		}
	}
	sum := uint64(0)
	for _, u := range utxos.Utxos {
		sum += u.Value
	}
	val, err := c.Bs.Poly.GetStorage(utils2.SideChainManagerContractAddress.ToHexString(), append(append(append([]byte(
		side_chain_manager.REDEEM_BIND), utils2.GetUint64Bytes(1)...), utils2.GetUint64Bytes(2)...), rk...))
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "status.tmpl", gin.H{"error": fmt.Sprintf("failed to get contract info: %v", err)})
		log.Errorf("failed to get contract info: %v", err)
		return
	}
	b := &side_chain_manager.ContractBinded{}
	b.Deserialization(common.NewZeroCopySource(val))

	val, err = c.Bs.Poly.GetStorage(utils2.SideChainManagerContractAddress.ToHexString(), append(append([]byte(
		side_chain_manager.BTC_TX_PARAM), rk...), utils2.GetUint64Bytes(1)...))
	if err != nil {
		ctx.HTML(http.StatusBadRequest, "status.tmpl", gin.H{"error": fmt.Sprintf("failed to get param info: %v", err)})
		log.Errorf("failed to get param info: %v", err)
		return
	}

	d := &side_chain_manager.BtcTxParamDetial{}
	d.Deserialization(common.NewZeroCopySource(val))

	ctx.HTML(http.StatusOK, "status.tmpl", gin.H{
		"sum":      sum,
		"total":    len(utxos.Utxos),
		"contract": hex.EncodeToString(b.Contract),
		"ver":      b.Ver,
		"mc":       d.MinChange,
		"fr":       d.FeeRate,
		"pver":     d.PVersion,
	})
}

func (c *Controller) HandleInit(ctx *gin.Context) {
	rpc, ok := ctx.GetPostForm("rpc")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "before.tmpl", gin.H{"data": "no rpc address set"})
		log.Errorf("[Web] no rpc address set")
		return
	}
	if rpc != "" {
		log.Infof("[Web] set rpc address to %s", rpc)
		c.Conf.PolyJsonRpcAddress = rpc
	}
	c.Bs.Poly.NewRpcClient().SetAddress(c.Conf.PolyJsonRpcAddress)

	ow, ok := ctx.GetPostForm("owallet")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "before.tmpl", gin.H{"data": "no poly wallet set"})
		log.Errorf("[Web] no poly wallet set")
		return
	}
	if ow != "" {
		log.Infof("[Web] change polygon wallet to %s", ow)
		c.Conf.WalletFile = ow
	}

	opwd, ok := ctx.GetPostForm("opwd")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "before.tmpl", gin.H{"data": "no poly wallet password set"})
		log.Errorf("[Web] no poly wallet password set")
		return
	}
	if opwd != "" {
		log.Infof("[Web] change polygon password to %s", opwd)
		c.Conf.WalletPwd = opwd
	}

	nt, ok := ctx.GetPostForm("nt")
	if !ok {
		ctx.HTML(http.StatusBadRequest, "before.tmpl", gin.H{"data": "no bitcoin net type set"})
		log.Errorf("[Web] no bitcoin net type set")
		return
	}
	if nt != "" {
		log.Infof("[Web] change btc net to %s", nt)
		c.Conf.ConfigBitcoinNet = nt
	}
	switch c.Conf.ConfigBitcoinNet {
	case "regtest":
		config.BtcNetParam = &chaincfg.RegressionNetParams
	case "test":
		config.BtcNetParam = &chaincfg.TestNet3Params
	default:
		config.BtcNetParam = &chaincfg.MainNetParams
	}

	db, ok := ctx.GetPostForm("db")
	if !ok {
		ctx.HTML(http.StatusOK, "before.tmpl", gin.H{"data": "no db path set"})
		log.Errorf("[Web] no db path set")
		return
	}
	if db != "" {
		log.Infof("[Web] change db path to %s", c.Conf.ConfigDBPath, db)
		c.Conf.ConfigDBPath = db
	}

	if err := c.getORCPwd(); err != nil {
		ctx.HTML(http.StatusOK, "before.tmpl", gin.H{
			"data": fmt.Sprintf("get account failed: %v", err),
		})
		log.Errorf("[Web] failed to get account: %v", err)
		return
	}

	ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title": "Vendor Tool",
	})
}

func (c *Controller) getORCPwd() error {
	acc, err := utils.GetAccountByPassword(c.Bs.Poly, c.Conf.WalletFile, []byte(c.Conf.WalletPwd))
	if err != nil {
		return fmt.Errorf("failed to get account from wallet file %s by password %s: %v", c.Conf.WalletFile,
			c.Conf.WalletPwd, err)
	}
	if c.Bs.Poly.GetRpcClient() == nil {
		c.Bs.Poly.NewRpcClient().SetAddress(c.Conf.PolyJsonRpcAddress)
	}
	c.Bs.Acc = acc
	return nil
}

func (c *Controller) Handle() {

}
