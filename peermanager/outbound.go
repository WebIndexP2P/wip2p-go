package peermanager

import (
  "log"
  "fmt"
  "encoding/hex"
  "time"
  "sort"

  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/peer"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession"

  _ "github.com/mtibben/androiddnsfix"
)

type queueItem struct {
  PeerId string
  UnauthedEndPoint *peer.EndPoint
  NextTryTime time.Time
}

/*func (q *queueItem) GetRef() string {
  if len(q.PeerId) > 0 {
    return q.PeerId
  } else {
    return UnauthedEndPoint
  }
}*/

var peerQueue []queueItem

//connections[ws.Conn]session -> sessions.go
//authedPeers map[common.Address]bool -> authed.go

func StartOutbound() {
  fmt.Println("Starting outbound p2p connections...")

  // wait 1 second before connecting to peers
  time.Sleep(1 * time.Second)

  for !globals.StartShutDown {

    if IncomingSyncInProgress {
      time.Sleep(2 * time.Second)
      continue
    }

    if len(peerQueue) == 0 {
      time.Sleep(2 * time.Second)
      continue
    }

    sort.Slice(peerQueue, func(i, j int) bool { return peerQueue[i].NextTryTime.Before(peerQueue[j].NextTryTime) })
    //fmt.Printf("%+v\n", peerQueue)
    firstConnect := peerQueue[0]
    //fmt.Printf("%+v\n", firstConnect)

    if time.Since(firstConnect.NextTryTime).Seconds() <= 0 {
      time.Sleep(2 * time.Second)
      continue
    }

    var tmpPeer peer.Peer
    if firstConnect.PeerId == "" {
      tmpPeer = peer.Peer{}
      tmpPeer.AddEndpoint(firstConnect.UnauthedEndPoint.ToURL())
    } else {
      accountB, _ := hex.DecodeString(firstConnect.PeerId[2:])
      account := common.BytesToAddress(accountB)
      tmpPeer, _ = peer.FetchFromDb(account, nil)
    }

    peerQueue = peerQueue[1:]
    go startSession(tmpPeer)

    // if no peers connected we dont want to loop again and again without a pause
    time.Sleep(2 * time.Second)
  }
}

func startSession(peer peer.Peer) {

  if globals.DebugLogging {
    if peer.PeerId.String() == "0x0000000000000000000000000000000000000000" {
      log.Printf("startSession with %+v\n", peer.GetNextEndPoint())
    } else {
      log.Printf("startSession with %+v\n", peer.PeerId.String())
    }
  }

  session := clientsession.Create()
  session.ExpectedPeerId = peer.PeerId
  session.SetRemoteEndpoint(peer.GetNextEndPoint())

  /*session.OnAuth = func() {
    fmt.Printf("We received auth event!\n")
  }*/
  /*session.OnError = func(err error) {
    fmt.Printf("OnError: %+v\n", err.Error())
  }*/

  err := session.Dial()
  if err != nil {
    fmt.Printf("%+v\n", err)
    return
  }

  // add the connection to the core "connections" map
  clientsession.OnConnect(session)

  session.StartComms()
}

func AddPeerToQueue(p peer.Peer) {

  if len(p.Endpoints) == 0 {
    return
  }

  // add all endpoints to unique map
  for _, ep := range p.Endpoints {
    addEndpoint(ep.ToURL())
  }

  if p.PeerId.String() == "0x0000000000000000000000000000000000000000" {
    log.Fatal("peerId is empty")
  }
  qi := queueItem{PeerId: p.PeerId.String(), NextTryTime: p.GetNextTryTime()}
  //log.Printf("Adding to queue %+v\n", qi)
  peerQueue = append(peerQueue,  qi)
}

func AddEndpointToQueue(endpoint *peer.EndPoint) bool {
  wasAdded := addEndpoint(endpoint.ToURL())
  if wasAdded == false {
    return false
  }
  qi := queueItem{UnauthedEndPoint: endpoint, NextTryTime: time.Now()}
  peerQueue = append(peerQueue,  qi)
  return true
}

func AddEndpointStringToQueue(endpoint string) bool {
  newEP := peer.ParseEndpointString(endpoint)
  return AddEndpointToQueue(&newEP)
}
