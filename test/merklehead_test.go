package testscript

import (
  "log"
  "fmt"
  "time"
  "strings"
  "testing"
  "crypto/ecdsa"
  "encoding/hex"
  "github.com/ipfs/go-cid"
  "github.com/ipfs/go-ipld-cbor"
  "github.com/ethereum/go-ethereum/crypto"
  mh "github.com/multiformats/go-multihash"

  "code.wip2p.com/mwadmin/wip2p-go/clientsession"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/test/lib"
)

var accounts []*ecdsa.PrivateKey
var session *clientsession.Session
var timestamp uint64

// create 18 accounts
// 1 root/node
// 16 level 1 invites
// 1 to error at 17
func init() {

  clientsession.OnBroadcast = func(session *clientsession.Session, data []interface{}) {
    fmt.Printf("broadcast\n")
  }

  clientsession.OnAuthed = func(session *clientsession.Session) bool {
    fmt.Printf("onauthed\n")
    return true
  }

  clientsession.OnEnd = func(session *clientsession.Session) {
    log.Fatal("Error: Should not disconnect")
  }

  // create accounts
  accounts = make([]*ecdsa.PrivateKey, 181)
  var tmperr error
  tmperr = tmperr
  for a := 0; a < 181; a++ {
    tmpKey := make([]byte, 32)
    tmpKey[31] = byte(a+1)
    accounts[a], tmperr = crypto.ToECDSA(tmpKey)
    if tmperr != nil {
      log.Fatal(tmperr)
    }
    //log.Printf("%s\n", crypto.PubkeyToAddress(accounts[a].PublicKey))
    //accounts[a], _ = crypto.GenerateKey()
  }
  //log.Fatal("exit")

  //globals.DebugLogging = true
  globals.NodePrivateKey = accounts[0]
  globals.PublicMode = true

  // establish encrypted session with peer
  session = clientsession.Create()
  session.RemoteEndpoint = "ws://127.0.0.1:9472"
  session.OnError = func(err error) {
    fmt.Printf(err.Error() + "\n")
  }

  err := session.Dial()
  if err != nil {
    fmt.Printf("%+v\n", err)
    return
  }

  // this blocks
  // the peerId of this node will be assigned as root
  session.StartAuthProcess()

  if !session.HasAuthed {
    fmt.Printf("HasAuthed = false, something gone wrong")
    return
  }
}

var invites []map[string]interface{}

func TestInvite(t *testing.T) {

  timestamp = uint64(time.Now().UTC().Unix()) + 1

  // first account to post with invites for remaining 16
  invites = make([]map[string]interface{}, 0)
  for _, account := range accounts[1:181] {
    accountB := crypto.PubkeyToAddress(account.PublicKey).Bytes()
    invites = append(invites, map[string]interface{}{"account": accountB, "timestamp": timestamp})
  }
  inviteKey := map[string]interface{}{"i": invites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

  timestamp++
  // make a post from 15 accounts
  for _, account := range accounts[1:16] {
    testDoc := "hi there " + testlib.RandString(10)
    bundle := testlib.CreateBundle(timestamp, testDoc, account)
    _, err := testlib.Post(session, bundle)
    if err != nil {
      t.Errorf("%s", err)
      return
    }
  }

}

func TestCheckMerkleheadHas16(t *testing.T) {
  merkleheadString := testlib.GetMerklehead(session)
  //fmt.Printf("%+v\n", merkleheadString)

  params := []interface{}{}
  params = append(params, merkleheadString)
  w := make(chan bool)
  session.SendRPC("doc_get", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("got error %+v", err)
    }
    cborStr := result.(string)
    cborBytes, _ := hex.DecodeString(cborStr[2:])
    rootDocNode, _ := cbornode.Decode(cborBytes, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
    objIface, _, _ := rootDocNode.Resolve(nil)
    obj := objIface.(map[string]interface{})

    if len(obj) != 16 {
      t.Errorf("merklehead expects 16 string keys, got %v", len(obj))
    }

    w <- true
  })
  <- w
}

func TestPost17ForcingNewCids(t *testing.T) {

  timestamp++
  testDoc := "hi there " + testlib.RandString(10)
  bundle := testlib.CreateBundle(timestamp, testDoc, accounts[17])
  _, err := testlib.Post(session, bundle)

  if err != nil {
    t.Errorf(err.Error())
    return
  }
}

func TestCheckMerkleheadIsNowCidsAndAccount17IsThere(t *testing.T) {
  merkleheadString := testlib.GetMerklehead(session)
  //fmt.Printf("%+v\n", merkleheadString)
  var targetCid cid.Cid

  params := []interface{}{}
  params = append(params, merkleheadString)
  w := make(chan bool)
  session.SendRPC("doc_get", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("got error %+v", err)
    }
    cborStr := result.(string)
    cborBytes, _ := hex.DecodeString(cborStr[2:])
    rootDocNode, _ := cbornode.Decode(cborBytes, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
    objIface, _, _ := rootDocNode.Resolve(nil)
    obj := objIface.(map[string]interface{})

    for k, v := range obj {
      if len(k) != 1 {
        t.Errorf("merklehead expects prefix->cids not address->timestamp")
        break
      }
      if fmt.Sprintf("%T", v) != "cid.Cid" {
        t.Errorf("merklehead expects type cid.Cid, got %s", fmt.Sprintf("%T", v))
        break
      }
    }

    //check for account 17
    account := crypto.PubkeyToAddress(accounts[17].PublicKey)
    prefix := strings.ToLower(account.String()[2:])[0:1]
    targetCid = obj[prefix].(cid.Cid)

    w <- true
  })
  <- w

  // now fetch the child merkle doc
  params = []interface{}{}
  params = append(params, targetCid.String())
  session.SendRPC("doc_get", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("got error %+v", err)
      w <- true
      return
    }
    cborStr := result.(string)
    cborBytes, _ := hex.DecodeString(cborStr[2:])
    rootDocNode, _ := cbornode.Decode(cborBytes, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
    objIface, _, _ := rootDocNode.Resolve(nil)
    obj := objIface.(map[string]interface{})

    account := crypto.PubkeyToAddress(accounts[17].PublicKey)
    tail := strings.ToLower(account.String()[2:])[1:]

    bFound := false
    for k := range obj {
      if k == tail {
        bFound = true
      }
    }
    if !bFound {
      t.Errorf("account 17 not found")
      w <- true
      return
    }

    w <- true
  })
  <- w
}

func TestPost18To23IntoExistingCids(t *testing.T) {
  timestamp++
  for a := 18; a < 24; a++ {
    testDoc := "hi there " + testlib.RandString(10)
    bundle := testlib.CreateBundle(timestamp, testDoc, accounts[a])
    _, err := testlib.Post(session, bundle)

    if err != nil {
      t.Errorf(err.Error())
      return
    }
  }
}

func TestCheckMerkledocF(t *testing.T) {

  _, err := testlib.GetMerkleDocForAddress(session, "0xf")
  if err != nil {
    t.Errorf(err.Error())
    return
  }

}

func TestPost24IntoNewCid(t *testing.T) {
  timestamp++
  testDoc := "hi there " + testlib.RandString(10)
  bundle := testlib.CreateBundle(timestamp, testDoc, accounts[24])
  _, err := testlib.Post(session, bundle)

  if err != nil {
    t.Errorf(err.Error())
    return
  }
}

func TestCheckAccount24InMerklehead(t *testing.T) {

  account := crypto.PubkeyToAddress(accounts[24].PublicKey)
  //prefix := strings.ToLower(account.String()[2:])[0:1]
  tail := strings.ToLower(account.String()[2:])[1:]

  doc, err := testlib.GetMerkleDocForAddress(session, account.String())
  if err != nil {
    t.Errorf("got error %+v", err)
    return
  }

  bFound := false
  for k := range doc {
    if k == tail {
      bFound = true
    }
  }
  if !bFound {
    t.Errorf("account 24 not found")
    return
  }
}

func TestPost25To181ForMerklehead2ndLevel(t *testing.T) {
  timestamp++
  for a := 25; a < 181; a++ {
    testDoc := "hi there " + testlib.RandString(10)
    bundle := testlib.CreateBundle(timestamp, testDoc, accounts[a])
    _, err := testlib.Post(session, bundle)

    if err != nil {
      t.Errorf(err.Error())
      return
    }
  }
}

/*func TestExit(t *testing.T) {
  log.Fatal("exit")
}*/


func TestCheckMerkledocForAccount180(t *testing.T) {
  account180 := crypto.PubkeyToAddress(accounts[180].PublicKey)
  _, err := testlib.GetMerkleDocForAddress(session, account180.String())
  if err != nil {
    t.Errorf(err.Error())
    return
  }
}

func TestRemoveAccount180(t *testing.T) {
  timestamp++
  invites := invites[:179]
  inviteKey := map[string]interface{}{"i": invites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}

func TestCheckMerkledocAccount180IsRemoved(t *testing.T) {
  account180 := crypto.PubkeyToAddress(accounts[180].PublicKey)
  _, err := testlib.GetMerkleDocForAddress(session, account180.String())
  if err == nil {
    t.Errorf("expects account not found error")
    return
  }
}

func TestCheckMerkledoc6IsTimestamps(t *testing.T) {
  doc, err := testlib.GetMerkleDocForAddress(session, "0x6")
  if err != nil {
    t.Errorf(err.Error())
    return
  }

  for _, value := range doc {
    if fmt.Sprintf("%T", value) == "cid.Cid" {
      t.Errorf("expect account/timestamp not Cids")
      return
    }
  }
}

//rollback to account 36
func TestRemoveAccountsBackTo36(t *testing.T) {
  timestamp++
  invites := invites[:36]
  inviteKey := map[string]interface{}{"i": invites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}
