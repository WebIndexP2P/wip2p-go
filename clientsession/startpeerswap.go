package clientsession

import (
  "log"
  "encoding/json"

  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/peer"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
)

func (s *Session) StartPeerSwap() error {

  c := make(chan error)
  response := make([]string, 0)

  // iterate peers
  tx := db.GetTx(false)
  defer tx.Rollback()

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

    if peerId.String() == s.RemotePeerId.String() {
      continue
    }

    response = append(response, p.GetReachableEndpointsForSegment(s.NetworkSegment)...)
  }

  params := make([]interface{}, 0)

  // convert all strings into interface for some go reason?
  if len(response) > 0 {
    paramData := make(map[string]interface{}, 0)
    endpointParam := make([]interface{}, len(response))
    for idx, endpoint := range response {
      endpointParam[idx] = endpoint
    }
    paramData["endpoints"] = endpointParam
    params = append(params, paramData)
  }

  // test secureOnly
  //paramData := make(map[string]interface{}, 0)
  //paramData["secureOnly"] = true
  //params = append(params, paramData)

  err := s.SendRPC("peer_swap", params, func(result interface{}, err error){
    params, success := result.(map[string]interface{})
    if !success {
      c <- nil
      return
    }
    endpointsIface, success := params["endpoints"]
    if !success {
      c <- nil
      return
    }

    endpoints, err := messages.ParsePeerSwap(endpointsIface)
    if err == nil {
      OnPeerSwap(s, endpoints)
    }

    c <- err
  })

  err = <- c

  if err != nil {
    log.Println("peer_swap failed")
  }

  return err
}
