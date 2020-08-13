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
package config

import (
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"io/ioutil"
	"os"
	"time"
)

var (
	SleepTime   time.Duration    = 10 * time.Second
	BtcNetParam *chaincfg.Params = nil
)

type Config struct {
	PolyJsonRpcAddress string
	WalletFile         string
	WalletPwd          string
	BtcWalletPwd       string
	PolyObLoopWaitTime int64
	BtcPrivkFile       string
	WatchingKeyToSign  string
	ConfigBitcoinNet   string
	ConfigDBPath       string
	RestPort           uint64
	SleepTime          int
	CircleToSaveHeight uint32
	Redeem             string
	SignerAddr         string
	ObServerAddr       string
	PolyStartHeight    uint32
	WebServerPort      string
}

func NewConfig(file string) (*Config, error) {
	conf := &Config{}
	err := conf.Init(file)
	if err != nil {
		return conf, fmt.Errorf("failed to new config: %v", err)
	}
	return conf, nil
}

func (this *Config) Init(fileName string) error {
	err := this.loadConfig(fileName)
	if err != nil {
		return fmt.Errorf("loadConfig error:%s", err)
	}
	return nil
}

func (this *Config) loadConfig(fileName string) error {
	data, err := this.readFile(fileName)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, this)
	if err != nil {
		return fmt.Errorf("json.Unmarshal TestConfig:%s error:%s", data, err)
	}
	return nil
}

func (this *Config) readFile(fileName string) ([]byte, error) {
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("OpenFile %s error %s", fileName, err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			fmt.Println(fmt.Errorf("file %s close error %s", fileName, err))
		}
	}()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll %s error %s", fileName, err)
	}
	return data, nil
}

func (this *Config) Save(fileName string) error {
	//bpwd, opwd := this.BtcWalletPwd, this.WalletPwd
	//this.BtcWalletPwd, this.WalletPwd = "", ""
	data, err := json.MarshalIndent(this, "", "\t")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(fileName, data, 0644); err != nil {
		return fmt.Errorf("failed to write conf file: %v", err)
	}
	//this.BtcWalletPwd, this.WalletPwd = bpwd, opwd
	return nil
}
