package ipldwalk

import (
  "log"
  "fmt"
  "encoding/hex"
  "code.wip2p.com/mwadmin/wip2p-go/db"

  "github.com/ipfs/go-cid"
  bolt "go.etcd.io/bbolt"
  "github.com/ipld/go-codec-dagpb"
  "github.com/ipld/go-ipld-prime/node/basicnode"
  cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

func WalkAndSaveUnixFS(dataset map[string][]byte, newCid cid.Cid, tx *bolt.Tx) (docsAdded int, totalSize int, docsNotFound int) {

  newMhHex := hex.EncodeToString(newCid.Hash())
  //fmt.Printf("WalkAndSaveUnixFS for %+v\n", newCid)

  if _, bFound := dataset[newMhHex]; !bFound {
    return 0, 0, 1
  }

  doc := db.Doc{}
  doc = doc

  np := basicnode.Prototype.Any
  nb := np.NewBuilder()

  //fmt.Printf("decoding %+v %+v\n", newCid.Type(), len(dataset[newMhHex]))
  if newCid.Type() == 0x70 {
    err := dagpb.DecodeBytes(nb, dataset[newMhHex])
    if err != nil {
      fmt.Printf("error %s with %+v\n", err)
      return 0, 0, 0
    }
    ndPb := nb.Build()
    links, _ := ndPb.LookupByString("Links")

    //fmt.Printf("Links %+v\n", links.Length())
    docsFound := 0
    totalSize := 0
    docsMissing := 0
    if links.Length() > 0 {
      li := links.ListIterator()
      for !li.Done() {
        _, item, err := li.Next()
        if err != nil {
          log.Printf("ERROR %+v\n", err)
        }

        hashIface, _ := item.LookupByString("Hash")
        link, _ := hashIface.AsLink()
        tmpCid := link.(cidlink.Link)
        //name, _ := item.LookupByString("Name")
        //nameStr, _ := name.AsString()
        //fmt.Printf("%+v %+v\n", nameStr, tmpCid)

        tmpDocsFound, tmpTotalSize, tmpDocsMissing := WalkAndSaveUnixFS(dataset, tmpCid.Cid, tx)
        docsFound += tmpDocsFound
        totalSize += tmpTotalSize
        docsMissing += tmpDocsMissing
      }
    } else {
      // looks like its a data doc
      //data, _ := ndPb.LookupByString("Data")
      //fmt.Printf("Data\n%+v\n", data)
    }

    doc.Add(newCid.Hash(), dataset[newMhHex], tx)
    docsFound += 1
    totalSize += len(dataset[newMhHex])
    //docsMissing += 0

    //fmt.Printf("Returning %v %v %v\n", docsFound, totalSize, docsMissing)
    return docsFound, totalSize, docsMissing

  } else if newCid.Type() == 0x55 {
    // save raw file
    doc.Add(newCid.Hash(), dataset[newMhHex], tx)
    return 1, len(dataset[newMhHex]), 0
  }

  return 0, 0, 0
}
