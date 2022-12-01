package clientsession

import (
  "log"
  "time"
  "bytes"
  "errors"
  "github.com/ipfs/go-cid"

  "code.wip2p.com/mwadmin/wip2p-go/merklehead"
)

func peerSyncFinished(session *Session, paramData []interface{}) (interface{}, error) {

  paramObj, ok := paramData[0].(map[string]interface{})
  if !ok {
    return nil, errors.New("params expects object at [0]")
  }

  // fetch new merklehead
  remoteMerklehead := paramObj["merklehead"].(string)
  if remoteMerklehead == "" {
    session.RemoteMerklehead = make([]byte, 0)
  } else {
    remoteMerkleheadCid, err := cid.Parse(remoteMerklehead)
    if err != nil {
      return nil, err
    }
    session.RemoteMerklehead = make([]byte, len(remoteMerkleheadCid.Hash()))
    copy(session.RemoteMerklehead, remoteMerkleheadCid.Hash())
  }

  if session.WeInitiated {
    if bytes.Equal(merklehead.Merklehead, session.RemoteMerklehead) {
      log.Printf("fully synced\n")
      session.HasSynced = true

      go func(){
        time.Sleep(1 * time.Second)
        session.StartLinkedDocsSwap()
      }()

    } else {
      log.Printf("merkleheads still dont match, consensus bug!!\n")
    }

    return "ok", nil

  } else {
    // remote has completed the sync, now its our turn
    go session.StartSync()
  }

  return "ok", nil
}
