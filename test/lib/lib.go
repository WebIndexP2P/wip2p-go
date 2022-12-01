package testlib

import (
  "fmt"
  "strings"
  "errors"
  "math/rand"
  "encoding/hex"
  "crypto/ecdsa"
  "github.com/ipfs/go-cid"
  "github.com/ipfs/go-ipld-cbor"
  mh "github.com/multiformats/go-multihash"
  "github.com/ethereum/go-ethereum/crypto"

  "code.wip2p.com/mwadmin/wip2p-go/signature"
  "code.wip2p.com/mwadmin/wip2p-go/sigbundle/sigbundlestruct"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession/messages"
)

var alpha = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// generates a random string of fixed size
func RandString(size int) string {
    buf := make([]byte, size)
    for i := 0; i < size; i++ {
        buf[i] = alpha[rand.Intn(len(alpha))]
    }
    return string(buf)
}

func CreateBundle(timestamp uint64, objData interface{}, privateKey *ecdsa.PrivateKey) (sigbundlestruct.SigBundle) {
  nd, err := cbornode.WrapObject(objData, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
  if err != nil {
    panic("error wrapping ipld object")
  }

  rootMultihash := "0x" + hex.EncodeToString(nd.Cid().Hash())

  signedString := fmt.Sprintf("[%v,\"%v\"]", timestamp, rootMultihash)
  hash, _ := signature.EthHashString(signedString)
  sigB, _ := signature.Sign(hash, privateKey)

  // construct paste_save payload
  pastedata := make([][]byte, 0)
  pastedata = append(pastedata, nd.RawData() )
  payload := sigbundlestruct.SigBundle{
    CborData: pastedata,
    Signature: sigB,
    Timestamp: uint64(timestamp),
    RootMultihash: nd.Cid().Hash(),
    Account: crypto.PubkeyToAddress(privateKey.PublicKey).Bytes(),
  }

  return payload
}

func Post(session *clientsession.Session, bundle sigbundlestruct.SigBundle) (interface{}, error) {
  type blah struct {
    Result interface{}
    Error error
  }
  w := make(chan blah)
  params := []interface{}{ messages.SigBundleToBundle(&bundle) }
  rpcerr := session.SendRPC("bundle_save", params, func(result interface{}, err error){
    if err != nil {
      session.OnError(err)
      w <- blah{nil, err}
    } else {
      w <- blah{result, nil}
    }
  })
  if rpcerr != nil {
    return nil, rpcerr
  }

  response := <- w
  return response.Result, response.Error
}

func GetMerklehead(session *clientsession.Session) (string) {
  params := []interface{}{}
  w := make(chan string)
  session.SendRPC("peer_info", params, func(result interface{}, err error){
    if err != nil {
      panic(err)
    }
    res := result.(map[string]interface{})
    w <- res["merklehead"].(string)
  })
  merklehead := <- w
  return merklehead
}

func GetDoc(session *clientsession.Session, targetCid string) (map[string]interface{}, error) {
  type Callback struct {
    Result map[string]interface{}
    Error error
  }

  params := []interface{}{}
  params = append(params, targetCid)
  w := make(chan Callback)
  session.SendRPC("doc_get", params, func(result interface{}, err error){
    if err != nil {
      w <- Callback{nil, err}
      return
    }

    cborStr := result.(string)
    cborBytes, _ := hex.DecodeString(cborStr[2:])
    rootDocNode, _ := cbornode.Decode(cborBytes, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
    objIface, _, _ := rootDocNode.Resolve(nil)
    obj := objIface.(map[string]interface{})

    w <- Callback{obj, nil}
  })
  results := <- w
  return results.Result, results.Error
}

func GetMerkleDocForAddress(session *clientsession.Session, address string) (map[string]interface{}, error) {

  merkleheadString := GetMerklehead(session)

  doc, err := GetMerkleDocRecursive(session, merkleheadString, strings.ToLower(address[2:]))
  if err != nil {
    return nil, err
  }

  return doc, nil
}

func GetMerkleDocRecursive(session *clientsession.Session, targetCid string, addressTail string) (map[string]interface{}, error) {

  //fmt.Printf("GetMerkleDocRecursive %s %s\n", targetCid, addressTail)

  // now fetch the child merkle doc
  doc, err := GetDoc(session, targetCid)
  if err != nil {
    return nil, err
  }

  bFoundTimestamp := false
  bFoundCid := false
  for _, value := range doc {
    if fmt.Sprintf("%T", value) == "int" {
      bFoundTimestamp = true
    } else {
      bFoundCid = true
    }
  }
  if bFoundTimestamp && bFoundCid {
    return nil, errors.New("Found both timestamp and cid for " + targetCid)
  }

  if bFoundTimestamp {
    if len(addressTail) == 0 {
      return doc, nil
    }

    if _, exists := doc[addressTail]; !exists {
      return nil, errors.New("Could not find address")
    } else {
      return doc, nil
    }
  } else {
    // scan for cid and recurse into it
    prefix := addressTail[:1]
    tail := addressTail[1:]

    for key, value := range doc {
      if key == prefix {
        newTargetCid := value.(cid.Cid).String()
        return GetMerkleDocRecursive(session, newTargetCid, tail)
      }
    }
  }

  return nil, errors.New("should not get here")

}
