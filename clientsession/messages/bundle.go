package messages

import (
  "log"
  "errors"
  //"strings"
  "encoding/hex"
  "encoding/json"
  "encoding/base64"

  "code.wip2p.com/mwadmin/wip2p-go/sigbundle/sigbundlestruct"
  //"code.wip2p.com/mwadmin/wip2p-go/core/globals"
)

type Bundle struct {
  Account   string   `json:"account,omitempty"`
  Multihash string   `json:"multihash"`
  Signature string   `json:"signature"`
  Timestamp uint64   `json:"timestamp"`
  CborData  []string `json:"cborData"`
  Car       string   `json:"car"`
  LatestSeqNo uint   `json:"latestSeqNo,omitempty"`
}

func ParseBundle(data interface{}) (*Bundle, error) {

  //log.Printf("%+v\n", data)

  var bundle Bundle
  responseB, _ := json.Marshal(data)
  json.Unmarshal(responseB, &bundle)

  if bundle.Multihash == "" {
    return nil, errors.New("expects multihash")
  }

  if bundle.Signature == "" {
    return nil, errors.New("expects signature")
  }

  if bundle.Timestamp == 0 {
    return nil, errors.New("expects Timestamp")
  }

  if (bundle.CborData == nil || len(bundle.CborData) == 0) && len(bundle.Car) == 0 {
    return nil, errors.New("expects cborData or car")
  }


  return &bundle, nil
}

func (b *Bundle) ToSigBundle() (*sigbundlestruct.SigBundle, error) {

  //log.Printf("%+v\n", b)

  sb := sigbundlestruct.SigBundle{}

  // signature
  sb.Signature, _ = hex.DecodeString(b.Signature[2:])
  if len(sb.Signature) != 65 {
    return nil, errors.New("signature expects 65 bytes")
  }

  // cbordata
  sb.CborData = make([][]byte, 0)
  for _, cborItem := range b.CborData {
    cborBytes, _ := base64.StdEncoding.DecodeString(cborItem)
    sb.CborData = append(sb.CborData, cborBytes)
  }

  sb.Car, _ = base64.StdEncoding.DecodeString(b.Car)

  sb.Timestamp = b.Timestamp

  if len(b.Account) == 0 {
    log.Printf("missing account\n")
    return nil, errors.New("missing account")
  }
  sb.Account, _ = hex.DecodeString(b.Account[2:])
  sb.RootMultihash, _ = hex.DecodeString(b.Multihash[2:])

  return &sb, nil
}

func SigBundleToBundle(sb *sigbundlestruct.SigBundle) Bundle {
  bundle := Bundle{}
  bundle.CborData = make([]string, 0)
  for _, cborDataItem := range sb.CborData {
    bundle.CborData = append(bundle.CborData, base64.StdEncoding.EncodeToString(cborDataItem))
  }
  bundle.Signature = "0x" + hex.EncodeToString(sb.Signature)
  bundle.Timestamp = sb.Timestamp
  bundle.Multihash = "0x" + hex.EncodeToString(sb.RootMultihash)
  bundle.Account = "0x" + hex.EncodeToString(sb.Account)
  return bundle
}
