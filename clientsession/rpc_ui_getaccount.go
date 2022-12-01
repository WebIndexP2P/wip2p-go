package clientsession

import (
	//"log"
	"errors"
	//"strconv"
	"encoding/hex"
	"encoding/base64"
	"github.com/ethereum/go-ethereum/common"

	"code.wip2p.com/mwadmin/wip2p-go/db"
	"code.wip2p.com/mwadmin/wip2p-go/account"
	"code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
)

func ui_getAccount(session *Session, paramData []interface{}) (messages.AccountInfo, error){

	// construct a response
	var optionData map[string]interface{}
	var includeInvites = false
	var includePaste = false

	response := messages.AccountInfo{}

	if len(paramData) == 0 {
		return response, errors.New("expects at least one arg as account")
	}
	accountStr, ok := paramData[0].(string)
	if !ok {
		return response, errors.New("arg[1]/account expects string")
	}
	if len(paramData) == 2 {
		optionData, _ = paramData[1].(map[string]interface{})
		includePaste, _ = optionData["includePaste"].(bool)
		includeInvites, _ = optionData["includeInvites"].(bool)
	}

	accountB, _ := hex.DecodeString(accountStr[2:])
	accountCommon := common.BytesToAddress(accountB)

	tx := db.GetTx(false)
	defer db.RollbackTx(tx)

	tmpAccount, success := account.FetchAccountFromDb(accountCommon, tx, false)
	if !success {
		return response, errors.New("account not found")
	}

	response.PostCount = tmpAccount.PasteCount
	response.ActiveLevel = tmpAccount.ActiveLevel()
	response.ActiveTimestamp = uint(tmpAccount.ActiveTimestamp())

	if includePaste {
		response.Multihash = "0x" + hex.EncodeToString(tmpAccount.RootMultihash)
		response.Signature = "0x" + hex.EncodeToString(tmpAccount.Signature)
		response.Timestamp = uint(tmpAccount.Timestamp)

		doc := db.Doc{}
		docWithRefs, _ := doc.Get(tmpAccount.RootMultihash, tx)
		if len(docWithRefs.Data) > 0 {
			response.CborData = []string{base64.StdEncoding.EncodeToString(docWithRefs.Data)}
		}

	}

	if includeInvites {
		if tmpAccount.ActiveLevel() != 0 {
			response.ActiveInviter = tmpAccount.ActiveInviter.InviterAccount.String()
			// make space for inviters, then copy
			response.Inviters = make([]messages.Inviter, 0)
			for _, tmpInvite := range tmpAccount.Inviters {
				response.Inviters = append(response.Inviters, tmpInvite.ConvertToMsg())
			}
			// even though the root account may in fact have inviters, we wont show them
			// because it may encourage people to spam it with invites just for advertising
		}
		// make space for invited, then copy
		response.Invited = make([]messages.Invited, 0)
		for _, tmpInvite := range tmpAccount.Invited {
			response.Invited = append(response.Invited, tmpInvite.ConvertToMsg())
		}
	}

	return response, nil

}
