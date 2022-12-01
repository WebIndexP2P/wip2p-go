package core

import (
  "fmt"
  "strings"
  "net/http"
  "errors"
  "bytes"
  "net/url"

  "github.com/ipfs/go-ipld-cbor"
  mh "github.com/multiformats/go-multihash"
  "github.com/ipfs/go-cid"
  "github.com/ipld/go-codec-dagpb"
  "github.com/ipld/go-ipld-prime/node/basicnode"
  cidlink "github.com/ipld/go-ipld-prime/linking/cid"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/names"
  "code.wip2p.com/mwadmin/wip2p-go/account"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)

func serveContent(w http.ResponseWriter, r *http.Request) {

  if !globals.PublicMode && !globals.EnableApi {
    http.Error(w, "serveContent not available in private mode", 403)
    return
  }

  // should support names and CIDs
  // check the domain name component first e.g. bay1234.localhost/serve , then the uri path /serve/bay1234
  nameparts := strings.Split(r.Host, ":")
  var targetCid string
  var remainingPath string
  if (strings.HasSuffix(nameparts[0], ".localhost")) {
    targetCid = nameparts[0][:len(nameparts[0])-10]
    remainingPath = r.URL.String()[6:]
  } else {
    slashPos := strings.Index(r.URL.String()[7:], "/")
    if slashPos == -1 {
      w.Header().Set("Location", r.URL.String() + "/")
      w.WriteHeader(http.StatusFound)
      return
    }
    //fmt.Printf("%+v\n", slashPos)
    targetCid = r.URL.String()[7:slashPos+7]
    remainingPath = r.URL.String()[slashPos+7:]
    if remainingPath == "" {
      remainingPath = "/"
    }
  }
  //fmt.Printf("Cid %s\n", targetCid)

  tCid, err := cid.Decode(targetCid)
  var tPbCid cid.Cid
  if err != nil {
    // maybe its a name?
    address, err := names.Lookup(targetCid)
    if err == nil {
      // lookup rootCid for account
      acct, _ := account.FetchAccountFromDb(address, nil, false)
      tCid = cid.NewCidV1(0x71, acct.RootMultihash)
      tPbCid = cid.NewCidV1(0x70, acct.RootMultihash)
    } else {
      http.Error(w, err.Error(), 404)
      return
    }
  } else {
    tPbCid = cid.NewCidV1(0x70, tCid.Hash())
  }

  //fmt.Printf("Got request for %s %s: %s\n", targetCid, r.URL, remainingPath)
  //fmt.Printf("Cids: %s %s\n", tCid, tPbCid)

  data, err := RecursiveFileFetch(remainingPath, tPbCid)
  if err != nil {
    data, err = RecursiveFileFetch(remainingPath, tCid)
  }
  if err != nil {
    http.Error(w, err.Error(), 404)
    return
  }

  if strings.Contains(r.URL.String(), ".css") {
    w.Header().Set("Content-Type", "text/css")
  }
  if strings.Contains(r.URL.String(), ".svg") {
    w.Header().Set("Content-Type", "image/svg+xml")
  }
  if strings.Contains(r.URL.String(), ".js") {
    w.Header().Set("Content-Type", "text/javascript")
  }
  w.Write(data)
}

func RecursiveFileFetch(path string, rootNode cid.Cid) ([]byte, error) {

  //fmt.Printf("RecursiveFileFetch %s %s\n", path, rootNode)

  tx := db.GetTx(false)
  defer db.RollbackTx(tx)

  doc := db.Doc{}
  docWithRefs, success := doc.Get(rootNode.Hash(), nil)

  if !success {
    return nil, errors.New("not found")
  }

  if len(docWithRefs.Data) == 0 {
    return nil, errors.New("no data")
  }

  //fmt.Printf("Type: %+v\n", rootNode.Type())
  if rootNode.Type() == 0x70 {

    if strings.Index(path, "?") >= 0 {
      path = path[:strings.Index(path, "?")]
    }

    targetPathName := ""
    remainingPath := ""
    slashPos := -1
    if path != "" {
      slashPos = strings.Index(path[1:], "/")
    }
    if slashPos == -1 {
      if path == "/" {
        targetPathName = "index.html"
      } else if path == "" {
        //fmt.Printf("must be a multi cid file!\n")
      } else {
        targetPathName = path
        if targetPathName[:1] == "/" {
          targetPathName = targetPathName[1:]
        }
        targetPathName, _ = url.QueryUnescape(targetPathName)
      }
    } else {
      targetPathName = path[1:slashPos+1]
      remainingPath = path[slashPos+1:]
    }
    if strings.Index(targetPathName, "?") >= 0 {
      targetPathName = targetPathName[:strings.Index(targetPathName, "?")]
    }

    //fmt.Printf("Looking for %s, remain %s\n", targetPathName, remainingPath)

    np := basicnode.Prototype.Any
    nb := np.NewBuilder()
    err := dagpb.DecodeBytes(nb, docWithRefs.Data)
    //fmt.Printf("%+v\n", err)
    if err == nil {
      ndPb := nb.Build()
      links, _ := ndPb.LookupByString("Links")
      //fmt.Printf("Links length = %+v\n", links.Length())
      if links.Length() > 0 {
        li := links.ListIterator()

        var bMultipartFile bytes.Buffer
        for !li.Done() {
          _, item, _ := li.Next()

          if targetPathName == "" {
            linkIface, _ := item.LookupByString("Hash")
            link, _ := linkIface.AsLink()
            targetCid := link.(cidlink.Link)
            data, err := RecursiveFileFetch(remainingPath, targetCid.Cid)
            if err != nil {
              return nil, err
            }
            bMultipartFile.Write(data)
          } else {
            // we found the file, return the results
            nameIface, _ := item.LookupByString("Name")
            name, _ := nameIface.AsString()
            //fmt.Printf("%+v\n", name)

            if name == targetPathName {
              linkIface, _ := item.LookupByString("Hash")
              link, _ := linkIface.AsLink()
              targetCid := link.(cidlink.Link)
              return RecursiveFileFetch(remainingPath, targetCid.Cid)
            }
          }

        }

        return bMultipartFile.Bytes(), nil
      } else {
        fmt.Printf("NO LINKS!\n")
      }
    }
  } else if rootNode.Type() == 0x55 {
    return docWithRefs.Data, nil
  } else {
    nd, err := cbornode.Decode(docWithRefs.Data, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
    if err != nil {
      return nil, err
    }

    rootDoc, _, _ := nd.Resolve(nil)

    if fmt.Sprintf("%T", rootDoc) == "string" {
      bufRootDoc := []byte(rootDoc.(string))
      return bufRootDoc, nil
    } else {
      //fmt.Printf("%T %+v\n", rootDoc, rootDoc)
      return nil, errors.New("not valid content")
    }

  }

  return nil, errors.New("shouldnt get here")
}
