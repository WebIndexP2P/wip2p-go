package clientsession

import (
  "log"

  "github.com/ethereum/go-ethereum/crypto"
  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/account"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
  "code.wip2p.com/mwadmin/wip2p-go/sigbundle/sigbundlestruct"
)

func (s *Session) StartRecovery() error {

  bundles := make([]sigbundlestruct.SigBundle, 0)

  if globals.PublicMode {

    _, err := s.StartInfo()
    if err != nil && err.Error() == "rootAccounts do not match" {
      s.Disconnect()
      return nil
    }

    addr := common.BytesToAddress(globals.RootAccount[:])
    targetAccount, _ := account.FetchAccountFromDb(addr, nil, false)
    bundle, err := targetAccount.ExportSigBundle()
    if err != nil {
      log.Fatal(err)
    }

    bundle.Account = globals.RootAccount[:]
    postBundle(s, bundle)
    return nil
  }

  // get our node address
  targetAccountAddress := crypto.PubkeyToAddress(globals.NodePrivateKey.PublicKey)

  // loop upwards through inviters and send all bundles to remote peer
  var ourAccountLoaded = false

  for true {

    //log.Printf("Target address %+v\n", targetAccountAddress)
    targetAccount, _ := account.FetchAccountFromDb(targetAccountAddress, nil, false)
    //log.Printf("Target %+v\n", targetAccount)

    // no need to send our bundle
    if ourAccountLoaded == false {
      ourAccountLoaded = true
      targetAccountAddress = targetAccount.ActiveInviter.InviterAccount
      continue
    }

    bundle, err := targetAccount.ExportSigBundle()
    if err != nil {
      log.Fatal(err)
    }

    bundle.Account = targetAccountAddress.Bytes()

    bundles = append(bundles, bundle)

    if targetAccount.ActiveLevel() == 0 {
      break
    }
  }

  //send peer_info so the remote has our rootAccount and merklehead
  s.StartInfo()

  for a := len(bundles) - 1; a >= 0; a-- {
    postBundle(s, bundles[a])
  }

  return nil
}


func postBundle(s *Session, bundle sigbundlestruct.SigBundle) {

  c := make(chan error)
  bundleMsg := messages.SigBundleToBundle(&bundle)

  err := s.SendRPC("bundle_save", []interface{}{bundleMsg}, func(result interface{}, err error){
    c <- nil
  })

  err = <- c
  if err != nil {
    log.Println("peer_recovery failed")
  }
}
