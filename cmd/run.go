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
package main

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	sdk "github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/poly/common/password"
	"github.com/polynetwork/vendortool/config"
	"github.com/polynetwork/vendortool/db"
	"github.com/polynetwork/vendortool/log"
	"github.com/polynetwork/vendortool/observer"
	"github.com/polynetwork/vendortool/rest/http/restful"
	"github.com/polynetwork/vendortool/rest/service"
	"github.com/polynetwork/vendortool/signer"
	"github.com/polynetwork/vendortool/utils"
	"github.com/polynetwork/vendortool/web"
	"github.com/urfave/cli"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func setupApp() *cli.App {
	app := cli.NewApp()
	app.Usage = "start vendor tool"
	app.Action = run
	app.Copyright = ""
	app.Flags = []cli.Flag{
		config.LogLevelFlag,
		config.ConfigFile,
		config.GoMaxProcs,
		config.PolyWalletPwd,
		config.BtcWalletPwd,
		config.RunMode,
		config.Web,
	}
	app.Before = func(context *cli.Context) error {
		cores := context.GlobalInt(config.GoMaxProcs.Name)
		runtime.GOMAXPROCS(cores)
		return nil
	}
	return app
}

func main() {
	if err := setupApp().Run(os.Args); err != nil {
		log.Errorf("fail to run: %v", err)
		os.Exit(1)
	}
}

func run(ctx *cli.Context) {
	logLevel := ctx.GlobalInt(config.LogLevelFlag.Name)
	log.InitLog(logLevel, log.Stdout)
	mode := ctx.GlobalString(config.RunMode.Name)
	isWeb := ctx.GlobalInt(config.Web.Name)

	conf, err := config.NewConfig(ctx.GlobalString(config.ConfigFile.Name))
	if err != nil {
		log.Fatalf("failed to new a config: %v", err)
		os.Exit(1)
	}
	conf.Save("./config/conf.json")
	if conf.SleepTime > 0 {
		config.SleepTime = time.Duration(conf.SleepTime) * time.Second
	}
	vdb, err := db.NewVendorDB(conf.ConfigDBPath)
	if err != nil {
		log.Fatalf("failed to new vendor db: %v", err)
		os.Exit(1)
	}

	poly := sdk.NewPolySdk()
	if isWeb == 1 {
		done := make(chan struct{})
		go func() {
			if err := web.StartWeb(conf, poly, vdb, done); err != nil {
				log.Fatalf("failed to start web server: %v", err)
				os.Exit(1)
			}
		}()
		<-done
		if err := conf.Save(ctx.GlobalString(config.ConfigFile.Name)); err != nil {
			log.Errorf("failed to save config: %v", err)
		}
	} else {
		switch conf.ConfigBitcoinNet {
		case "regtest":
			config.BtcNetParam = &chaincfg.RegressionNetParams
		case "test":
			config.BtcNetParam = &chaincfg.TestNet3Params
		case "main":
			config.BtcNetParam = &chaincfg.MainNetParams
		default:
			log.Fatalf("wrong net type: %s", conf.ConfigBitcoinNet)
			os.Exit(1)
		}
		if err = utils.SetUpPoly(poly, conf.PolyJsonRpcAddress); err != nil {
			panic(err)
		}
	}

	var opwd []byte
	if pwd := ctx.GlobalString(config.PolyWalletPwd.Name); pwd != "" {
		opwd = []byte(pwd)
	} else if conf.WalletPwd == "" {
		fmt.Println("enter your polygon wallet password:")
		if opwd, err = password.GetPassword(); err != nil {
			log.Fatalf("password is not found in config file and enter password failed: %v", err)
			os.Exit(1)
		}
		fmt.Println("done")
	} else {
		opwd = []byte(conf.WalletPwd)
	}

	var bpwd []byte
	if pwd := ctx.GlobalString(config.BtcWalletPwd.Name); pwd != "" {
		bpwd = []byte(pwd)
	} else if conf.BtcWalletPwd == "" {
		fmt.Println("enter your btc wallet password:")
		if bpwd, err = password.GetPassword(); err != nil {
			log.Fatalf("password is not found in config file and enter password failed: %v", err)
			os.Exit(1)
		}
		fmt.Println("done")
	} else {
		bpwd = []byte(conf.BtcWalletPwd)
	}

	rb, err := hex.DecodeString(conf.Redeem)
	if err != nil {
		log.Errorf("failed to decode redeem: %v", err)
		os.Exit(1)
	}

	switch mode {
	case "all":
		txchan := make(chan *utils.ToSignItem, 100)
		if err := startObserver(conf, txchan, poly, rb, vdb); err != nil {
			log.Fatalf("failed to start ob: %v", err)
			os.Exit(1)
		}
		if _, err := startSigner(conf, txchan, poly, vdb, rb, opwd, bpwd); err != nil {
			log.Fatalf("failed to start signer: %v", err)
			os.Exit(1)
		}
	case "onlyob":
		if err := startObserver(conf, nil, poly, rb, vdb); err != nil {
			log.Fatalf("failed to start ob: %v", err)
			os.Exit(1)
		}
	case "onlysig":
		s, err := startSigner(conf, nil, poly, vdb, rb, opwd, bpwd)
		if err != nil {
			log.Fatalf("failed to start signer: %v", err)
			os.Exit(1)
		}
		if err := startServer(conf, s); err != nil {
			log.Fatalf("Failed to start rest service: %v", err)
			os.Exit(1)
		}
	default:
		log.Fatalf("wrong mode: %s", mode)
		os.Exit(1)
	}

	waitToExit()
}

func startServer(conf *config.Config, s *signer.Signer) error {
	serv := service.NewService(s)
	restServer := restful.InitRestServer(serv, conf.RestPort, conf.ObServerAddr)
	go restServer.Start()

	return nil
}

func startObserver(conf *config.Config, txchan chan *utils.ToSignItem, poly *sdk.PolySdk, rb []byte,
	vdb *db.VendorDB) error {
	ob := observer.NewObserver(poly, txchan, conf.PolyObLoopWaitTime, rb, conf.WatchingKeyToSign,
		conf.ConfigDBPath, conf.SignerAddr, conf.CircleToSaveHeight, conf.PolyStartHeight, vdb)
	go ob.Listen()

	return nil
}

func startSigner(conf *config.Config, txchan chan *utils.ToSignItem, poly *sdk.PolySdk, vdb *db.VendorDB, rb, opwd, bpwd []byte) (*signer.Signer, error) {
	acct, err := utils.GetAccountByPassword(poly, conf.WalletFile, opwd)
	if err != nil {
		return nil, fmt.Errorf("[startSigner] GetAccountByPassword failed: %v", err)
	}
	s, err := signer.NewSigner(conf.BtcPrivkFile, bpwd, txchan, acct, poly, rb, vdb)
	if err != nil {
		return nil, fmt.Errorf("[startSigner] failed to new a signer: %v", err)
	}
	if txchan != nil {
		go s.Signing()
	}

	return s, nil
}

func waitToExit() {
	exit := make(chan bool, 0)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for sig := range sc {
			log.Infof("server received exit signal:%v.", sig.String())
			close(exit)
			break
		}
	}()
	<-exit
}
