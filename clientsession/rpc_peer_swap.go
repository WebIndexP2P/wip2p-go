package clientsession

import (
  "log"
  "strings"
  "encoding/json"

  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/peer"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
)

func peerSwap(session *Session, paramData []interface{}) (interface{}, error) {

  secureOnly := false

  if len(paramData) > 0 {
    paramIface, _ := paramData[0].(map[string]interface{})

    // get endpoints
    endpointsIface, _ := paramIface["endpoints"]
    incomingEndpoints, err := messages.ParsePeerSwap(endpointsIface)
    if err == nil {
      OnPeerSwap(session, incomingEndpoints)
    }

    // check for secure flag
    secureOnlyIface, success := paramIface["secureOnly"]
    if success {
      secureOnly = secureOnlyIface.(bool)
    }
  }

  tx := db.GetTx(false)
  defer db.RollbackTx(tx)

  endpoints := make([]string, 0)

  // load peers from db
  peerBucket := tx.Bucket([]byte("peers"))
  cursor := peerBucket.Cursor()
  for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
    p := peer.Peer{}
    err := json.Unmarshal(v, &p)
    if err != nil {
      log.Fatal(err)
    }

    peerId := common.BytesToAddress(k)

    if peerId.String() == session.RemotePeerId.String() {
      continue
    }

    endpoints = append(endpoints, p.GetReachableEndpointsForSegment(session.NetworkSegment)...)
  }

  if secureOnly {
    secureEndpoints := make([]string, 0)
    for _, ep := range endpoints {
      if strings.HasPrefix(ep, "wss://") {
        secureEndpoints = append(secureEndpoints, ep)
      }
    }
    endpoints = secureEndpoints
  }

  response := map[string]interface{}{}
  if len(endpoints) > 0 {
    response["endpoints"] = endpoints
  }

  return response, nil
}
