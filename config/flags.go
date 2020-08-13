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
	"github.com/urfave/cli"
)

const (
	DEFAULT_LOG_LEVEL   = 2
	DEFAULT_MAXPROC_NUM = 4
)

var (
	LogLevelFlag = cli.UintFlag{
		Name:  "loglevel",
		Usage: "Set the log level to `<level>` (0~6). 0:Trace 1:Debug 2:Info 3:Warn 4:Error 5:Fatal 6:MaxLevel",
		Value: DEFAULT_LOG_LEVEL,
	}

	ConfigFile = cli.StringFlag{
		Name:  "config",
		Usage: "the config file of polygon service.",
		Value: "./conf.json",
	}

	GoMaxProcs = cli.IntFlag{
		Name:  "gomaxprocs",
		Usage: "max number of cpu core that runtime can use.",
		Value: DEFAULT_MAXPROC_NUM,
	}

	PolyWalletPwd = cli.StringFlag{
		Name:  "polypwd",
		Usage: "the password of polygon wallet.",
		Value: "",
	}

	BtcWalletPwd = cli.StringFlag{
		Name:  "btcpwd",
		Usage: "the password of btc wallet.",
		Value: "",
	}

	RunMode = cli.StringFlag{
		Name:  "mode",
		Usage: "the mode for this tool, eg: onlysig, onlyob, all",
		Value: "all",
	}

	Web = cli.IntFlag{
		Name:  "web",
		Usage: "start web server or not: 1(Y), 0(N)",
		Value: 1,
	}
)
