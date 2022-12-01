package clientsession

import (
  //"fmt"
  "log"
  "errors"
  "encoding/json"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/account"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
  "code.wip2p.com/mwadmin/wip2p-go/deltalog"
)

func bundleGetBySequence(paramData []interface{}) ([]interface{}, error) {
    // return an array of [{account, timestamp},...]
    list := make([]messages.SequenceListItem, 0)
    //numRows := 10

    startSequenceNo, success := paramData[0].(float64)
    if !success {
      return nil, errors.New("expects valid startSequenceNo")
    }

    logItems := deltalog.GetBatch(uint64(startSequenceNo), true)

    tx := db.GetTx(false)
    defer db.RollbackTx(tx)

    for _, logItem := range logItems {

      var tmpItem messages.SequenceListItem

      if logItem.Removed {
        tmpItem = messages.SequenceListItem{SeqNo: uint(logItem.SeqNo), Account: logItem.Address.String(), Removed: true}
      } else {
        a, found := account.FetchAccountFromDb(logItem.Address, tx, false)
        if !found {
          log.Fatal("sequence account not found in db")
        }
        tmpItem = messages.SequenceListItem{SeqNo: uint(logItem.SeqNo), Account: logItem.Address.String(), Timestamp: uint(a.Timestamp)}
      }

      list = append(list, tmpItem)
    }

    var responseObj []interface{}
    responseB, _ := json.Marshal(list)
    json.Unmarshal(responseB, &responseObj)

    return responseObj, nil
}
