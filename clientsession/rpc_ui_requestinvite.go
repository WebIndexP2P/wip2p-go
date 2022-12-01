package clientsession

import (
  //"fmt"
  "log"
  "errors"
  "strings"
  "code.wip2p.com/mwadmin/wip2p-go/util"
  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/autoinvites"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
)

func ui_requestInvite(session *Session, paramData []interface{}) (interface{}, error) {
  // if no invite code, try

  input := ""
  if len(paramData) == 1 {
    var ok bool
    input, ok = paramData[0].(string)
    if !ok {
      return nil, errors.New("expects status or invite key as string")
    }
  }
  if input == "status" {
    response := map[string]interface{}{}
    nokey, key := autoinvites.GetStatus()
    response["nokey"] = nokey
    response["key"] = key
    return response, nil
  } else {
    // redeem!
    tx := db.GetTx(true)
    defer db.RollbackTx(tx)

    inviteDetails, err := autoinvites.RedeemInviteKey(input, tx)
    if err != nil {
      if strings.HasPrefix(err.Error(), "encoding/hex: ") {
        return nil, errors.New("code is invalid")
      } else {
        return nil, err
      }
    }

    // invite account
    if util.AllZero(session.RemotePeerId.Bytes()) {
      log.Fatal("remote peer id is all zeros")
    }

    sigBundle, err := autoinvites.CreateInvite(session.RemotePeerId, *inviteDetails, tx)
    if err != nil {
      return nil, err
    }

    params := []interface{}{messages.SigBundleToBundle(sigBundle)}
    OnBroadcast(session, params)

    db.CommitTx(tx)
    return "ok", nil
  }
}
