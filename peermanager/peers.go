package peermanager

import (
  "log"
  "fmt"
  "encoding/json"

  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/peer"
)

func PeersInit() {
  tx := db.GetTx(true)
  // load peers from db
  peerBucket := tx.Bucket([]byte("peers"))
  c := peerBucket.Cursor()
  counter := 0
  for k, v := c.First(); k != nil; k, v = c.Next() {
    //fmt.Printf("key=%s, value=%s\n", k, v)
    p := peer.Peer{}
    err := json.Unmarshal(v, &p)
    if err != nil {
      log.Fatal(err)
    }

    p.PeerId = common.BytesToAddress(k)

    AddPeerToQueue(p)

    counter++
  }
  db.CommitTx(tx)
  fmt.Printf("Loaded %v peer(s)\n", counter)
}

func ClearAll() {
  tx := db.GetTx(true)
  peerBucket := tx.Bucket([]byte("peers"))
  c := peerBucket.Cursor()
  for k, _ := c.First(); k != nil; k, _ = c.Next() {
    peerBucket.Delete(k)
  }
  db.CommitTx(tx)
}
