package clientsession

import (
  "fmt"
  "log"
  //"errors"

  //"bytes"
  "encoding/json"
  "encoding/base64"

  bolt "go.etcd.io/bbolt"
  "github.com/ipfs/go-cid"
  "github.com/ipfs/go-ipld-cbor"
  mh "github.com/multiformats/go-multihash"
  "github.com/ipld/go-codec-dagpb"
  "github.com/ipld/go-ipld-prime/node/basicnode"
  cidlink "github.com/ipld/go-ipld-prime/linking/cid"

  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/account"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  //"code.wip2p.com/mwadmin/wip2p-go/sigbundle"
  //"code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
)

func (s *Session) StartLinkedDocsSwap() {
  log.Printf("Start LinkedDocs Swap...\n")

  tx := db.GetTx(true)
  defer db.CommitTx(tx)

  b := tx.Bucket([]byte("accounts"))
  b.ForEach(func(k, accountB []byte) error {
    var account account.AccountStruct

  	json.Unmarshal(accountB, &account)
  	account.Address = common.BytesToAddress(k)

    //fmt.Printf("%+v\n", account)

    if account.SyncStatus == 2 {

      //fmt.Printf("Fetching LinkedDocs for %+v\n", account.Address)

      //recursively walk the ipld docs fetching as we go
      tmpCid := cid.NewCidV1(0x71, account.RootMultihash)
      docs, size, missed := s.RecursiveFetchDocs(tmpCid, tx)
      if docs == 0 {
        tmpCid = cid.NewCidV1(0x70, account.RootMultihash)
        docs, size, missed = s.RecursiveFetchDocs(tmpCid, tx)
      }

      //fmt.Printf("Finished recursion result: %+v %+v %+v\n", docs, size, missed)
      if missed == 0 {
        account.SyncStatus = 1
        account.SaveToDb(tx)
        log.Printf("Downloaded %+v docs with %+v bytes for Account %+v\n", docs, size, account.Address)
      }
    }
    return nil
  })

  log.Printf("LinkedDocs Swap finished\n")

}

func (s *Session) RecursiveFetchDocs(targetCid cid.Cid, tx *bolt.Tx) (docsFetchedFromPeer int, totalSizeDownloaded int, docsMissing int) {

  //fmt.Printf("RecursiveFetchDocs %+v\n", targetCid)

  // fetch the doc
  // write it to db
  // get the links and recursively go into them
  doc := db.Doc{}
  docWithRefs, success := doc.Get(targetCid.Hash(), tx)
  var docBytes []byte
  if success && len(docWithRefs.Data) > 0 {
    docBytes = docWithRefs.Data
  } else {
    //fetch from peer
    var err interface{}
    docBytes, err = s.RequestDocFromPeer(targetCid)
    if err != nil {
      return 0, 0, 1
    } else {
      doc.Add(targetCid.Hash(), docBytes, tx)
      //fmt.Printf("FIXME: Write to DB\n")
      docsFetchedFromPeer = 1
      totalSizeDownloaded = len(docBytes)
    }
  }

  if targetCid.Type() == 0x71 {
    nd, err := cbornode.Decode(docBytes, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
    if err != nil {
      //fmt.Printf("error decoding, assume its a dag-pb file %+v %+v %+v\n", docsFound, totalSize, docsMissing)
      return docsFetchedFromPeer, totalSizeDownloaded, 0
    }

    for _, link := range(nd.Links()) {
      docs, size, missed := s.RecursiveFetchDocs(link.Cid, tx)
      docsFetchedFromPeer += docs
      totalSizeDownloaded += size
      docsMissing += missed
    }
  } else if targetCid.Type() == 0x70 {
    np := basicnode.Prototype.Any
    nb := np.NewBuilder()

    err := dagpb.DecodeBytes(nb, docBytes)
    if err != nil {
      fmt.Printf("error %s with %+v\n", err)
      return 0, 0, 0
    }
    ndPb := nb.Build()
    links, _ := ndPb.LookupByString("Links")
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

        docs, size, missed := s.RecursiveFetchDocs(tmpCid.Cid, tx)
        docsFetchedFromPeer += docs
        totalSizeDownloaded += size
        docsMissing += missed
      }
    }
  }

  return docsFetchedFromPeer, totalSizeDownloaded, docsMissing
}

func (s *Session) RequestDocFromPeer(targetCid cid.Cid) ([]byte, error) {

  //log.Printf("RequestDocFromPeer() %+v\n", targetCid)

  var resultB []byte
  waitchan := make(chan error)
  s.SendRPC("doc_get", []interface{}{targetCid.String(), "base64"}, func(result interface{}, err error){
    //log.Printf("doc_get responded %+v %+v\n", result, err)
    if err != nil {
      if globals.DebugLogging {
        log.Printf("swap_linkeddocs.go RequestDocFromPeer() %+v\n", err)
      }
      waitchan <- err
      return
    }

    resultB, _ = base64.StdEncoding.DecodeString(result.(string))

    waitchan <- nil
  })

  err := <- waitchan
  return resultB, err
}
