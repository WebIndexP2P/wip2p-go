package clientsession

import (
	//"fmt"
	"errors"

	"code.wip2p.com/mwadmin/wip2p-go/sigbundle"
	"code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
)

// returns (sequence no, array of accounts that had content removed, error)
func bundleSave(session *Session, paramData []interface{}) (uint, []string, error) {

	accountsWithContentRemoved := make([]string, 0)

	if len(paramData) != 1 {
		return 0, accountsWithContentRemoved, errors.New("params expects one array element")
	}

	bundle, err := messages.ParseBundle(paramData[0])
	if err != nil {
		return 0, accountsWithContentRemoved, err
	}
	newSigBundle, err := bundle.ToSigBundle()
	if err != nil {
		return 0, accountsWithContentRemoved, err
	}

	seqNo, accountsWithContentRemoved, err := sigbundle.ValidateAndSave(*newSigBundle, nil)
	if err != nil {
		return 0, accountsWithContentRemoved, err
	} else {
		return seqNo, accountsWithContentRemoved, nil
	}
}
