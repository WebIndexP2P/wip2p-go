package clientsession

import (
  //"fmt"
  //"errors"
  //"encoding/hex"
  //"github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  acct "code.wip2p.com/mwadmin/wip2p-go/account"
  "code.wip2p.com/mwadmin/wip2p-go/deltalog"
)

var cached [][2]string

func bundleGetRecent(session *Session, paramData []interface{}) ([]interface{}, error) {
    // return an array of [[account, timestamp],...]

    tx := db.GetTx(false)
    defer db.RollbackTx(tx)

    recentAddresses := deltalog.GetRecent(8, false)

    //var results []interface{}
    results := make([]interface{}, 0)

    counter := 0
    for _, reqAddress := range recentAddresses {

      account, success := acct.FetchAccountFromDb(reqAddress, tx, false)

      // skip the "remove content" delta logs because they dont have an associated account
      if success == false {
        continue
      }

      //var account acct.AccountStruct
      //json.Unmarshal(accountB, &account)

      tmpResult := []interface{}{}
      tmpResult = append(tmpResult, reqAddress.String())
      tmpResult = append(tmpResult, account.Timestamp)
      tmpResult = append(tmpResult, account.PasteSize)
      if account.PasteCount == 0 {
        tmpResult = append(tmpResult, 1)
      } else {
        tmpResult = append(tmpResult, account.PasteCount)
      }
      results = append(results, tmpResult)
      counter++
      if counter >= 8 {
        break
      }
	  }

    return results, nil
}
