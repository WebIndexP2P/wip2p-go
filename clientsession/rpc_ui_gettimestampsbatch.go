package clientsession

import (
	//"fmt"
	"strconv"
	"github.com/ethereum/go-ethereum/common"

	"code.wip2p.com/mwadmin/wip2p-go/account"
)

func ui_getTimestampsBatch(session *Session, paramData []interface{}) ([]string, error) {

	// construct a response
	var response []string
	response = make([]string, 0)

	if len(paramData) == 0 {
		return response, nil
	}

	for _, acct := range paramData {
		acct := acct.(string)
		addr := common.HexToAddress(acct)

		a, success := account.FetchAccountFromDb(addr, nil, false)
		if !success {
			response = append(response, "")
		} else {
			response = append(response, strconv.FormatUint(a.Timestamp, 10))
		}
	}

	return response, nil

}
