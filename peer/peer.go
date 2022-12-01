package peer

import (
  //"fmt"
  "log"
  "time"
  "strings"
  "encoding/json"

  bolt "go.etcd.io/bbolt"
  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
)

type Peer struct {
  PeerId common.Address `json:"-"`
  Created uint

  Endpoints []EndPoint
  RootAccount common.Address
  Merklehead string
  Version string

  SequenceSeed uint
  LatestSequenceNo uint
  LastComms uint

  LastSyncSequenceNo uint // our local position with this peer
}

//var peerList []Peer
//var peerMap map[[20]byte]uint

func (p *Peer) AddEndpoint(endpoint string) {

  newEP := ParseEndpointString(endpoint)

  for _, ep := range p.Endpoints {
    if ep.Equals(&newEP) {
      return
    }
  }

  p.Endpoints = append(p.Endpoints, newEP)
}

func (p *Peer) UpdateLastConnectAttempt(endpoint string) {
  ep := p.GetEndPoint(endpoint)
  ep.LastConnectAttempt = time.Now()
}

func (p *Peer) MarkAsConnectable(endpoint string) {
  ep := p.GetEndPoint(endpoint)
  ep.IsReachable = true
  ep.ConsecutiveConnectFails = 0
}

func (p *Peer) EndpointFailure(endpoint string) {
  ep := p.GetEndPoint(endpoint)
  ep.ConsecutiveConnectFails++
}

func (p *Peer) UpdateInfo(info messages.PeerInfo) {

  if p.SequenceSeed != 0 && info.SequenceSeed != p.SequenceSeed {
    log.Printf("Peer %v has reset their sequence seed!\n", p.PeerId.String())
    p.LastSyncSequenceNo = 0
  }

  p.Merklehead = info.Merklehead
  p.RootAccount = common.HexToAddress(info.RootAccount)
  p.Version = info.Version
  p.LastComms = uint(time.Now().UTC().Unix())
  p.SequenceSeed = info.SequenceSeed
  p.LatestSequenceNo = info.LatestSequenceNo

  endpoints := strings.Split(info.Endpoints, ",")
  for _, ep := range endpoints {
    if ep == "" {
      continue
    }
    p.AddEndpoint(ep)
  }

}

func (p *Peer) GetEndPoint(targetEP string) *EndPoint {
  //log.Printf("GetEndPoint " + targetEP)
  for idx, ep := range p.Endpoints {
    if ep.ToURL() == targetEP {
      return &p.Endpoints[idx]
    }
  }
  return nil
}

func (p *Peer) GetNextTryTime() time.Time {
  var minNextTryTime time.Time
  var tmpNextTryTime time.Time
  for idx, ep := range p.Endpoints {
    //log.Printf("%+v\n", ep)
    if idx == 0 {
      minNextTryTime = ep.GetNextTryTime()
    } else {
      tmpNextTryTime = ep.GetNextTryTime()
      if tmpNextTryTime.Before(minNextTryTime) {
        minNextTryTime = tmpNextTryTime
      }
    }
  }
  return minNextTryTime
}

func (p *Peer) GetNextEndPoint() string {
  var minNextTryTime time.Time
  var nextEndpoint *EndPoint
  for idx, ep := range p.Endpoints {
    tmpNextTryTime := ep.GetNextTryTime()
    if nextEndpoint == nil || tmpNextTryTime.Before(minNextTryTime) {
      minNextTryTime = tmpNextTryTime
      nextEndpoint = &p.Endpoints[idx]
    }
  }
  return nextEndpoint.ToURL()

}

func (p *Peer) GetReachableEndpointsForSegment(segment string) []string {
  ret := make([]string, 0)
  for _, ep := range p.Endpoints {
    if ep.IsReachable == false {
      continue
    }
    if segment == "private" {
      ret = append(ret, ep.ToURL())
    } else if segment == "public" && ep.NetworkSegment == "public" {
      ret = append(ret, ep.ToURL())
    }
  }
  return ret
}

func FetchFromDb(peerId common.Address, tx *bolt.Tx) (Peer, bool) {

	//log.Printf("Peer.FetchFromDb - %+v\n", peerId.String())

  p := Peer{}

	if tx == nil {
		tx = db.GetTx(false)
		defer db.RollbackTx(tx)
	}

	ab := tx.Bucket([]byte("peers"))
	peerB := ab.Get(peerId.Bytes())

	if peerB == nil {
		return p, false
	}

	json.Unmarshal(peerB, &p)
	p.PeerId = peerId

	return p, true
}

func (p *Peer) SaveToDb(tx *bolt.Tx) bool {

  if globals.DebugLogging {
    log.Printf("Saving Peer %+v to db\n", p.PeerId.String())
  }

  if p.PeerId.String() == "0x0000000000000000000000000000000000000000" {
    log.Fatal("peer.go - SaveToDb() - PeerId is null")
  }

  if tx == nil {
    tx = db.GetTx(true)
    defer db.CommitTx(tx)
  }

  pb := tx.Bucket([]byte("peers"))
  peerB, err := json.Marshal(p)
  if err != nil {
    log.Fatal(err)
  }
  pb.Put(p.PeerId.Bytes(), peerB)

  return true
}
