package clientsession

import (
  //"fmt"
  "log"
  "bytes"
  "errors"
  "encoding/hex"

  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/sigbundle/sigbundlestruct"
  "code.wip2p.com/mwadmin/wip2p-go/merklehead"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
)

// we only want one incoming sync process at any given time
func (s *Session) StartSync() {
  if globals.DebugLogging {
    log.Printf("Starting sync session\n")
  }

  if len(s.RemoteMerklehead) == 0 {
    if globals.DebugLogging {
      log.Printf("remote has no content")
    }
  } else if bytes.Equal(merklehead.Merklehead, s.RemoteMerklehead) {
    log.Printf("already synced\n")
    s.HasSynced = true
    return
  } else {
    if globals.DebugLogging {
      log.Printf("StartSwap <- downloading new content from %+v\n", s.RemotePeerId.String())
    }

    s.StartSequenceSwap()

    // check again
    if bytes.Equal(merklehead.Merklehead, s.RemoteMerklehead) {
      log.Printf("fully synced\n")
      s.HasSynced = true
    }
  }

  response := map[string]interface{}{}
  response["merklehead"] = merklehead.MerkleheadAsString()
  params := []interface{}{response}

  // send a syncFinished message so remote begins their sync if required
  if globals.DebugLogging {
    log.Printf("peer_syncFinished -> %+v\n", s.RemotePeerId.String())
  }
  s.SendRPC("peer_syncFinished", params, func(result interface{}, err error){
    // wait for response, then start linkeddocsswap
    if s.WeInitiated == false {
      go s.StartLinkedDocsSwap()
    }
  })
}

func (s *Session) BundleToPasteSaveParam(account []byte, bundle map[string]interface{}) sigbundlestruct.SigBundle {

  if bundle == nil {
    panic("bundle should not be nil")
  }

  tmpSigbundle := sigbundlestruct.SigBundle{}

  // add account
  tmpSigbundle.Account = account
  sigB, _ := hex.DecodeString(bundle["signature"].(string)[2:])
  tmpSigbundle.Signature = make([]byte, len(sigB))
  copy(tmpSigbundle.Signature, sigB)
  tmpSigbundle.Timestamp = uint64(bundle["timestamp"].(float64))

  // convert cborData to base64 and array
  cborData := bundle["cborData"].(string)
  cborDataB, _ := hex.DecodeString(cborData[2:])
  tmpSigbundle.CborData = make([][]byte, 1)
  tmpSigbundle.CborData[0] = make([]byte, len(cborDataB))
  copy(tmpSigbundle.CborData[0], cborDataB)

  // convert root multihash
  mhB, _ := hex.DecodeString(bundle["multihash"].(string)[2:])
  tmpSigbundle.RootMultihash = make([]byte, len(mhB))
  copy(tmpSigbundle.RootMultihash, mhB)

  return tmpSigbundle
}

func (s *Session) RequestSignedBundleFromPeer(account []byte) (*messages.Bundle, error) {
  acct := common.BytesToAddress(account)
  waitchan := make(chan error)
  paramObj := map[string]interface{}{
    "account": acct.String(),
  }
  var bundle *messages.Bundle
  s.SendRPC("bundle_get", []interface{}{paramObj}, func(result interface{}, err error){
    if err != nil {
      waitchan <- err
      return
    }
    bundle, err = messages.ParseBundle(result)
    if err != nil {
      waitchan <- errors.New("remote has responded with an invalid bundle")
      return
    } else {
      waitchan <- nil
    }
  })
  err := <- waitchan
  return bundle, err
}
