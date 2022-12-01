package ipldwalk

import (
  "fmt"
  "log"
  "bytes"
  "strings"
  "encoding/hex"
  bolt "go.etcd.io/bbolt"
  "github.com/ipfs/go-cid"
  "github.com/ipfs/go-ipld-format"
  "github.com/ipfs/go-ipld-cbor"
  mh "github.com/multiformats/go-multihash"

  "github.com/ipld/go-ipld-prime/node/basicnode"
  "github.com/ipld/go-ipld-prime/codec/dagcbor"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)

func UpdateRoot(dataset map[string][]byte, newCid cid.Cid, origMultihash []byte, tx *bolt.Tx) (docsAdded int, totalSize int, docsNotFound int) {

  //log.Printf("UpdateRoot %+v %+v\n", newCid, origMultihash)
  //log.Printf("dataset length: %+v\n", len(dataset))

  // only run this if the account didn't already have this root_multihash
  newMultihash := newCid.Hash()

  if bytes.Equal(newMultihash, origMultihash) {
    log.Fatal("cannot add same multihash")
  }

  // descend through ipld tree comparing the same path
  // if root doc has wip2p/i data then import it
  // if root doc has linked wip2p data
  //  load original root doc
  //   if the linked multihash changed then attempt to load from the provided dataset, or the existing datastore
  //
  //   recursive remove old link

  newMhHex := hex.EncodeToString(newMultihash)
  doc := db.Doc{}

  /*fmt.Printf("%+v\n", newMhHex)
  for key, data := range dataset {
    fmt.Printf("%+v %+v\n", key, len(data))
  }*/

  // if its DagCBor, just try decode it first, it might be DagProtobuf
  var nd *cbornode.Node
  if newCid.Type() == 0x71 {
    np := basicnode.Prototype.Any
    nb := np.NewBuilder()
    r := bytes.NewReader(dataset[newMhHex])
    err := dagcbor.Decode(nb, r)
    if err != nil {
      newCid = cid.NewCidV1(cid.DagProtobuf, newMultihash)
    } else {
      nd, err = cbornode.Decode(dataset[newMhHex], mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
      if err != nil {
        log.Fatal("ipldwalk.go -> UpdateRoot()/1 %+v\n", err)
      }
    }
  }

  if newCid.Type() == 0x70 || newCid.Type() == 0x55 {
    docsAdded, totalSize, docsNotFound = WalkAndSaveUnixFS(dataset, newCid, tx)

    docsAdded = docsAdded
    totalSize = totalSize
    docsNotFound = docsNotFound
    return docsAdded, totalSize, docsNotFound
  }

  newLinks := nd.Links()

  oldLinks := make([]*format.Link, 0)
  if origMultihash != nil {
    // fetch the orig doc, and get unique links
    docWithRefs, success := doc.Get(origMultihash, tx)
    if !success {
      panic("multihash not found in db.")
    }
    var err error
    nd, err = cbornode.Decode(docWithRefs.Data, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
    if err != nil {
      log.Fatal("ipldwalk.go -> UpdateRoot()/2 %+v\n", err)
    }
    oldLinks = nd.Links()
  }

  // find unique links
  uniqueLinksMap := make(map[string]cid.Cid)
  for _, newlink := range(newLinks) {
    hashHex := hex.EncodeToString(newlink.Cid.Hash())
    if _, success := uniqueLinksMap[hashHex]; !success {
      uniqueLinksMap[hashHex] = newlink.Cid
    }
  }

  // find removed links
  oldLinksMap := make(map[string]interface{})
  for _, oldlink := range(oldLinks) {
    bFound := false
    for _, newlink := range(newLinks) {
      if bytes.Equal(newlink.Cid.Hash(), oldlink.Cid.Hash()) {
        bFound = true
        break
      }
    }
    if !bFound {
      hashHex := hex.EncodeToString(oldlink.Cid.Hash())
      oldLinksMap[ hashHex ] = oldlink.Cid.Type()
    }
  }

  // add new links
  for hashHex, tmpCid := range(uniqueLinksMap) {
    cidType := tmpCid.Type()
    multihash, _ := hex.DecodeString(hashHex)
    if cidType == 113 { //dag-cbor
      docs, size := RecursiveAdd(dataset, multihash, tx)
      docsAdded += docs
      totalSize += size
    } else {
      //fmt.Printf("Adding %+v\n", hashHex)
      if _, success := dataset[hashHex]; !success {
        docWithRefs, success := doc.Get(multihash, tx)
        if !success || docWithRefs.Data == nil {
          log.Printf("could not find linked doc\n")
          docsNotFound++
          doc.Add(multihash, nil, tx)
          continue
        } else {
          dataset[hashHex] = docWithRefs.Data
        }
      }
      doc.Add(multihash, dataset[hashHex], tx)
      docsAdded += 1
      totalSize += len(dataset[hashHex])
    }
  }

  // remove missing links
  //fmt.Printf("%+v\n", oldLinksMap)
  for hashHex, cidType := range(oldLinksMap) {
    multihash, _ := hex.DecodeString(hashHex)
    if cidType == 113 { //dag-cbor
      RecursiveRemove(multihash, tx)
    } else {
      //fmt.Printf("Removing %+v\n", hashHex)
      doc.RemoveRef(multihash, tx)
    }
  }

  // update the root doc
  doc.Add(newMultihash, dataset[newMhHex], tx)
  docsAdded += 1
  totalSize += len(dataset[newMhHex])

  return docsAdded, totalSize, docsNotFound

}

func RecursiveAdd(dataset map[string][]byte, newMultihash []byte, tx *bolt.Tx) (docsAdded int, totalSize int) {

  fmt.Printf("RecursiveAdd -> %+v\n", newMultihash)
  log.Fatal("NOT YET IMPLEMENTED")

  // write out root doc
  //doc.Add(newMultihash, dataset[newMhHex], tx)
  //return 1, len(dataset[newMhHex])
  return 1, 0
}

func RecursiveRemove(multihash []byte, tx *bolt.Tx) {

  //log.Printf("RecursiveRemove for %+v\n", hex.EncodeToString(multihash))

  // load the doc, traverse into each linked doc
  doc := db.Doc{}
  docWithRefs, success := doc.Get(multihash, tx)
  if !success {
    if globals.DebugLogging {
      fmt.Printf("multihash not found in db, skipping")
    }
    return
  }

  if docWithRefs.Refs == 1 {
    //fmt.Printf("%+v\n", docWithRefs)
    // doc only has the one ref, which means it will be removed
    // remove the linked docs
    nd, err := cbornode.Decode(docWithRefs.Data, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
    if err != nil {
      log.Printf("ipldwalk.go -> RecursiveRemove() error %+v\n", err)
    } else {
      links := nd.Links()
      if len(links) > 0 {
        for _, link := range links {
          //log.Printf("found cid link %+v\n", link.Cid)
          RecursiveRemove(link.Cid.Hash(), tx)
        }
      }
    }
  }

  // remove the ref to this doc
  doc.RemoveRef(multihash, tx)
}

func Get(multihash []byte, path string, tx *bolt.Tx) (interface{}) {
  //log.Printf("get path %+v\n", path)

  doc := db.Doc{}
  docWithRefs, success := doc.Get(multihash, tx)
  if !success {
    log.Fatal("multihash not found in db..")
  }

  nd, err := cbornode.Decode(docWithRefs.Data, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
  if err != nil {
    log.Fatal("ipldwalk.go -> Get() %+v\n", err)
  }

  pathParts := strings.Split(path, "/")[1:]
  obj, _, _ :=  nd.Resolve(pathParts)

  return obj
}
