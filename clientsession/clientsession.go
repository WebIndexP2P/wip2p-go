package clientsession

import (
  "io"
  //"fmt"
  "log"
  "sync"
  "errors"
  "encoding/json"
  "crypto/ecdsa"
  "crypto/rand"

  "golang.org/x/crypto/nacl/box"
  "github.com/gorilla/websocket"
  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/util"
  "code.wip2p.com/mwadmin/wip2p-go/peer"
  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)

var OnConnect func(*Session)
var OnEnd func(*Session)
var OnBroadcast func(*Session, []interface{})
var OnAuthed func(*Session) bool
var OnPeerSwap func(*Session, []string)

type naclKeyPair struct {
  PrivateKey [32]byte
  PublicKey [32]byte
}

type Session struct {
  Ws *websocket.Conn
  LastRequestId int
  mu sync.Mutex
  LastError error

  LocalPrivateKey *ecdsa.PrivateKey
  LocalNaclKeyPair naclKeyPair

  ExpectedPeerId common.Address

  RemoteEndpoint string
  RemotePeerId common.Address
  RemoteNaclPublicKey [32]byte
  RemoteMerklehead []byte
  LastSyncSequenceNo uint
  NetworkSegment string

  HasAuthed bool
  WeInitiated bool
  HaveRemotePublicKey bool
  RemoteRestricted bool
  LocalRestricted bool
  HasSynced bool

  RemoteWantsAccountContentRemovals bool
  DontValidateAndSave bool

  PendingRequests map[uint]func(interface{}, error) // RPC requests we have made and waiting for a response
  //OnAuth func()
  OnError func(error)
}

func Create() *Session {

  if OnBroadcast == nil {
    log.Fatal("OnBroadcast is not set")
  }

  s := Session{}
  s.PendingRequests = make(map[uint]func(interface{}, error), 0)
  s.initAuthKeys()
  return &s
}

func (s *Session) initAuthKeys() {
  localAuthPubKey, localAuthPrivateKey, err := box.GenerateKey(rand.Reader)

  s.LocalPrivateKey = globals.NodePrivateKey
  s.LocalNaclKeyPair.PublicKey = *localAuthPubKey
  s.LocalNaclKeyPair.PrivateKey = *localAuthPrivateKey

  if err != nil {
    log.Fatal(err)
  }
}

func (s *Session) SetRemoteEndpoint(endpoint string) {
  s.RemoteEndpoint = endpoint
  ep := peer.ParseEndpointString(endpoint)
  s.NetworkSegment = util.GetNetworkSegment(ep.Host)
}

func (s *Session) Dial() error {
  //log.Printf("Dial %+v\n", s)
  s.WeInitiated = true
  c, _, err := websocket.DefaultDialer.Dial(s.RemoteEndpoint, nil)
  if err != nil {

    if s.ExpectedPeerId.String() != "0x0000000000000000000000000000000000000000" {

      tx := db.GetTx(true)
      tmpPeer, _ := peer.FetchFromDb(s.ExpectedPeerId, tx)
      tmpPeer.UpdateLastConnectAttempt(s.RemoteEndpoint)
      tmpPeer.EndpointFailure(s.RemoteEndpoint)
      tmpPeer.SaveToDb(tx)
      //log.Printf("%+v\n", tmpPeer)
      db.CommitTx(tx)
    }

    OnEnd(s)
    return err
  }
  log.Printf("connected to %v\n", s.RemoteEndpoint)
  if globals.AndroidCallback != nil {
    globals.AndroidCallback("connected to " + s.RemoteEndpoint)
  }
  s.Ws = c

  // handle incoming messages
  go s.HandleMessages()

  // we initiated, so start the auth process
  //s.startAuth()
  return nil
}

func (s *Session) Disconnect() error {
  log.Printf("closing connection to %v\n", s.RemoteEndpoint)
  s.Ws.Close()
  return nil
}

func (s *Session) HandleMessages() {
  var message []byte
  var err error
  for {
    _, message, err = s.Ws.ReadMessage()
    if err != nil {
      //log.Println("read error:", err)
      log.Println("removing connection", s.RemoteEndpoint)
      if OnEnd != nil {
        OnEnd(s)
      }
      break
    }

    if s.HaveRemotePublicKey {
      decryptedMessage, success := s.Decrypt(message)
      if !success {
        s.SendMessage([]byte("{\"error\":\"problem with encryption\"}"), false)
        return
      }
      s.ParseDecrypted(decryptedMessage)
    } else {
      s.ParseDecrypted(message)
    }
  }
}

func (s *Session) ParseDecrypted(message []byte) {

  if globals.DebugLogging {
    log.Printf("incoming message: %+v\n", string(message))
  }

  var response = make(map[string]interface{})
  var jsonData map[string]interface{}
  var result interface{}

  encrypt := true
  preReadyError := false

  var err error

  bShouldRespond := func() bool {
    err = json.Unmarshal(message, &jsonData)
    if err != nil {
      err = errors.New("invalid json rpc request")
      encrypt = false
      return true
    }

    // its valid json
    if jsonData["id"] == nil {
      err = errors.New("missing id")
      return true
    }

    _, hasResult := jsonData["result"]
    if hasResult || jsonData["error"] != nil {
      s.ProcessRPCResult(jsonData)
      // we processed a response, no need to respond
      return false
    }

    if jsonData["method"] == nil {
      err = errors.New("missing method or result")
      return true
    }

    response["id"] = jsonData["id"]
    paramData, ok := jsonData["params"].([]interface{})
    if !ok {
      err = errors.New("params expects array")
      return true
    }

    method := jsonData["method"]
    if !s.HasAuthed && (method != "peer_hello" && method != "peer_auth" && method != "ui_requestInvite") {
      err = errors.New("unauthorized")
      return true
    }

    if s.LocalRestricted && (method != "peer_info" && method != "bundle_save" && method != "peer_restricted" && method != "ui_requestInvite") {
      err = errors.New("unauthorized")
      return true
    }

    switch jsonData["method"] {
    case "peer_auth":
      result, err = peerAuth(s, paramData)
      if err != nil {
        preReadyError = true
      }
    case "paste_save", "bundle_save":
      var seqNo uint = 0
      var accountsWithContentRemoved []string

      if s.DontValidateAndSave == false {
        seqNo, accountsWithContentRemoved, err = bundleSave(s, paramData)
      }
      paramArray, success1 := jsonData["params"].([]interface{})
      paramData, success2 := paramArray[0].(map[string]interface{})
      if err == nil && success1 && success2 {
        if seqNo > 0 {
          paramData["seqNo"] = seqNo
        }
        if seqNo != globals.LatestSequenceNo {
          paramData["latestSeqNo"] = globals.LatestSequenceNo
        }
        if s.RemoteWantsAccountContentRemovals {
          paramData["accountsContentRemoved"] = accountsWithContentRemoved
        }

        OnBroadcast(s, paramArray)
      }
    case "paste_get", "bundle_get":
      result, err = bundleGet(paramData, nil)
    case "doc_get":
      result, err = docGet(paramData)
    case "ui_getAccount":
      result, err = ui_getAccount(s, paramData)
    case "paste_getBySequence", "bundle_getBySequence":
      result, err = bundleGetBySequence(paramData)
    case "paste_getRecent", "bundle_getRecent":
      result, err = bundleGetRecent(s, paramData)
    case "peer_hello":
      result, err = peerHello(s, paramData)
      encrypt = false
      if err != nil {
        preReadyError = true
      }
    case "peer_swap":
      result, err = peerSwap(s, paramData)
    case "peer_syncFinished":
      result, err = peerSyncFinished(s, paramData)
    case "peer_info":
      result, err = peerInfo(s, paramData)
    case "peer_restricted":
      result, err = peerRestricted(s, paramData)
    case "ui_getTimestampsBatch":
      result, err = ui_getTimestampsBatch(s, paramData)
    case "ui_requestInvite":
      result, err = ui_requestInvite(s, paramData)
    default:
      err = errors.New("unknown method")
    }
    return true
  }()

  if !bShouldRespond {
    return
  }

  if err != nil {
    response["error"] = err.Error()
  } else {
    if result == nil {
      response["result"] = "ok"
    } else {
      response["result"] = result
    }
  }

  bytes, _ := json.Marshal(response)

  err = s.SendMessage(bytes, encrypt)
  if err != nil && globals.DebugLogging {
    log.Printf("Error sending to websocket: %s\n", err)
  }

  if preReadyError {
    s.Ws.Close()
  }
}

func (s *Session) ProcessRPCResult(jsonData map[string]interface{}) {
  id := uint(jsonData["id"].(float64))
  results, _ := jsonData["result"]
  errIface, _ := jsonData["error"]

  var err error

  if errIface != nil {
    err = errors.New(errIface.(string))
  }

  callback := s.PendingRequests[id]
  delete(s.PendingRequests, id)

  if callback != nil {
    callback(results, err)
  }
}

func (s *Session) Decrypt(msg []byte) ([]byte, bool) {

  if len(msg) <= 24 {
    return nil, false
  }

  var nonce [24]byte
  copy(nonce[:], msg[:24])
  incomingBox := msg[24:]

  data, success := box.Open(nil, incomingBox, &nonce, &s.RemoteNaclPublicKey, &s.LocalNaclKeyPair.PrivateKey)

  return data, success
}

func (s *Session) Encrypt(msg []byte) ([]byte, error) {

  if !s.HaveRemotePublicKey {
    return nil, errors.New("no remote public key, cannot encrypt")
  }

  var nonce [24]byte
  if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
    log.Println("error with Encrypt...")
    panic(err)
  }

  enc := box.Seal(nonce[:], msg, &nonce, &s.RemoteNaclPublicKey, &s.LocalNaclKeyPair.PrivateKey)

  return enc, nil
}


func (s *Session) SendMessage(msg []byte, encrypt bool) error {

  if globals.DebugLogging {
    log.Printf("outgoing message: %+v\n", string(msg))
  }

  // we need to create a lock here to make sure we dont concurrent write
  // a scenario where this does happen is when a new paste arrives.. e.g.
  //  * new sigbundle arrives from peer A, we save it to the db
  //  * we start to prepare to send the new sigbundle to peer B
  //  * peer B also sends us the same sigbundle
  //  * we reply to peer B with "timestamp not newer" and "bundle_save" at the same time

  s.mu.Lock()
  defer s.mu.Unlock()

  if encrypt {
    encData, err := s.Encrypt(msg)
    if err != nil {
      return err
    }
    return s.Ws.WriteMessage(websocket.BinaryMessage, encData)
  } else {
    return s.Ws.WriteMessage(websocket.BinaryMessage, msg)
  }
}

func (s *Session) SendRPC(method string, paramData []interface{}, handleResponse func(interface{}, error)) error {

  s.LastRequestId++
  id := s.LastRequestId
  request := make(map[string]interface{})
  request["id"] = id
  request["method"] = method
  request["params"] = paramData

  reqBytes, _ := json.Marshal(request);
  if handleResponse != nil {
    s.PendingRequests[uint(id)] = handleResponse
  }

  encrypt := true
  if !s.HaveRemotePublicKey {
    encrypt = false
  }

  return s.SendMessage(reqBytes, encrypt)
}
