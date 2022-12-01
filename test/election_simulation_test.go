package testscript

import (
  "log"
  "fmt"
  "time"
  "testing"
  "crypto/ecdsa"
  "math/rand"
  "github.com/ethereum/go-ethereum/crypto"

  "code.wip2p.com/mwadmin/wip2p-go/clientsession"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/test/lib"
  "code.wip2p.com/mwadmin/wip2p-go/util"
)

var rootAccount *ecdsa.PrivateKey
//var accounts []*ecdsa.PrivateKey
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

  // set root Account
  tmpKey := make([]byte, 32)
  intBytes := util.Itob(uint64(1))
  copy(tmpKey[24:32], intBytes)
  var err error
  rootAccount, err = crypto.ToECDSA(tmpKey)
  if err != nil {
    log.Fatal(err)
  }

  //globals.DebugLogging = true
  globals.NodePrivateKey = rootAccount
  globals.PublicMode = true

  // establish encrypted session with peer
  session = clientsession.Create()
  session.RemoteEndpoint = "ws://127.0.0.1:9472"
  session.OnError = func(err error) {
    fmt.Printf(err.Error() + "\n")
  }

  err = session.Dial()
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

func TestCreateInviters(t *testing.T) {

  timestamp = uint64(time.Now().UTC().Unix())
  accounts := make([]*ecdsa.PrivateKey, 0)

  invites = make([]map[string]interface{}, 0)
  for a := 0; a < 100; a++ {
    tmpKey := make([]byte, 32)
    intBytes := util.Itob(uint64(a+2))
    copy(tmpKey[24:32], intBytes)
    account, err := crypto.ToECDSA(tmpKey)
    accounts = append(accounts, account)
    if err != nil {
      log.Fatal(err)
    }

    accountB := crypto.PubkeyToAddress(account.PublicKey).Bytes()
    invites = append(invites, map[string]interface{}{"account": accountB, "timestamp": timestamp})
  }
  inviteKey := map[string]interface{}{"i": invites, "public": true}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, rootAccount)
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

  // advance the timestamp by one
  timestamp = timestamp + 1

  for idx, account := range accounts {
    recursiveInvite(idx, account, t)
  }

}

func recursiveInvite(offset int, k *ecdsa.PrivateKey, t *testing.T) {

  accounts := make([]*ecdsa.PrivateKey, 0)

  invites = make([]map[string]interface{}, 0)
  for a := 0; a < 1000; a++ {
    tmpKey := make([]byte, 32)
    keyNum := uint64((a + (offset * 1000) + 1002))
    if keyNum % 10000 == 0 {
      fmt.Printf("processed = %+v\n", keyNum)
    }
    intBytes := util.Itob(keyNum)
    copy(tmpKey[24:32], intBytes)
    account, err := crypto.ToECDSA(tmpKey)
    accounts = append(accounts, account)
    if err != nil {
      log.Fatal(err)
    }

    accountB := crypto.PubkeyToAddress(account.PublicKey).Bytes()
    invites = append(invites, map[string]interface{}{"account": accountB, "timestamp": timestamp})
  }
  inviteKey := map[string]interface{}{"i": invites}
  jsonData := map[string]interface{}{"wip2p": inviteKey}

  bundle := testlib.CreateBundle(timestamp, jsonData, k)
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }

  // advance the timestamp
  if uint64(time.Now().UTC().Unix()) > timestamp {
    timestamp = uint64(time.Now().UTC().Unix())
  } else {
    timestamp = timestamp + 1
  }

  // now each account will publish their vote
  for _, account := range accounts {
    publishVote(account, t)
  }
}

func publishVote(account *ecdsa.PrivateKey, t *testing.T) {

  candidates := []string{"Bob Paulson [REP]", "Paul Bobson [DEM]"}

  rand.Seed(time.Now().UnixNano())
  rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })

  houseVotes := candidates
  senateVotes := make([]string, 0)

  vote := make(map[string]interface{})
  vote["e"] = "ny"
  vote["d"] = "2022-05-14"
  vote["h"] = houseVotes
  vote["s"] = senateVotes
  namespace := make(map[string]interface{})
  namespace["election"] = vote

  bundle := testlib.CreateBundle(timestamp, namespace, account)
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}
