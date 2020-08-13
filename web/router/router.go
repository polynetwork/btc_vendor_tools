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
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/vendortool/config"
	"github.com/polynetwork/vendortool/db"
	"github.com/polynetwork/vendortool/web/controller"
	"github.com/polynetwork/vendortool/web/service"
	"net/http"
)

func Init(poly *poly_go_sdk.PolySdk, conf *config.Config, vdb *db.VendorDB, done chan struct{}) error {
	r := gin.Default()
	r.LoadHTMLGlob("web/views/*")

	s := service.NewBtcService(poly)
	ds := service.NewDbService(vdb)
	c := &controller.Controller{
		Bs:   s,
		Ds:   ds,
		Conf: conf,
	}

	r.GET("/", func(context *gin.Context) {
		context.HTML(http.StatusOK, "choice.tmpl", gin.H{
			"title": "First to be a vendor or already ?",
		})
	})

	// brunch init
	r.POST("/before_init", func(context *gin.Context) {
		context.HTML(http.StatusOK, "before.tmpl", gin.H{
			"title":   "Configuration Before Start",
			"rpc":     conf.PolyJsonRpcAddress,
			"owallet": conf.WalletFile,
			"opwd":    conf.WalletPwd,
			"db":      conf.ConfigDBPath,
		})
	})
	r.POST("/init", c.HandleInit)
	r.POST("/geneprivk", c.HandleGenePrivk)
	r.POST("/generdm", c.HandleGeneRedeem)
	r.POST("/sign_contract", c.HandleSignContract)
	r.POST("/set_contract", c.HandleSetContract)
	r.POST("/sign_param", c.HandleSignParam)
	r.POST("/set_param", c.HandleSetParam)

	// brunch start directly
	r.POST("/start_vendor_tool", func(context *gin.Context) {
		context.HTML(http.StatusOK, "conf.tmpl", gin.H{
			"title":     "Configuration for Vendor Tool",
			"rpc":       conf.PolyJsonRpcAddress,
			"owallet":   conf.WalletFile,
			"opwd":      conf.WalletPwd,
			"bwallet":   conf.BtcPrivkFile,
			"bpwd":      conf.BtcWalletPwd,
			"wait_time": conf.PolyObLoopWaitTime,
			"db":        conf.ConfigDBPath,
			"redeem":    conf.Redeem,
		})
	})
	r.POST("/status", func(context *gin.Context) {
		c.HandleStatus(context)
		close(done)
	})

	// brunch function
	r.POST("/func_signcontract", func(context *gin.Context) {
		context.HTML(http.StatusOK, "sign_contract.tmpl", gin.H{})
	})
	r.POST("/sign_contract_tool", c.HandleFuncSignContract)

	r.POST("/func_updatecontract", func(context *gin.Context) {
		context.HTML(http.StatusOK, "set_contract.tmpl", gin.H{
			"wallet": c.Conf.WalletFile,
		})
	})
	r.POST("/set_contract_tool", c.HandleFuncSetContract)

	r.POST("/func_signparam", func(context *gin.Context) {
		context.HTML(http.StatusOK, "sign_param.tmpl", gin.H{})
	})
	r.POST("/sign_param_tool", c.HandleFuncSignParam)

	r.POST("/func_updateparam", func(context *gin.Context) {
		context.HTML(http.StatusOK, "set_param.tmpl", gin.H{
			"wallet": c.Conf.WalletFile,
		})
	})
	r.POST("/set_param_tool", c.HandleFuncSetParam)

	return r.Run(":" + c.Conf.WebServerPort)
}
