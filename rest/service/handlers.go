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
	"encoding/hex"
	"fmt"
	"github.com/polynetwork/vendortool/log"
	"github.com/polynetwork/vendortool/rest/http/common"
	"github.com/polynetwork/vendortool/rest/http/restful"
	"github.com/polynetwork/vendortool/rest/utils"
	"github.com/polynetwork/vendortool/signer"
	locutil "github.com/polynetwork/vendortool/utils"
)

type Service struct {
	signer *signer.Signer
}

func NewService(signer *signer.Signer) *Service {
	return &Service{
		signer: signer,
	}
}

func (serv *Service) SignTx(params map[string]interface{}) map[string]interface{} {
	resp := &common.Response{
		Action: common.ACTION_SIGNTX,
	}
	req := &common.SignItemReq{}
	if err := utils.ParseParams(req, params); err != nil {
		log.Errorf("[Rest] SignTx: decode params failed, err: %s", err)
		resp.Error = restful.INVALID_PARAMS
		resp.Desc = fmt.Sprintf("SignTx: decode params failed, err: %s", err)
		m, _ := utils.RefactorResp(resp, resp.Error)
		return m
	}

	raw, err := hex.DecodeString(req.Raw)
	if err != nil {
		log.Errorf("[Rest] SignTx: decode raw failed, err: %s", err)
		resp.Error = restful.ILLEGAL_DATAFORMAT
		resp.Desc = fmt.Sprintf("SignTx: decode raw failed, err: %s", err)
		m, _ := utils.RefactorResp(resp, resp.Error)
		return m
	}

	item := &locutil.ToSignItem{}
	if err := item.Deserialize(raw); err != nil {
		log.Errorf("[Rest] SignTx: deserialize failed, err: %s", err)
		resp.Error = restful.ILLEGAL_DATAFORMAT
		resp.Desc = fmt.Sprintf("[Rest] SignTx: deserialize failed, err: %s", err)
		m, _ := utils.RefactorResp(resp, resp.Error)
		return m
	}

	if err := serv.signer.Sign(item); err != nil {
		log.Errorf("[Rest] SignTx: deserialize failed, err: %s", err)
		resp.Error = restful.INTERNAL_ERROR
		resp.Desc = fmt.Sprintf("[Rest] SignTx: sign failed, err: %s", err)
		m, _ := utils.RefactorResp(resp, resp.Error)
		return m
	}

	m, err := utils.RefactorResp(resp, resp.Error)
	if err != nil {
		log.Errorf("[Rest] SignTx: failed, err: %v", err)
	} else {
		log.Infof("[Rest] SignTx: resp success")
	}
	return m
}
