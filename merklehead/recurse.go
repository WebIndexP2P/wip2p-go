package merklehead

import (
  "fmt"
  "log"
  "errors"

  bolt "go.etcd.io/bbolt"
  "github.com/ipfs/go-cid"
  "github.com/ipfs/go-ipld-cbor"
  mh "github.com/multiformats/go-multihash"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)

func addItemRecursive(docCidBytes []byte, addressTail string, timestamp uint64, tx *bolt.Tx) (*cid.Cid, error) {

  if globals.DebugLogging {
    fmt.Printf("merklehead.addItemRecursive() Cid %v, addressTail %v\n", cid.NewCidV1(0x71, docCidBytes), addressTail)
  }

  doc := db.Doc{}

  rootDoc := make(map[string]interface{})

  // fetch the root doc
  docWithRefs, success := doc.Get(docCidBytes, tx)
  if !success {
    log.Fatal("doc is missing ", cid.NewCidV1(0x71, docCidBytes))
  }

  rootDocNode, err := cbornode.Decode(docWithRefs.Data, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
  if err != nil {
    log.Fatal(err)
  }

  rootDocIFace, _, _ := rootDocNode.Resolve(nil)
  rootDoc = rootDocIFace.(map[string]interface{})

  // check if keys are partial->cids or address->timestamp
  //var key string
  var value interface{}
  for _, value = range rootDoc {
    break
  }

  if fmt.Sprintf("%T", value) == "int" {
    if len(rootDoc) < 16 {
      rootDoc[addressTail] = timestamp
    } else {
      // the merkledoc is full, convert to child docs
      newRootDoc := make(map[string]interface{})
      for address, timestamp := range rootDoc {
        prefix := address[:1]
        tail := address[1:]
        if _, exists := newRootDoc[prefix]; exists {
          tmpDoc := newRootDoc[prefix].(map[string]interface{})
          tmpDoc[tail] = timestamp
        } else {
          tmpDoc := make(map[string]interface{})
          tmpDoc[tail] = timestamp
          newRootDoc[prefix] = tmpDoc
        }
      }
      // dont forget to add the original account that posted something
      prefix := addressTail[:1]
      tail := addressTail[1:]
      if _, exists := newRootDoc[prefix]; exists {
        tmpDoc := newRootDoc[prefix].(map[string]interface{})
        tmpDoc[tail] = timestamp
      } else {
        tmpDoc := make(map[string]interface{})
        tmpDoc[tail] = timestamp
        newRootDoc[prefix] = tmpDoc
      }

      // now loop through all the new cid docs and save them
      for prefix, childDoc := range newRootDoc {
        newChildNode, err := cbornode.WrapObject(childDoc, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
        if err != nil {
          log.Fatal(err)
        }

        // adds doc to doc store
        doc := db.Doc{}
        doc.Add(newChildNode.Cid().Hash(), newChildNode.RawData(), tx)

        newRootDoc[prefix] = newChildNode.Cid()
      }

      rootDoc = newRootDoc
    }
  } else {
    // this merkledoc is already in cids mode
    prefix := addressTail[:1]
    tail := addressTail[1:]
    var newCid *cid.Cid

    if cidHash, exists := rootDoc[prefix]; exists {

      childCid := cidHash.(cid.Cid)

      //fmt.Printf("--> Adding to existing Cid doc %s\n", childCid)

      newCid, _ = addItemRecursive(childCid.Hash(), tail, timestamp, tx)

    } else if len(rootDoc) == 16 {
      log.Printf("%+v\n%+v\n", rootDoc, prefix)
      log.Fatal("we should have found a match")
    } else {
      // make a new child node
      childDoc := make(map[string]interface{})
      childDoc[tail] = timestamp

      newChildNode, err := cbornode.WrapObject(childDoc, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
      if err != nil {
        log.Fatal(err)
      }

      // adds doc to doc store
      doc := db.Doc{}
      doc.Add(newChildNode.Cid().Hash(), newChildNode.RawData(), tx)
      tmpCid := newChildNode.Cid()
      newCid = &tmpCid

      //fmt.Printf("--> Created brand new Cid doc %s\n", newCid)
    }

    rootDoc[prefix] = newCid
  }

  newDocNode, err := cbornode.WrapObject(rootDoc, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
  if err != nil {
    log.Fatal(err)
  }

  // adds doc to doc store
  doc.Add(newDocNode.Cid().Hash(), newDocNode.RawData(), tx)

  //UpdateMerklehead(rootDoc, addressTail, tx)
  newCid := newDocNode.Cid()

  return &newCid, nil
}

// returns
//  cid of child doc
//  map of account/timestamps
//  isCollapseDoc
//  error
func removeItemRecursive(docCidBytes []byte, addressTail string, tx *bolt.Tx) (*cid.Cid, map[string]interface{}, bool, error) {

  if globals.DebugLogging {
    fmt.Printf("merklehead.removeItemRecursive() Cid %v, addressTail %v\n", cid.NewCidV1(0x71, docCidBytes), addressTail)
  }

  doc := db.Doc{}

  rootDoc := make(map[string]interface{})

  // fetch the root doc
  docWithRefs, success := doc.Get(docCidBytes, tx)
  if !success {
    log.Fatal("doc is missing ", cid.NewCidV1(0x71, docCidBytes))
  }

  rootDocNode, err := cbornode.Decode(docWithRefs.Data, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
  if err != nil {
    log.Fatal(err)
  }

  rootDocIFace, _, _ := rootDocNode.Resolve(nil)
  rootDoc = rootDocIFace.(map[string]interface{})

  /******************************
  recurse down to the account being removed
  remove account
  store any other account/timestamps in array
  go up one recursion returning array and new cid
  recurse into other cids and append account/timestamps to array
  if array hits 17, return to top with just cid
  if we are 16 or less, construct a new doc with the accounts/timestamps
  remove all the child cids
  return with just the cid
  ******************************/

  // check if keys are partial->cids or address->timestamp
  //var key string
  var value interface{}
  for _, value = range rootDoc {
    break
  }

  var remainMap map[string]interface{}
  isBottomLevel := false

  if fmt.Sprintf("%T", value) == "int" {
    isBottomLevel = true
    bFound := false
    for address, _ := range rootDoc {
      if address == addressTail {
        bFound = true
        delete(rootDoc, address)
        break
      }
    }
    if !bFound {
      return nil, nil, true, errors.New("address not found")
    } else {
      // add remainder items to arrItems
      remainMap = make(map[string]interface{})
      for addressTail, timestamp := range rootDoc {
        remainMap[addressTail] = timestamp
      }
    }
  } else {
    // this doc contains cid links
    //isBottomLevel := false //defaults to false
    prefix := addressTail[:1]
    tail := addressTail[1:]
    targetCid := rootDoc[prefix].(cid.Cid)
    newCid, tmpRemainMap, bottomLevelFound, err := removeItemRecursive(targetCid.Hash(), tail, tx)
    if err != nil {
      return nil, nil, bottomLevelFound, err
    }

    // init the remainMap
    remainMap = make(map[string]interface{})

    // remove childCid if its empty
    if len(tmpRemainMap) == 0 {
      delete(rootDoc, prefix)
      newCid = nil
    } else {
      for address, timestamp := range tmpRemainMap {
        remainMap[address] = timestamp
      }
    }

    if bottomLevelFound {
      //fmt.Printf("Now we iterate the other cids\n")
      for addressTail, cidIface := range rootDoc {
        if addressTail == prefix {
          continue
        }
        targetCid := cidIface.(cid.Cid)
        tmpDoc := FetchDoc(targetCid.Hash(), tx)
        //fmt.Printf("FetchDoc = %+v\n", tmpDoc)

        foundCids := false
        for childAddressTail, value := range tmpDoc {
          if fmt.Sprintf("%T", value) == "cid.Cid" {
            foundCids = true
            break
          } else {
            remainMap[childAddressTail] = value
          }

          if len(remainMap) > 16 {
            break
          }
        }

        if foundCids {
          break
        }

        if len(remainMap) > 16 {
          break
        }
      }

      //fmt.Printf("remainMap length = %v\n", len(remainMap))
      if len(remainMap) <= 16 {
        //fmt.Printf("remainMap is <= 16, collapse child cids\n")
        rootDoc = remainMap
        newCid = nil
      }
    }

    // update the cid
    if newCid != nil {
      rootDoc[prefix] = newCid
    }
  }

  // save the updated doc
  newNode, err := cbornode.WrapObject(rootDoc, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
  if err != nil {
    log.Fatal(err)
  }

  // adds doc to doc store
  doc.Add(newNode.Cid().Hash(), newNode.RawData(), tx)
  tmpCid := newNode.Cid()
  newCid := &tmpCid

  return newCid, remainMap, isBottomLevel, nil
}
