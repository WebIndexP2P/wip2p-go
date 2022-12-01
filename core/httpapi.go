package core

import (
  "strings"
  "errors"
  "net/http"
  "encoding/hex"
  "encoding/json"

  "github.com/ipfs/go-ipld-cbor"
  mh "github.com/multiformats/go-multihash"
  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  acct "code.wip2p.com/mwadmin/wip2p-go/account"
)

func serveHttpApi(w http.ResponseWriter, r *http.Request) {

  if !globals.PublicMode && !globals.EnableApi {
    http.Error(w, "Public API not available in private mode", 403)
    return
  }

  uriParts := strings.Split(r.RequestURI, "/")

  if uriParts[1] != "api" {
    http.Error(w, "invalid path", 400)
    return
  }

  if uriParts[2] == "getcontent" {
    if len(uriParts) != 4 {
      http.Error(w, "missing account", 400)
      return
    }
    result, err := getContent(uriParts[3:])
    if err != nil {
      http.Error(w, err.Error(), 400)
      return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Write([]byte(result))
  /*} else if uriParts[2] == "indexerfeed" {
    serveIndexerFeedWs(w, r)*/
  } else {
    http.Error(w, "unknown api method", 400)
  }

}

func getContent(params []string) (string, error) {
  address := params[0]
  path := ""
  if len(params) > 2 {
    path = params[1]
  }
  path = path

  if len(address) != 42 {
    return "", errors.New("invalid account")
  }

  reqAccountB, _ := hex.DecodeString(address[2:])
  reqAddress := common.BytesToAddress(reqAccountB)

  tx := db.GetTx(false)
  defer db.RollbackTx(tx)

  account, success := acct.FetchAccountFromDb(reqAddress, tx, false)
  if !success {
    return "", errors.New("account not found")
  }

  doc := db.Doc{}
  docWithRefs, _ := doc.Get(account.RootMultihash, tx)

  nd, err := cbornode.Decode(docWithRefs.Data, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
  if err != nil {
    return "", err
  }

  rootDoc, _, _ := nd.Resolve(nil)
  bytes, err := json.Marshal(rootDoc)

  if err != nil {
    return "", err
  }

  return string(bytes), nil
}
