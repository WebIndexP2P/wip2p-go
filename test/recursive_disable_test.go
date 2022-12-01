package testscript

import (
  "fmt"
  "log"
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
var accountBInviteTimestamp uint64
var accountCLastPosTimestamp uint64

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

  // Account A (root) to post with invite for account B
  invites := make([]map[string]interface{}, 0)
  accountToBeInvited := crypto.PubkeyToAddress(accounts[1].PublicKey).Bytes()
  fmt.Printf("%s invites %s\n", crypto.PubkeyToAddress(accounts[0].PublicKey).String(), crypto.PubkeyToAddress(accounts[1].PublicKey).String())
  invites = append(invites, map[string]interface{}{"account": accountToBeInvited, "timestamp": timestamp})
  inviteKey := map[string]interface{}{"i": invites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

  // Account B to invite C
  timestamp++
  accountBInviteTimestamp = timestamp
  invites = make([]map[string]interface{}, 0)
  accountToBeInvited = crypto.PubkeyToAddress(accounts[2].PublicKey).Bytes()
  fmt.Printf("%s invites %s\n", crypto.PubkeyToAddress(accounts[1].PublicKey).String(), crypto.PubkeyToAddress(accounts[2].PublicKey).String())
  invites = append(invites, map[string]interface{}{"account": accountToBeInvited, "timestamp": timestamp})
  inviteKey = map[string]interface{}{"i": invites}
  jsonData = map[string]interface{}{"wip2p": inviteKey}

  bundle = testlib.CreateBundle(timestamp, jsonData, accounts[1])
  _, err = testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

  // Account C to invite D
  timestamp++
  accountCLastPosTimestamp = timestamp
  invites = make([]map[string]interface{}, 0)
  accountToBeInvited = crypto.PubkeyToAddress(accounts[3].PublicKey).Bytes()
  fmt.Printf("%s invites %s\n", crypto.PubkeyToAddress(accounts[2].PublicKey).String(), crypto.PubkeyToAddress(accounts[3].PublicKey).String())
  invites = append(invites, map[string]interface{}{"account": accountToBeInvited, "timestamp": timestamp})
  inviteKey = map[string]interface{}{"i": invites}
  jsonData = map[string]interface{}{"wip2p": inviteKey}

  bundle = testlib.CreateBundle(timestamp, jsonData, accounts[2])
  _, err = testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

  // Account D to invite E
  timestamp++
  invites = make([]map[string]interface{}, 0)
  accountToBeInvited = crypto.PubkeyToAddress(accounts[4].PublicKey).Bytes()
  fmt.Printf("%s invites %s\n", crypto.PubkeyToAddress(accounts[3].PublicKey).String(), crypto.PubkeyToAddress(accounts[4].PublicKey).String())
  invites = append(invites, map[string]interface{}{"account": accountToBeInvited, "timestamp": timestamp})
  inviteKey = map[string]interface{}{"i": invites}
  jsonData = map[string]interface{}{"wip2p": inviteKey}

  bundle = testlib.CreateBundle(timestamp, jsonData, accounts[3])
  _, err = testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

}

func Test_RemoveLevelTwoAccountInvite(t *testing.T) {
  // first account to post with invites for remaining 16
  timestamp++
  inviteKey := map[string]interface{}{"i": "it's nothin"}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[1])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}

func Test_ConfirmChildAccountsDisabled(t *testing.T) {
  params := []interface{}{}
  removedAccount := crypto.PubkeyToAddress(accounts[2].PublicKey).String()
  params = append(params, map[string]interface{}{ "account": removedAccount })
  w := make(chan bool)
  session.SendRPC("bundle_get", params, func(result interface{}, err error){
    if err == nil {
      t.Errorf("expect error 'account not found'")
    } else if err.Error() != "account not found" {
      t.Errorf("unexpected error %s", err)
    }
    w <- true
  })
  <- w

  // account [3]
  params = []interface{}{}
  removedAccount = crypto.PubkeyToAddress(accounts[3].PublicKey).String()
  params = append(params, map[string]interface{}{ "account": removedAccount })
  session.SendRPC("bundle_get", params, func(result interface{}, err error){
    if err == nil {
      t.Errorf("expect error for accounts[3] 'account not found'")
    } else if err.Error() != "account not found" {
      t.Errorf("unexpected error %s", err)
    }
    w <- true
  })
  <- w

  // account [4]
  params = []interface{}{}
  removedAccount = crypto.PubkeyToAddress(accounts[4].PublicKey).String()
  params = append(params, map[string]interface{}{ "account": removedAccount })
  session.SendRPC("bundle_get", params, func(result interface{}, err error){
    if err == nil {
      t.Errorf("expect error 'account not found'")
    } else if err.Error() != "account not found" {
      t.Errorf("unexpected error for accounts[4] %s", err)
    }
    w <- true
  })
  <- w
}

func Test_RemovalFromLatestList(t *testing.T) {
  params := []interface{}{}
  w := make(chan bool)
  session.SendRPC("bundle_getRecent", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("%s", err)
    } else {
      items := result.([]interface{})
      for _, itemIface := range items {
        item := itemIface.([]interface{})
        localAccount := strings.ToLower(item[0].(string))
        nodeAccount := strings.ToLower(crypto.PubkeyToAddress(accounts[2].PublicKey).String())
        if localAccount == nodeAccount {
          t.Errorf("%s", "account[2] should not appear")
          break
        }
        nodeAccount = strings.ToLower(crypto.PubkeyToAddress(accounts[3].PublicKey).String())
        if localAccount == nodeAccount {
          t.Errorf("%s", "accounts[3] should not appear")
          break
        }
        // account[4] should not appear because it never posted anything
        nodeAccount = strings.ToLower(crypto.PubkeyToAddress(accounts[4].PublicKey).String())
        if localAccount == nodeAccount {
          t.Errorf("%s", "accounts[4] should not appear")
          break
        }

      }
    }
    w <- true
  })
  <- w
}

func Test_ReenableAccount(t *testing.T) {
  timestamp++

  invites := make([]map[string]interface{}, 0)
  accountB := crypto.PubkeyToAddress(accounts[2].PublicKey).Bytes()
  invites = append(invites, map[string]interface{}{"account": accountB, "timestamp": accountBInviteTimestamp})
  inviteKey := map[string]interface{}{"i": invites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[1])
  result, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
  if result != "ok" {
    t.Errorf("expects ok")
    return
  }
}

func Test_ConfirmAccountsReenabled(t *testing.T) {
  params := []interface{}{}
  removedAccount := crypto.PubkeyToAddress(accounts[2].PublicKey).String()
  params = append(params, map[string]interface{}{ "account": removedAccount })
  w := make(chan bool)
  session.SendRPC("bundle_get", params, func(result interface{}, err error){
    if err != nil {
      t.Error(err)
    }
    rObj, ok := result.(map[string]interface{})

    if !ok {
      t.Errorf("expects json object")
    }
    if _, bFound := rObj["multihash"]; !bFound {
      t.Errorf("expects multihash key")
    }
    w <- true
  })
  <- w

  removedAccount = crypto.PubkeyToAddress(accounts[3].PublicKey).String()
  params = append(params, map[string]interface{}{ "account": removedAccount })
  session.SendRPC("bundle_get", params, func(result interface{}, err error){
    if err != nil {
      t.Error(err)
    }
    rObj, ok := result.(map[string]interface{})
    if !ok {
      t.Errorf("expects json object")
    }
    if _, bFound := rObj["multihash"]; !bFound {
      t.Errorf("expects multihash key")
    }
    w <- true
  })
  <- w
}

func Test_CheckTimestampInMerkleheadMatchesLastPost(t *testing.T) {

  merkleheadString := testlib.GetMerklehead(session)
  //fmt.Printf("%+v\n", merkleheadString)

  doc, err := testlib.GetDoc(session, merkleheadString)
  if err != nil {
    t.Errorf("got error %+v", err)
    return
  }

  accountC := strings.ToLower(crypto.PubkeyToAddress(accounts[2].PublicKey).String())
  timestamp, success := doc[accountC[2:]]

  // should only be one
  if !success {
    t.Errorf("missing account C")
    return
  }

  if timestamp != int(accountCLastPosTimestamp) {
    t.Errorf("merklehead timestamp expects %v, got %v\n", accountCLastPosTimestamp, timestamp)
    return
  }
}

// confirm merklehead
func Test_AccountOneToSaveData(t *testing.T) {

  timestamp++

  jsonData := map[string]interface{}{"boo": "blah"}

  bundle := testlib.CreateBundle(timestamp + 10, jsonData, accounts[1])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}

func Test_CheckRecentList(t *testing.T) {
  params := []interface{}{}
  w := make(chan bool)
  session.SendRPC("bundle_getRecent", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("%s", err)
    } else {
      items := result.([]interface{})
      bFoundOne := false
      bFoundTwo := false
      for _, itemIface := range items {
        item := itemIface.([]interface{})
        localAccount := strings.ToLower(item[0].(string))
        nodeAccount := strings.ToLower(crypto.PubkeyToAddress(accounts[0].PublicKey).String())
        if localAccount == nodeAccount {
          bFoundOne = true
          continue
        }
        nodeAccount = strings.ToLower(crypto.PubkeyToAddress(accounts[1].PublicKey).String())
        if localAccount == nodeAccount {
          bFoundTwo = true
          continue
        }
      }
      if !bFoundOne || !bFoundTwo {
        fmt.Printf("%v %v\n", bFoundOne, bFoundTwo)
        t.Errorf("accounts missing from recent list")
      }
    }
    w <- true
  })
  <- w
}
