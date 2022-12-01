package testscript

import (
  "fmt"
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

func init() {

  clientsession.OnBroadcast = func(session *clientsession.Session, data []interface{}) {
    fmt.Printf("broadcast\n")
  }

  clientsession.OnAuthed = func(session *clientsession.Session) bool {
    fmt.Printf("onauthed\n")
    return true
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

func TestCreateRoot(t *testing.T) {

  timestamp = uint64(time.Now().UTC().Unix()) + 1

  // first account to post with invites for remaining 16
  publicFlag := map[string]interface{}{"public": false}
  jsonData := map[string]interface{}{"wip2p": publicFlag, "test": "blah"}

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

}

func TestRootAccount(t *testing.T) {
	params := []interface{}{}
  rootAccount := crypto.PubkeyToAddress(accounts[0].PublicKey).String()
  params = append(params, rootAccount)
  //params = append(params, map[string]interface{}{"includePaste": true, "includeInvites": true})
	w := make(chan bool)
	session.SendRPC("ui_getAccount", params, func(result interface{}, err error){
	  if err != nil {
		  t.Errorf("got error %+v", err)
    }
    res := result.(map[string]interface{})
    if res["activeLevel"].(float64) != 0 {
      t.Errorf("expects activeLevel 0, got %+v", res["activeLevel"])
    }
    if res["activeTimestamp"].(float64) != 1 {
      t.Errorf("expects activeTimestamp 1, got %+v", res["activeTimestamp"])
    }
    if res["postcount"].(float64) != 1 {
      t.Errorf("expects postcount 1, got %+v", res["postcount"])
    }
	  w <- true
	})
	<- w
}

func TestPublicFlag(t *testing.T) {

  session.Disconnect()

	globals.NodePrivateKey = accounts[1]
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

  session.StartAuthProcess()

  if !session.HasAuthed {
    fmt.Printf("HasAuthed = false, something gone wrong")
    return
  }

}
