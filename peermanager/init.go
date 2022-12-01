package peermanager

import (

  //"fmt"
  "log"
  "time"
  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/peer"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession"
)

var IncomingSyncInProgress bool

func Init(cliBootstrapPeers []string) {

  // parse -boot command line args into db
  // load peers into queue
  // load bootstrap endpoints into queue

  IncomingSyncInProgress = false

  // init all the arrays
  peerQueue = make([]queueItem, 0)

  authedPeers = make(map[common.Address]bool)
  uniqueEndpointsMap = make(map[string]bool)

  ParseCliArgs(cliBootstrapPeers)

  // load all peers
  PeersInit()

  // load all bootstrap peers from db
  LoadBootstrapEndpointsFromDb()

  setCallbacks()
}

func setCallbacks() {
  clientsession.OnEnd = func(session *clientsession.Session) {

    //fmt.Println("session OnEnd")
    // either we authed and are using peerId
    // of we failed before that and update the newEndpoints[endpointUrl]

    if session.LastError != nil {
      if session.LastError.Error() == "rootAccounts do not match" {
        log.Printf("Peer %s has a different rootHash, ignoring this peer\n", session.RemotePeerId)
      } else if session.LastError.Error() == "account not authorized" {
        if session.HasAuthed {
          log.Printf("Remote peer %s responded with 'account not authorized', ignoring this peer\n", session.RemotePeerId)
        } else {
          log.Printf("Remote peer %s responded with 'account not authorized', ignoring this peer\n", session.RemoteEndpoint)
        }
      }
      return
    }

    var newQueueItem queueItem
    if session.WeInitiated {
      if session.ExpectedPeerId.String() == "0x0000000000000000000000000000000000000000" {
        ep := peer.ParseEndpointString(session.RemoteEndpoint)
        newQueueItem = queueItem{UnauthedEndPoint: &ep, NextTryTime: time.Now().Add(1 * time.Minute)}
        peerQueue = append(peerQueue, newQueueItem)
      } else {
        p, success := peer.FetchFromDb(session.ExpectedPeerId, nil)
        if !success {
          log.Fatal("init.go - peer not found")
        }
        //log.Printf("re-adding peer %s for %+v\n", session.ExpectedPeerId.String(), p.GetNextTryTime())
        newQueueItem = queueItem{PeerId: session.ExpectedPeerId.String(), NextTryTime: p.GetNextTryTime()}
        peerQueue = append(peerQueue, newQueueItem)
      }
    } else {
      if session.ExpectedPeerId.String() != "0x0000000000000000000000000000000000000000" {
        // its a new peer, might as well add it to the queue
        newQueueItem = queueItem{PeerId: session.RemotePeerId.String(), NextTryTime: time.Now().Add(1 * time.Minute)}
        peerQueue = append(peerQueue, newQueueItem)
      }
    }


    RemoveSession(session)

    if session.HasAuthed {
      RemoveAuthedPeer(session.RemotePeerId)
    }
  }

  clientsession.OnBroadcast = func(session *clientsession.Session, paramData []interface{}){
    broadcast <- broadcastStruct{session, paramData}
  }

  clientsession.OnConnect = func(session *clientsession.Session){
    AddSession(session)
  }

  clientsession.OnAuthed = func(session *clientsession.Session) bool {
    //log.Printf("init.go -> OnAuthed()")
    result := AddAuthedPeer(session.RemotePeerId)
    if result {
      for idx, qi := range peerQueue {
        if qi.PeerId == session.RemotePeerId.String() {
          ret := make([]queueItem, 0)
          ret = append(ret, peerQueue[:idx]...)
          peerQueue = append(ret, peerQueue[idx+1:]...)
          break
        }
      }
    }
    return result
  }

  clientsession.OnPeerSwap = func(session *clientsession.Session, endpoints []string) {

    if globals.DebugLogging {
      log.Printf("OnPeerSwap %+v\n", endpoints)
    }

    for _, endpoint := range endpoints {
      AddEndpointStringToQueue(endpoint)
    }
  }

}
