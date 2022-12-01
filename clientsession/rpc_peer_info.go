package clientsession

import (
  "log"
  "bytes"
  "errors"
  "encoding/hex"
  //"encoding/json"
  "github.com/ipfs/go-cid"
  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/invite"
  "code.wip2p.com/mwadmin/wip2p-go/account"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
  "code.wip2p.com/mwadmin/wip2p-go/merklehead"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)

func peerInfo(session *Session, paramData []interface{}) (messages.PeerInfo, error) {

  //response := map[string]interface{}{}
  var info messages.PeerInfo

  // if there is paramData, process it
  if len(paramData) == 1 {
    info, _ = messages.ParsePeerInfo(paramData[0])

    if len(info.RootAccount) < 42 {
      return info, errors.New("invalid root account")
    }

    remoteRootAccountB, err := hex.DecodeString(info.RootAccount[2:])
    if err != nil {
      return info, errors.New("invalid root account")
    }

    // update our rootAccount if needed
    if globals.GetRootFromNextPeer {
      copy(globals.RootAccount[:], remoteRootAccountB)
      tx := db.GetTx(true)
      doc := db.Config{}
      doc.Write("rootAccount", globals.RootAccount[:], tx)

      inviters := make([]invite.Inviter, 0)
      inviters = append(inviters, invite.Inviter{Timestamp: uint64(1)})
      rootAccount := account.AccountStruct{Address: common.BytesToAddress(globals.RootAccount[:]), Inviters: inviters, ActiveInviter: inviters[0], Enabled: true}
      rootAccount.SaveToDb(tx)

      globals.GetRootFromNextPeer = false;
      conf := db.Config{}
      conf.Delete("getRootFromNextPeer", tx)

      db.CommitTx(tx)
      log.Printf("rootAccount set to 0x%v\n", hex.EncodeToString(globals.RootAccount[:]))
    }

    if bytes.Equal(remoteRootAccountB, globals.RootAccount[:]) == false {
      return info, errors.New("rootAccounts do not match")
    }

    remoteMerklehead := info.Merklehead
    if remoteMerklehead == "" {
      session.RemoteMerklehead = make([]byte, 0)
    } else {
      remoteMerkleheadCid, err := cid.Parse(remoteMerklehead)
      if err != nil {
        return info, err
      }
      session.RemoteMerklehead = make([]byte, len(remoteMerkleheadCid.Hash()))
      copy(session.RemoteMerklehead, remoteMerkleheadCid.Hash())
    }


    p := UpdatePeer(session, &info)
    // bring our lastSyncSequenceNo from Peer into session
    session.LastSyncSequenceNo = p.LastSyncSequenceNo
  }

  // dont send restricted our info
  if (session.LocalRestricted && globals.PublicMode == false) {
    return info, nil
  }

  // create a new info struct for our response
  info = messages.PeerInfo{}
  info.Merklehead = merklehead.MerkleheadAsString()
  info.RootAccount = common.BytesToAddress(globals.RootAccount[:]).String()
  info.Endpoints = globals.Endpoints
  info.Version = "wip2p-go " + globals.AppVersion
  info.SequenceSeed = globals.SequenceSeed
  info.LatestSequenceNo = globals.LatestSequenceNo

  //var responseObj map[string]interface{}
  //responseB, _ := json.Marshal(info)
  //json.Unmarshal(responseB, &responseObj)

  return info, nil
}
