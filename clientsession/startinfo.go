package clientsession

import (
  "fmt"
  "log"
  "time"
  "bytes"
  "errors"
  "strings"
  "strconv"
  "encoding/hex"

  bolt "go.etcd.io/bbolt"
  "github.com/ipfs/go-cid"
  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/peer"
  "code.wip2p.com/mwadmin/wip2p-go/merklehead"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)


func (s *Session) StartInfo() (messages.PeerInfo, error) {

  c := make(chan error)

  // exchange merkleheads
  params := make([]interface{}, 0)

  // send our info
  response := messages.PeerInfo{}
  response.Merklehead = merklehead.MerkleheadAsString()
  response.RootAccount = common.BytesToAddress(globals.RootAccount[:]).String()
  response.Endpoints = globals.Endpoints
  response.Version = globals.AppName + " " + globals.AppVersion
  response.SequenceSeed = globals.SequenceSeed
  response.LatestSequenceNo = globals.LatestSequenceNo

  params = append(params, response)

  var info messages.PeerInfo

  err := s.SendRPC("peer_info", params, func(result interface{}, err error){

    // if root accounts match then all good
    // else we can either disconnect, or perhaps ask for a few wanted accounts just to see

    if err != nil {
      c <- err
      return
    }

    info, err = messages.ParsePeerInfo(result)
    if err != nil {
      c <- err
      return
    }

    remoteRootAccount := info.RootAccount
    remoteRootAccountB, err := hex.DecodeString(remoteRootAccount[2:])
    if err != nil {
      c <- err
      return
    }

    if bytes.Equal(remoteRootAccountB, globals.RootAccount[:]) == false {
      fmt.Printf("%+v %+v\n", remoteRootAccountB, globals.RootAccount[:])
      c <- errors.New("rootAccounts do not match")
      return
    }

    remoteMerklehead := info.Merklehead
    if remoteMerklehead == "" {
      s.RemoteMerklehead = make([]byte, 0)
    } else {
      remoteMerkleheadCid, err := cid.Parse(remoteMerklehead)
      if err != nil {
        c <- err
        return
      }
      s.RemoteMerklehead = make([]byte, len(remoteMerkleheadCid.Hash()))
      copy(s.RemoteMerklehead, remoteMerkleheadCid.Hash())
    }

    p := UpdatePeer(s, &info)
    // bring our lastSyncSequenceNo from Peer into session
    s.LastSyncSequenceNo = p.LastSyncSequenceNo

    c <- nil
  })

  err = <- c

  if err != nil {
    log.Println("peer_info failed")
  }

  return info, err
}

func UpdatePeer(s *Session, info *messages.PeerInfo) peer.Peer {

  found := false
  var p peer.Peer
  var tx *bolt.Tx

  if db.IsInit() {
    tx = db.GetTx(true)
    defer db.CommitTx(tx)

    p, found = peer.FetchFromDb(s.RemotePeerId, tx)
  } else {
    p = peer.Peer{}
  }
  if !found {
    // add these for a new peer
    p.PeerId = s.RemotePeerId
    p.Created = uint(time.Now().UTC().Unix())
  }

  if strings.HasPrefix(info.Endpoints, ":") {
    epWeUsed := peer.ParseEndpointString(s.RemoteEndpoint)
    port, _ := strconv.ParseUint(info.Endpoints[1:], 10, 32)
    epWeUsed.Port = uint(port)

    info.Endpoints = epWeUsed.ToURL()
  }

  if s.WeInitiated {
    p.AddEndpoint(s.RemoteEndpoint)
    p.UpdateLastConnectAttempt(s.RemoteEndpoint)
    p.MarkAsConnectable(s.RemoteEndpoint)
  }

  // update info, should reset LastSyncSequenceNo if SequenceSeed has changed
  p.UpdateInfo(*info)

  if db.IsInit() {
    p.SaveToDb(tx)
  }

  return p
}
