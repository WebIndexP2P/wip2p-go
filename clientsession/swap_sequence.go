package clientsession

import (
  "log"
  "bytes"
  "encoding/json"

  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/account"
  "code.wip2p.com/mwadmin/wip2p-go/sigbundle"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
)

func (s *Session) StartSequenceSwap() {
  curSeqNo := s.LastSyncSequenceNo
  if curSeqNo == 0 {
    curSeqNo = 1
  }

  bRunning := true
  for bRunning {
    lastSeqNo := s.doSequenceSwapBatch(curSeqNo)
    if lastSeqNo == curSeqNo {
      break;
    } else {
      curSeqNo = lastSeqNo + 1
    }
  }
  //log.Printf("Sync finished\n")
}

func (s *Session) doSequenceSwapBatch(startSequenceId uint) uint {

  lastSeqNo := startSequenceId

  //log.Printf("doSequenceSwapBatch %v\n", startSequenceId)

  list, err := s.RequestNextSequenceFromPeer(startSequenceId)
  if err != nil {
    log.Println("bundle_getBySequence failed")
  }

  // process list
  //log.Printf("%+v\n", list)
  for _, sequenceItem := range list {

    lastSeqNo = sequenceItem.SeqNo
    if sequenceItem.Removed {
      continue
    }

    lookupAddress := common.HexToAddress(sequenceItem.Account)

    success := s.SaveAccountPlusMissingInviters(lookupAddress, sequenceItem.Timestamp)
    if !success {
      log.Printf("error in sync, account provided %s has no invite\n", lookupAddress)
      break // break for debugging purposes
    }
  }
  return lastSeqNo
}

func (s *Session) RequestNextSequenceFromPeer(startSequence uint) ([]messages.SequenceListItem, error) {

  list := make([]messages.SequenceListItem, 0)

  waitchan := make(chan error)
  s.SendRPC("bundle_getBySequence", []interface{}{startSequence}, func(result interface{}, err error){
    if err != nil {
      log.Printf("%+v\n", err)
      waitchan <- err
    }

    resultB, _ := json.Marshal(result)
    json.Unmarshal(resultB, &list)

    waitchan <- nil
  })

  err := <- waitchan
  return list, err
}

func (s *Session) RequestBundleFromPeer(account common.Address) (*messages.Bundle, error) {

  var bundle *messages.Bundle
  waitchan := make(chan error)
  s.SendRPC("bundle_get", []interface{}{map[string]interface{}{"account": account}}, func(result interface{}, err error){
    if err != nil {
      log.Printf("%+v\n", err)
      waitchan <- err
    }
    bundle, err = messages.ParseBundle(result)
    waitchan <- err
  })
  err := <- waitchan
  return bundle, err
}

func (s *Session) SaveAccountPlusMissingInviters(lookupAddress common.Address, timestamp uint) (success bool) {

  //log.Printf("---> SaveAccountPlusMissingInviters %v\n", lookupAddress.String())
  //defer log.Printf("<--- SaveAccountPlusMissingInviters %v\n", lookupAddress.String())

  // first check if the account exists, and check timestamp
  localCopyOfAccount, found := account.FetchAccountFromDb(lookupAddress, nil, false)
  updateExistingAccount := false
  if found {
    // compare timestamps
    if timestamp <= uint(localCopyOfAccount.Timestamp) {
      //log.Printf("Account already exists and is up to date\n")
      return true
    } else {
      updateExistingAccount = true
    }
  }

  // we want the bundle and the invite details
  var err error
  accountInfoFromPeer, err := s.GetAccountFromPeer(lookupAddress)
  if err != nil {
    return false
  }

  if updateExistingAccount == false {
    // get Inviter from invite data
    inviterAcctHex := accountInfoFromPeer.ActiveInviter
    inviterAcct := common.HexToAddress(inviterAcctHex)
    var activeInviteAccount common.Address
    var activeInviteTimestamp uint
    bFound := false
    // find the invite in the list
    for _, invite := range accountInfoFromPeer.Inviters {
      tmpAccount := common.HexToAddress(invite.Account)
      if bytes.Equal(tmpAccount.Bytes(), inviterAcct.Bytes()) {
        activeInviteAccount = tmpAccount
        activeInviteTimestamp = invite.Timestamp
        bFound = true
        break
      }
    }
    if !bFound {
      log.Printf("active inviter details not found, skipping\n")
      return false
    }

    tmpSuccess := s.SaveAccountPlusMissingInviters(activeInviteAccount, activeInviteTimestamp)
    if !tmpSuccess {
      return false
    }
  }

  sb, err := accountInfoFromPeer.ToSigBundle(lookupAddress.String())
  if err != nil {
    log.Printf("YYYYY: %+v\n", err)
    return false
  }

  sb.Account = lookupAddress.Bytes()
  _, _, err = sigbundle.ValidateAndSave(*sb, nil)
  if err != nil {
    log.Printf("%+v\n", err)
    return false
  }

  return true
}

func (s *Session) GetAccountFromPeer(address common.Address) (messages.AccountInfo, error) {
  accountInfo := messages.AccountInfo{}

  waitchan := make(chan error)
  params := []interface{}{address.String(), map[string]interface{}{"includeInvites": true, "includePaste": true}}
  s.SendRPC("ui_getAccount", params, func(result interface{}, err error){
    if err != nil {
      log.Printf("%+v\n", err)
      waitchan <- err
      return
    }

    //log.Printf("getAccount result = %+v\n", result)
    accountInfo, err = messages.ParseAccountInfo(result)
    if err != nil {
      waitchan <- err
      return
    }

    // check account exists
    waitchan <- nil
  })
  err := <- waitchan

  return accountInfo, err
}
