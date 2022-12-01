package testscript

import (
  "fmt"
  "time"
  "testing"
  "crypto/ecdsa"
  "encoding/hex"

  //"github.com/ipfs/go-ipld-cbor"
  "github.com/ipfs/go-cid"
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

func TestPostAlbumWithImageButMissingImageData(t *testing.T) {
  timestamp = uint64(time.Now().UTC().Unix())
  tmpCid, _ := cid.Parse("Qmc32mcWsm7k9gnbu1MCTN1ToARQStT4ckD7RPvZZHqmve")

  jsonData := map[string]interface{}{
      "wip2p": map[string]interface{}{
        "public": true,
      },
      "cheese": map[string]interface{}{
        "albums": []interface{}{
          map[string]interface{}{"n": "my album", "t": timestamp,
            "p": []interface{}{
              map[string]interface{}{"n":"first pic", "i": tmpCid},
            },
          },
        },
      },
    }

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}

func TestConfirmPostUsingGetRecent(t *testing.T) {
  params := []interface{}{}
  w := make(chan bool)
  session.SendRPC("bundle_getRecent", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("%s", err)
    } else {
      firstItem := result.([]interface{})[0].([]interface{})
      postBytes := firstItem[2].(float64)
      if postBytes != 108 {
        t.Errorf("bytes should equal 108 got %v", postBytes)
      }
    }
    w <- true
  })
  <- w
}

func TestPostAlbumWithImageAndData(t *testing.T) {
  timestamp += 1
  tmpCid, _ := cid.Parse("Qmc32mcWsm7k9gnbu1MCTN1ToARQStT4ckD7RPvZZHqmve")
  tmpCidData, _ := hex.DecodeString("0a0908021203626f6f1803")

  jsonData := map[string]interface{}{
      "wip2p": map[string]interface{}{
        "public": true,
      },
      "cheese": map[string]interface{}{
        "albums": []interface{}{
          map[string]interface{}{"n": "my album", "t": timestamp,
            "p": []interface{}{
              map[string]interface{}{"n":"first pic", "i": tmpCid},
            },
          },
        },
      },
    }

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  bundle.CborData = append(bundle.CborData, tmpCidData)

  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}

func TestConfirmPostWithDataUsingGetRecent(t *testing.T) {
  params := []interface{}{}
  w := make(chan bool)
  session.SendRPC("bundle_getRecent", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("%s", err)
    } else {
      firstItem := result.([]interface{})[0].([]interface{})
      postBytes := firstItem[2].(float64)
      if postBytes != 119 {
        t.Errorf("bytes should equal 119 got %v", postBytes)
      }
    }
    w <- true
  })
  <- w
}

func TestFetchImageData(t *testing.T) {
  params := []interface{}{"Qmc32mcWsm7k9gnbu1MCTN1ToARQStT4ckD7RPvZZHqmve", "base64"}
  w := make(chan bool)
  session.SendRPC("doc_get", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("%s", err)
    } else {
      if result != "CgkIAhIDYm9vGAM=" {
        t.Errorf("bytes not as expected")
      }
    }
    w <- true
  })
  <- w
}

func TestPostAlbumPointingToExistingImage(t *testing.T) {
  timestamp += 1
  tmpCid, _ := cid.Parse("Qmc32mcWsm7k9gnbu1MCTN1ToARQStT4ckD7RPvZZHqmve")
  //tmpCidData, _ := hex.DecodeString("0a0908021203626f6f1803")

  jsonData := map[string]interface{}{
      "wip2p": map[string]interface{}{
        "public": true,
      },
      "cheese": map[string]interface{}{
        "albums": []interface{}{
          map[string]interface{}{"n": "my album", "t": timestamp,
            "p": []interface{}{
              map[string]interface{}{"n":"first pic", "i": tmpCid},
            },
          },
        },
      },
    }

  bundle := testlib.CreateBundle(timestamp, jsonData, accounts[0])
  _, err := testlib.Post(session, bundle)
  if err != nil {
    t.Errorf("%s", err)
    return
  }
}

func TestFetchImageDataAgain(t *testing.T) {
  params := []interface{}{"Qmc32mcWsm7k9gnbu1MCTN1ToARQStT4ckD7RPvZZHqmve", "base64"}
  w := make(chan bool)
  session.SendRPC("doc_get", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("%s", err)
    } else {
      if result != "CgkIAhIDYm9vGAM=" {
        t.Errorf("bytes not as expected")
      }
    }
    w <- true
  })
  <- w
}

func TestConfirmPostWithDataUsingGetRecent2(t *testing.T) {
  params := []interface{}{}
  w := make(chan bool)
  session.SendRPC("bundle_getRecent", params, func(result interface{}, err error){
    if err != nil {
      t.Errorf("%s", err)
    } else {
      firstItem := result.([]interface{})[0].([]interface{})
      postBytes := firstItem[2].(float64)
      if postBytes != 119 {
        t.Errorf("bytes should equal 119 got %v", postBytes)
      }
    }
    w <- true
  })
  <- w
}
