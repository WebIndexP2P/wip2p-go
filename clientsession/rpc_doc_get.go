package clientsession

import (
  //"fmt"
  "errors"
  "encoding/hex"
  "encoding/base64"
  "github.com/ipfs/go-cid"

  "code.wip2p.com/mwadmin/wip2p-go/db"
)

func docGet(paramData []interface{}) (string, error) {

  var response string

  inCid, success := paramData[0].(string)
  if !success {
    return "", errors.New("missing cid")
  }

  var inEncoding string
  if len(paramData) == 2 {
    inEncoding, _ = paramData[1].(string)
    if inEncoding != "base64" {
      inEncoding = "hex"
    }
  }

  tmpCid, err := cid.Parse(inCid)
  if err != nil {
    err := errors.New("error with cid, " + err.Error())
    return "", err
  }

  doc := db.Doc{}
  docWithRefs, success := doc.Get(tmpCid.Hash(), nil)

  if !success {
    return "", errors.New("error fetching document")
  }

  if len(docWithRefs.Data) == 0 {
    return "", errors.New("doc with no data")
  }

  if inEncoding == "base64" {
    response = base64.StdEncoding.EncodeToString(docWithRefs.Data)
  } else {
    response = "0x" + hex.EncodeToString(docWithRefs.Data)
  }

  return response, nil
}
