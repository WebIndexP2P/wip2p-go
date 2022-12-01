package merklehead

import (
  "log"
  "fmt"
  //"errors"
  //"strings"
  //"strconv"
  bolt "go.etcd.io/bbolt"
  "github.com/ipfs/go-cid"
  "github.com/ipfs/go-ipld-cbor"
  mh "github.com/multiformats/go-multihash"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  //"code.wip2p.com/mwadmin/wip2p-go/ipldwalk"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)

var Merklehead []byte // this is a multihash of an IpldCborNode

func Init() {
  config := db.Config{}
  mhead := config.Read("merklehead", nil)

  if len(mhead) > 0 {
    Merklehead = make([]byte, len(mhead))
    copy(Merklehead, mhead)

    tmpCid := cid.NewCidV1(0x71, Merklehead)
    fmt.Printf("Merklehead: %v\n", tmpCid)
    if globals.AndroidCallback != nil {
      globals.AndroidCallback("merklehead: " + MerkleheadAsString())
    }
  } else {
    fmt.Printf("Merklehead not yet set\n")
    if globals.AndroidCallback != nil {
      globals.AndroidCallback("merklehead not yet set")
    }
  }
}

func MerkleheadAsString() string {
  if Merklehead != nil {
    return cid.NewCidV1(0x71, Merklehead).String()
  } else {
    return ""
  }
}

func Add(address string, timestamp uint64, tx *bolt.Tx) error {

  if globals.DebugLogging {
    fmt.Printf("merklehead.Add() address %v\n", address)
  }

  if address[:2] == "0x" {
    log.Fatal("address expects no 0x prefix")
  }

  var newCid *cid.Cid
  var err error

  if Merklehead == nil {
    rootDoc := make(map[string]interface{})
    rootDoc[address] = timestamp
    newCid = saveDoc(rootDoc, tx)
  } else {
    newCid, err = addItemRecursive(Merklehead, address, timestamp, tx)
    if err != nil {
      return err
    }
  }

  if newCid == nil {
    log.Fatal("missing new Cid")
  }

  // update the Merklehead global and conf
  Merklehead = make([]byte, len(newCid.Hash()))
  copy(Merklehead, newCid.Hash())

  config := db.Config{}
  config.Write("merklehead", Merklehead, tx)
  if globals.DebugLogging {
    fmt.Printf("Merklehead is now %v\n", MerkleheadAsString())
  }
  if globals.AndroidCallback != nil {
    globals.AndroidCallback("merklehead is now " + MerkleheadAsString())
  }

  return nil
}

func Remove(address string, tx *bolt.Tx) error {

  if globals.DebugLogging {
    fmt.Printf("merklehead.Remove() address %v\n", address)
  }

  if address[:2] == "0x" {
    log.Fatal("address expects no 0x prefix")
  }

  var newCid *cid.Cid
  var err error

  if Merklehead == nil {
    log.Fatal("Cannot reduce merklehead to nil")
  } else {
    newCid, _, _, err = removeItemRecursive(Merklehead, address, tx)
    if err != nil {
      return err
    }
  }

  if newCid == nil {
    log.Printf("missing new Cid\n")
    return nil
  }

  // update the Merklehead global and conf
  Merklehead = make([]byte, len(newCid.Hash()))
  copy(Merklehead, newCid.Hash())

  config := db.Config{}
  config.Write("merklehead", Merklehead, tx)
  if globals.DebugLogging {
    fmt.Printf("Merklehead is now %v\n", MerkleheadAsString())
  }

  return nil
}

func saveDoc(doc map[string]interface{}, tx *bolt.Tx) *cid.Cid {
  docNode, err := cbornode.WrapObject(doc, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
  if err != nil {
    log.Fatal(err)
  }

  // adds doc to doc store
  libDoc := db.Doc{}
  libDoc.Add(docNode.Cid().Hash(), docNode.RawData(), tx)

  newCid := docNode.Cid()
  return &newCid
}

func FetchDoc(cidBytes []byte, tx *bolt.Tx) map[string]interface{} {
  doc := db.Doc{}
  docWithRefs, success := doc.Get(cidBytes, tx)
  if !success {
    log.Fatal("doc is missing ", cid.NewCidV1(0x71, cidBytes))
  }
  rootDocNode, err := cbornode.Decode(docWithRefs.Data, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
  if err != nil {
    log.Fatal(err)
  }
  rootDocIFace, _, _ := rootDocNode.Resolve(nil)
  rootDoc := rootDocIFace.(map[string]interface{})
  return rootDoc
}
