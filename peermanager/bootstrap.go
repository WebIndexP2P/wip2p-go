package peermanager

import (
  "fmt"
  "os"
  "encoding/binary"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/peer"
)

func ParseCliArgs(cliBootstrapPeers []string) {
  // process any command line bootstrap peers
  bpdb := db.BootstrapPeer{}
  tx := db.GetTx(true)

  if len(cliBootstrapPeers) == 1 && cliBootstrapPeers[0] == "clear" {
    // remove all the bootstrap peers in the db
    fmt.Printf("clearing all bootstrap endpoints from db\n")
    bootBucket := tx.Bucket([]byte("bootstrap"))
    c := bootBucket.Cursor()
    for k, _ := c.First(); k != nil; k, _ = c.Next() {
      bootBucket.Delete(k)
    }
    db.CommitTx(tx)
    os.Exit(0)
  } else {
    for _, bp := range cliBootstrapPeers {
      bpdb.Add(bp, tx)
    }
  }

  db.CommitTx(tx)
}

func LoadBootstrapEndpointsFromDb() {
  tx := db.GetTx(true)
  bootBucket := tx.Bucket([]byte("bootstrap"))
  counter := 0
  c := bootBucket.Cursor()
  for k, v := c.First(); k != nil; k, v = c.Next() {
    endpoint := string(k)
    v = v

    // check if endpoint already exists
    if uniqueEndpointExists(endpoint) {
      continue
    }

    failedAttempts := uint(binary.LittleEndian.Uint32(v))
    newBEP := peer.ParseEndpointString(endpoint)
    newBEP.ConsecutiveConnectFails = failedAttempts

    AddEndpointToQueue(&newBEP)
    counter++
  }

  if counter > 0 {
    fmt.Printf("Loaded %v additional bootstrap endpoint(s)\n", counter)
  }
  db.CommitTx(tx)
}
