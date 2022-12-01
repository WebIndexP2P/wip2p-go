package clientsession

import (
  //"fmt"
  "errors"
  "encoding/hex"
  "encoding/base64"

  bolt "go.etcd.io/bbolt"
  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
  acct "code.wip2p.com/mwadmin/wip2p-go/account"
)

func bundleGet(paramData []interface{}, tx *bolt.Tx) (messages.Bundle, error) {

  var response messages.Bundle

  if len(paramData) == 0 {
    return response, errors.New("expects params array data")
  }

  paramObj, ok := paramData[0].(map[string]interface{})
  if !ok {
    return response, errors.New("expects object with 'account'")
  }

  requestedAccountIface, success := paramObj["account"]
  if !success {
    return response, errors.New("missing account param")
  }
  requestedAccount, success := requestedAccountIface.(string)
  if !success {
    return response, errors.New("account param expects string")
  }

  if len(requestedAccount) != 42 {
    return response, errors.New("account expects length 42")
  }

  reqAccountB, _ := hex.DecodeString(requestedAccount[2:])
  reqAddress := common.BytesToAddress(reqAccountB)

  if tx == nil {
    tx = db.GetTx(false)
    defer db.RollbackTx(tx)
  }

  account, success := acct.FetchAccountFromDb(reqAddress, tx, false)
  if !success {
    return response, errors.New("account not found")
  }

  if len(account.RootMultihash) > 0 {
    response.Multihash = "0x" + hex.EncodeToString(account.RootMultihash)
  } else {
    return response, errors.New("account has not posted anything")
  }

  if len(account.Signature) > 0 {
    response.Signature = "0x" + hex.EncodeToString(account.Signature)
  }
  response.Timestamp = account.Timestamp

  doc := db.Doc{}
  docWithRefs, _ := doc.Get(account.RootMultihash, tx)

  if len(docWithRefs.Data) > 0 {
    response.CborData = make([]string, 0)
    response.CborData = append(response.CborData, base64.StdEncoding.EncodeToString(docWithRefs.Data))
  }

  return response, nil
}
