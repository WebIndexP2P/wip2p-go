package clientsession

import (
  "log"
  //"fmt"
  "bytes"
  "errors"
  "strconv"
  "encoding/hex"
  "github.com/ethereum/go-ethereum/crypto"
  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/signature"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/account"
)

func (s *Session) StartAuthProcess() {

  c := make(chan error)

  s.StartHello(c)
  err := <- c

  if err == nil {
    s.StartAuth(c)
    err = <- c
    if err != nil {
      s.LastError = err
      //log.Printf("%+v\n", err)
      s.Disconnect()
    }
  }

}

func (s *Session) StartHello(c chan error) {

  // add our ephemeral nacl publicKey
  params := make([]interface{}, 1)
  param1 := map[string]interface{}{}
  pubKeyString := hex.EncodeToString(s.LocalNaclKeyPair.PublicKey[:])
  param1["pubKeyAuth"] = "0x" + pubKeyString
  params[0] = param1

  err := s.SendRPC("peer_hello", params, func(result interface{}, err error){

    r, _ := result.(map[string]interface{})
    if r == nil {
      err := errors.New("peer_hello returned invalid response")
      c <- err
      log.Println(err)
      s.OnError(err)
      return
    }

    pubKeyAuth, ok := r["pubKeyAuth"].(string)
    if !ok {
      err := errors.New("missing pubKeyAuth")
      c <- err
      log.Println(err)
      s.OnError(err)
      return
    }

    pubKeyB, err := hex.DecodeString(pubKeyAuth[2:])
    copy(s.RemoteNaclPublicKey[:], pubKeyB)
    s.HaveRemotePublicKey = true

    c <- nil
    //end callback
  })

  if err != nil {
    err := errors.New("peer_hello failed")
    s.OnError(err)
    return
  }
}

func (s *Session) StartAuth(c chan error) {

  // sign our nacl public key
  signedString := "0x" + hex.EncodeToString(s.LocalNaclKeyPair.PublicKey[:])
  signedString = "\x19Ethereum Signed Message:\n" + strconv.Itoa(len(signedString)) + signedString
  signedBytes := []byte(signedString)
  hash := crypto.Keccak256Hash(signedBytes)
  localSigB, err := crypto.Sign(hash.Bytes(), s.LocalPrivateKey)

  // convert v to normalized for solidity
  localSigB[64] = localSigB[64] + 27

  params := make([]interface{}, 0)
  param := map[string]interface{}{}
  param["signature"] = "0x" + hex.EncodeToString(localSigB)
  params = append(params, param)

  // send the request
  err = s.SendRPC("peer_auth", params, func(result interface{}, err error){

    if err != nil {
      c <- err
      return
    }

    remoteSigParam, ok := result.(map[string]interface{})
    if !ok {
      c <- errors.New("result expects object")
      return
    }

    errIface, found := remoteSigParam["error"]
    if found {
      log.Printf("%+v\n", errIface.(string))
      c <- errors.New(errIface.(string))
      return
    }

    // verify received signature
    remoteHash, _ := signature.EthHashBytes(s.RemoteNaclPublicKey[:])
    remoteSigB, _ := hex.DecodeString(remoteSigParam["signature"].(string)[2:])
    recAddressB, _ := signature.Recover(remoteHash, remoteSigB)

    if !globals.PublicMode {
      if bytes.Equal(globals.RootAccount[:], recAddressB) {
        // looks like root, allow it
      } else {
        _, success := account.FetchAccountFromDb(common.BytesToAddress(recAddressB), nil, false)
        if !success {
          // basically we need to be able to connect to a private tree and download the content to see if we are included in the allowed accounts
          // so rather than disconnect, we flag as "restricted" until our status can be determined
          // we dont want to send private data to this peer unless we are a member
          s.LocalRestricted = true
        }
      }
    }

    restrictedIface, ok := remoteSigParam["restricted"]
    if ok {
      s.RemoteRestricted = restrictedIface.(bool)
      //log.Printf("Remote says restricted\n")
    }

    s.RemotePeerId = common.BytesToAddress(recAddressB)

    wasAdded := OnAuthed(s)
    if !wasAdded {
      log.Println("duplicate session for " + s.RemotePeerId.String())
      c <- errors.New("peer already has another session")
      s.Ws.Close()
      return
    }

    s.HasAuthed = true

    restrictedText := ""
    if s.LocalRestricted {
      restrictedText = " (restricted)"
    }
    log.Println(s.RemoteEndpoint + " authed as " + s.RemotePeerId.String() + restrictedText)

    c <- nil
  })

  if err != nil {
    log.Println("peer_auth failed")
    c <- err
  }
}
