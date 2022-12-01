package db

import (
  "fmt"
  "log"
  //"encoding/hex"
  bolt "go.etcd.io/bbolt"
  "github.com/ipfs/go-cid"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/db/docwithrefs"
)

type Doc struct {}

func (d *Doc) Add(multihash []byte, data []byte, tx *bolt.Tx) {

  //fmt.Printf("pastedata.go -> Add() %+v %+v\n", hex.EncodeToString(multihash), len(data))

  docWithRefs, success := d.Get(multihash, tx)

  if !success {
    docWithRefs.Data = data
    docWithRefs.Refs = 1
  } else {
    if len(docWithRefs.Data) == 0 && data != nil {
      docWithRefs.Data = data
    }
    docWithRefs.Refs++
  }

  //fmt.Printf("%+v\n", ipldStore)
  tmpBytes := docWithRefs.Marshal()
  //fmt.Printf("%+v\n", tmpBytes)

  pb := tx.Bucket([]byte("pastes"))
  err := pb.Put(multihash, tmpBytes)

  if err != nil {
    log.Fatal("problem saving doc")
  }
}

func (d *Doc) AddRef(multihash []byte, tx *bolt.Tx) {
  docWithRefs, success := d.Get(multihash, tx)

  if !success {
    panic("problem fetching doc")
  } else {
    docWithRefs.Refs++
  }

  tmpBytes := docWithRefs.Marshal()

  pb := tx.Bucket([]byte("pastes"))
  err := pb.Put(multihash, tmpBytes)

  if err != nil {
    log.Fatal("problem saving doc")
  }
}

func (d *Doc) RemoveRef(multihash []byte, tx *bolt.Tx) {
  docWithRefs, success := d.Get(multihash, tx)
  if !success {
    log.Printf("RemoveRef() Error -> missing data\n")
    return
  }

  docWithRefs.Refs--

  pb := tx.Bucket([]byte("pastes"))

  var err error
  if docWithRefs.Refs > 0 {
    tmpBytes := docWithRefs.Marshal()
    err = pb.Put(multihash, tmpBytes)

  } else {
    // remove the doc
    cid := cid.NewCidV1(0x71, multihash)
    if globals.DebugLogging {
      fmt.Printf("Deleting doc %v\n", cid.String())
    }
    err = pb.Delete(multihash)
  }

  if err != nil {
    log.Fatal("problem removing doc")
  }

}

func (d *Doc) Get(multihash []byte, tx *bolt.Tx) (docwithrefs.DocWithRefs, bool) {

  if tx == nil {
    tx = GetTx(false)
    defer RollbackTx(tx)
  }

  pb := tx.Bucket([]byte("pastes"))
  docB := pb.Get(multihash)

  //fmt.Printf("%+v\n", docB)

  if docB == nil {
    return docwithrefs.DocWithRefs{}, false
  } else {
    docWithRefs, err := docwithrefs.Unmarshal(docB)
    if err != nil {
      return docWithRefs, false
    } else {
      return docWithRefs, true
    }
  }

}
