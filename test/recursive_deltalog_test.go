package testscript

import (
  "fmt"
  "log"
  "time"
  "testing"
  "crypto/ecdsa"
  "github.com/ethereum/go-ethereum/crypto"

  "code.wip2p.com/mwadmin/wip2p-go/clientsession"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/test/lib"
)

var accounts []*ecdsa.PrivateKey
var session *clientsession.Session
var timestamp uint64
var origInviteTimestamp uint64

func init() {

  clientsession.OnBroadcast = func(session *clientsession.Session, data []interface{}) {
    fmt.Printf("broadcast\n")
  }

  clientsession.OnAuthed = func(session *clientsession.Session) bool {
    //fmt.Printf("onauthed\n")
    return true
  }

  clientsession.OnEnd = func(session *clientsession.Session) {
    log.Fatal("Error: Should not disconnect")
  }

  // create accounts
  accounts = make([]*ecdsa.PrivateKey, 5)
  for a := 0; a < 5; a++ {
    accounts[a], _ = crypto.GenerateKey()
  }

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

func Test_CreateNestedAccounts(t *testing.T) {

  timestamp = uint64(time.Now().UTC().Unix()) + 1
  origInviteTimestamp = timestamp

  // Account A (root) to post with invite for account B
  invites := make([]map[string]interface{}, 0)
  accountToBeInvited := crypto.PubkeyToAddress(accounts[1].PublicKey).Bytes()
  invites = append(invites, map[string]interface{}{"account": accountToBeInvited, "timestamp": timestamp})
  inviteKey := map[string]interface{}{"i": invites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

  // Account B to post something
  timestamp++
  accountToBeInvited = crypto.PubkeyToAddress(accounts[2].PublicKey).Bytes()
  jsonData = map[string]interface{}{"test":"hi there"}

  bundle = testlib.CreateBundle(timestamp, jsonData, accounts[1])
  _, err = testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

}

func Test_RemoveInviteForAccountB(t *testing.T) {
  // Account A (root) to remove invite for account B
  timestamp++
  invites := make([]map[string]interface{}, 0)
  inviteKey := map[string]interface{}{"i": invites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}

func Test_CheckSeqNos(t *testing.T) {
  params := []interface{}{}
  params = append(params, 0)
  w := make(chan bool)
  session.SendRPC("bundle_getBySequence", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf(err.Error())
    }

    resultArr := result.([]interface{})
    if len(resultArr) != 2 {
      t.Errorf("expecting two seq item")
    }

    w <- true
  })
  <- w
}

func Test_ReinviteAccountB(t *testing.T) {

  timestamp++

  // Account A (root) to post with invite for account B
  invites := make([]map[string]interface{}, 0)
  accountToBeInvited := crypto.PubkeyToAddress(accounts[1].PublicKey).Bytes()
  invites = append(invites, map[string]interface{}{"account": accountToBeInvited, "timestamp": origInviteTimestamp})
  inviteKey := map[string]interface{}{"i": invites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}

func Test_CheckSeqNosAgain(t *testing.T) {
  params := []interface{}{}
  params = append(params, 0)
  w := make(chan bool)
  session.SendRPC("bundle_getBySequence", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf(err.Error())
    }

    resultArr := result.([]interface{})

    if len(resultArr) != 2 {
      t.Errorf("Expects two seq accounts")
    }

    seq2 := resultArr[1].(map[string]interface{})
    if seq2["seqNo"].(float64) != 6 {
      t.Errorf("AccountB seqNo expects 6, got %v\n", seq2["seqNo"])
    }

    w <- true
  })
  <- w
}

func Test_CheckPeerInfo(t *testing.T) {
  params := []interface{}{}
  w := make(chan bool)
  session.SendRPC("peer_info", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf(err.Error())
    }

    info := result.(map[string]interface{})

    if info["latestSequenceNo"].(float64) != 6 {
      t.Errorf("Expect latest sequence number of 6, got %v\n", info["latestSequenceNo"])
    }

    w <- true
  })
  <- w
}
