package testscript

import (
  "log"
  "fmt"
  "time"
  "testing"
  "crypto/ecdsa"
  "github.com/ethereum/go-ethereum/crypto"

  "code.wip2p.com/mwadmin/wip2p-go/clientsession"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/test/lib"
  "code.wip2p.com/mwadmin/wip2p-go/util"
)

var accounts []*ecdsa.PrivateKey
var session *clientsession.Session
var timestamp uint64

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
  accounts = make([]*ecdsa.PrivateKey, 5000)
  var tmperr error
  tmperr = tmperr
  for a := 1; a <= 5000; a++ {
    tmpKey := make([]byte, 32)
    intBytes := util.Itob(uint64(a))
    copy(tmpKey[24:32], intBytes)
    accounts[a-1], tmperr = crypto.ToECDSA(tmpKey)
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
  for _, account := range accounts[1:1450] {
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
  for _, account := range accounts[1:1450] {
    testDoc := "hi there " + testlib.RandString(10)
    bundle := testlib.CreateBundle(timestamp, testDoc, account)
    _, err := testlib.Post(session, bundle)
    if err != nil {
      t.Errorf("%s", err)
      return
    }
  }

}

func TestInviteAccount1(t *testing.T) {

  timestamp = uint64(time.Now().UTC().Unix()) + 1

  // first account to post with invites for remaining 16
  invites = make([]map[string]interface{}, 0)
  for _, account := range accounts[1450:2900] {
    accountB := crypto.PubkeyToAddress(account.PublicKey).Bytes()
    invites = append(invites, map[string]interface{}{"account": accountB, "timestamp": timestamp})
  }
  inviteKey := map[string]interface{}{"i": invites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[1])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

  timestamp++
  // make a post from 15 accounts
  for _, account := range accounts[1450:2900] {
    testDoc := "hi there " + testlib.RandString(10)
    bundle := testlib.CreateBundle(timestamp, testDoc, account)
    _, err := testlib.Post(session, bundle)
    if err != nil {
      t.Errorf("%s", err)
      return
    }
  }

}

func TestInviteAccount2(t *testing.T) {

  timestamp = uint64(time.Now().UTC().Unix()) + 1

  // first account to post with invites for remaining 16
  invites = make([]map[string]interface{}, 0)
  for _, account := range accounts[2900:4350] {
    accountB := crypto.PubkeyToAddress(account.PublicKey).Bytes()
    invites = append(invites, map[string]interface{}{"account": accountB, "timestamp": timestamp})
  }
  inviteKey := map[string]interface{}{"i": invites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[2])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

  timestamp++
  // make a post from 15 accounts
  for _, account := range accounts[2900:4350] {
    testDoc := "hi there " + testlib.RandString(10)
    bundle := testlib.CreateBundle(timestamp, testDoc, account)
    _, err := testlib.Post(session, bundle)
    if err != nil {
      t.Errorf("%s", err)
      return
    }
  }

}
