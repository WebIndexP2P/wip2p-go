package testscript

import (
  "log"
  "fmt"
  "time"
  "strings"
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
var origRootTimestamps uint64
var accountOneBundleTimestamp uint64

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
  accounts = make([]*ecdsa.PrivateKey, 10)
  for a := 0; a < 10; a++ {
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

var rootInvites []map[string]interface{}

func TestCreateAccounts(t *testing.T) {

  timestamp = uint64(time.Now().UTC().Unix())
  origRootTimestamps = timestamp
  rootInvites = make([]map[string]interface{}, 0)

  // first account to post with invites for remaining 16
  accountB := crypto.PubkeyToAddress(accounts[1].PublicKey).Bytes()
  rootInvites = append(rootInvites, map[string]interface{}{"account": accountB, "timestamp": timestamp})
  accountB = crypto.PubkeyToAddress(accounts[2].PublicKey).Bytes()
  rootInvites = append(rootInvites, map[string]interface{}{"account": accountB, "timestamp": timestamp})
  inviteKey := map[string]interface{}{"i": rootInvites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}

func TestAccountsExist(t *testing.T) {
  params := []interface{}{}
  rootAccount := crypto.PubkeyToAddress(accounts[0].PublicKey).String()
  params = append(params, rootAccount)
	w := make(chan bool)
	session.SendRPC("ui_getAccount", params, func(result interface{}, err error){
	  if err != nil {
		  t.Errorf("got error %+v", err)
    }
    res := result.(map[string]interface{})
    if res["activeLevel"].(float64) != 0 {
      t.Errorf("expects activeLevel 0, got %+v", res["activeLevel"])
    }
    if res["postcount"].(float64) != 1 {
      t.Errorf("expects postcount 1, got %+v", res["postcount"])
    }
	  w <- true
	})
	<- w

  params = []interface{}{}
  accountOne := crypto.PubkeyToAddress(accounts[1].PublicKey).String()
  params = append(params, accountOne)
  w = make(chan bool)
  session.SendRPC("ui_getAccount", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("got error %+v", err)
    }
    res := result.(map[string]interface{})
    if res["activeLevel"].(float64) != 1 {
      t.Errorf("expects activeLevel 1, got %+v", res["activeLevel"])
    }
    if res["postcount"].(float64) != 0 {
      t.Errorf("expects postcount 0, got %+v", res["postcount"])
    }
    w <- true
  })
  <- w

}

func Test_AccountOneToSaveData(t *testing.T) {

  timestamp++

  accountOneBundleTimestamp = timestamp + 10
  jsonData := map[string]interface{}{"boo": "blah"}

  bundle := testlib.CreateBundle(accountOneBundleTimestamp, jsonData, accounts[1])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}

func Test_GetMerkleheadDoc1(t *testing.T) {

  merkleheadString := testlib.GetMerklehead(session)
  //fmt.Printf("%+v\n", merkleheadString)

  doc, err := testlib.GetDoc(session, merkleheadString)
  if err != nil {
    t.Errorf("got error %+v", err)
    return
  }

  rootAccount := strings.ToLower(crypto.PubkeyToAddress(accounts[0].PublicKey).String())
  accountOne := strings.ToLower(crypto.PubkeyToAddress(accounts[1].PublicKey).String())
  accountTwo := strings.ToLower(crypto.PubkeyToAddress(accounts[2].PublicKey).String())

  _, success := doc[rootAccount[2:]]
  if !success {
    t.Errorf("merklehead rootAccount should exist")
  }

  _, success = doc[accountOne[2:]]
  if !success {
    t.Errorf("merklehead accountOne should exist")
  }
  _, success = doc[accountTwo[2:]]
  if success {
    t.Errorf("merklehead accountTwo should not exist")
  }
  //fmt.Printf("%+v\n", obj)
  //levelOneCid = obj["1"].(cid.Cid)

}

func TestChangeAccountLevel(t *testing.T) {

  timestamp++

  accountB := crypto.PubkeyToAddress(accounts[1].PublicKey).Bytes()
  rootInvites[0] = map[string]interface{}{"account": accountB, "timestamp": origRootTimestamps, "lvlgap": 1}
  inviteKey := map[string]interface{}{"i": rootInvites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}

func TestCheckAccountOneNowLevelTwo(t *testing.T) {
  params := []interface{}{}
  accountOne := crypto.PubkeyToAddress(accounts[1].PublicKey).String()
  params = append(params, accountOne)
  w := make(chan bool)
  session.SendRPC("ui_getAccount", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("got error %+v", err)
    }
    res := result.(map[string]interface{})
    if res["activeLevel"].(float64) != 2 {
      t.Errorf("expects activeLevel 2, got %+v", res["activeLevel"])
    }
    w <- true
  })
  <- w

}

func Test_GetMerkleheadDoc2(t *testing.T) {

  merkleheadString := testlib.GetMerklehead(session)

  doc, err := testlib.GetDoc(session, merkleheadString)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

  accountOne := strings.ToLower(crypto.PubkeyToAddress(accounts[1].PublicKey).String())

  timestamp, success := doc[accountOne[2:]]
  if !success {
    t.Errorf("accountOne not found")
    return
  }

  if timestamp != int(accountOneBundleTimestamp) {
    t.Errorf("merklehead timestamp expects %v, got %v\n", accountOneBundleTimestamp, timestamp)
    return
  }
}

var accountTwoInviteTimestampForAccountOne uint64

func Test_AccountTwoAddSecondInviteForAccountOne(t *testing.T) {

  timestamp++
  accountTwoInviteTimestampForAccountOne = timestamp

  // first account to post with invites for remaining 16
  accountB := crypto.PubkeyToAddress(accounts[1].PublicKey).Bytes()
  invites := make([]map[string]interface{}, 0)
  invites = append(invites, map[string]interface{}{"account": accountB, "timestamp": timestamp, "lvlgap": 1})
  inviteKey := map[string]interface{}{"i": invites}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[2])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}

func TestCheckOneStillLevelTwo(t *testing.T) {
  params := []interface{}{}
  accountOne := crypto.PubkeyToAddress(accounts[1].PublicKey).String()
  params = append(params, accountOne)
  params = append(params, map[string]interface{}{"includeInvites": true})
  w := make(chan bool)
  session.SendRPC("ui_getAccount", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("got error %+v", err)
    }
    res := result.(map[string]interface{})
    if res["activeLevel"].(float64) != 2 {
      t.Errorf("expects activeLevel 2, got %+v", res["activeLevel"])
    }
    if len(res["inviters"].([]interface{})) != 2 {
      t.Errorf("expects two invites, got %+v", len(res["inviters"].([]interface{})))
    }
    w <- true
  })
  <- w

}

func Test_RootAccountDropAccountOne(t *testing.T) {

  timestamp++

  // first account to post with invites for remaining 16
  rootInvites = rootInvites[1:]
  inviteKey := map[string]interface{}{"i": rootInvites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}

func Test_CheckAccountOneNowLevelThree(t *testing.T) {
  params := []interface{}{}
  accountOne := crypto.PubkeyToAddress(accounts[1].PublicKey).String()
  params = append(params, accountOne)
  params = append(params, map[string]interface{}{"includeInvites": true})
  w := make(chan bool)
  session.SendRPC("ui_getAccount", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("got error %+v", err)
    }
    res := result.(map[string]interface{})
    if res["activeLevel"].(float64) != 3 {
      t.Errorf("expects activeLevel 3, got %+v", res["activeLevel"])
    }
    if len(res["inviters"].([]interface{})) != 1 {
      t.Errorf("expects one invite, got %+v", len(res["inviters"].([]interface{})))
    }
    w <- true
  })
  <- w

}

func Test_GetMerkleheadDoc3(t *testing.T) {

  //var levelOneCid cid.Cid
  merkleheadString := testlib.GetMerklehead(session)

  doc, err := testlib.GetDoc(session, merkleheadString)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

  rootAccount := strings.ToLower(crypto.PubkeyToAddress(accounts[0].PublicKey).String())
  accountOne := strings.ToLower(crypto.PubkeyToAddress(accounts[1].PublicKey).String())
  accountTwo := strings.ToLower(crypto.PubkeyToAddress(accounts[2].PublicKey).String())

  _, found := doc[rootAccount[2:]]
  if !found {
    t.Errorf("merklehead rootAccount should exist")
  }
  _, found = doc[accountOne[2:]]
  if !found {
    t.Errorf("merklehead accountOne should not exist")
  }

  _, found = doc[accountTwo[2:]]
  if !found {
    t.Errorf("merklehead accountTwo should exist")
  }
}

func Test_AccountOneCreateInvalidInvite(t *testing.T) {

  jsonData := map[string]interface{}{"boo": "blah"}
  invites := make([]map[string]interface{}, 0)
  accountThree := crypto.PubkeyToAddress(accounts[3].PublicKey)
  invites = append(invites, map[string]interface{}{"account": accountThree, "timestamp": accountTwoInviteTimestampForAccountOne - 1})
  inviteKey := map[string]interface{}{"i": invites}
  jsonData["wip2p"] = inviteKey

  bundle := testlib.CreateBundle(accountOneBundleTimestamp + 1, jsonData, accounts[1])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

}

func Test_AccountThreeDoesntExist(t *testing.T) {
  params := []interface{}{}
  accountThree := crypto.PubkeyToAddress(accounts[3].PublicKey).String()
  params = append(params, accountThree)
  w := make(chan bool)
  session.SendRPC("ui_getAccount", params, func(result interface{}, err error){
    if err == nil {
      t.Errorf("expect account not found error %+v", err)
    }
    w <- true
  })
  <- w

}

func Test_AccountTwoShiftInviteTimestampEarlyToActivateAccountThreeInvites(t *testing.T) {

  timestamp++

  accountB := crypto.PubkeyToAddress(accounts[1].PublicKey).Bytes()
  invites := make([]map[string]interface{}, 0)
  invites = append(invites, map[string]interface{}{"account": accountB, "timestamp": accountTwoInviteTimestampForAccountOne - 2, "lvlgap": 1})
  inviteKey := map[string]interface{}{"i": invites}
  jsonData := map[string]interface{}{"wip2p": inviteKey}


  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[2])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

}

func Test_AccountThreeDoesExist(t *testing.T) {
  params := []interface{}{}
  accountThree := crypto.PubkeyToAddress(accounts[3].PublicKey).String()
  params = append(params, accountThree)
  w := make(chan bool)
  session.SendRPC("ui_getAccount", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("got error %+v", err)
    }
    w <- true
  })
  <- w

}
