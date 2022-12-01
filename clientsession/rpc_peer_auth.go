package clientsession

import (
  //"fmt"
  "log"
  "bytes"
  "errors"
  "strconv"
  "encoding/hex"
  "github.com/ethereum/go-ethereum/crypto"
  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/account"
  "code.wip2p.com/mwadmin/wip2p-go/db"
  //"code.wip2p.com/mwadmin/wip2p-go/util"
  "code.wip2p.com/mwadmin/wip2p-go/signature"
  "code.wip2p.com/mwadmin/wip2p-go/invite"
  "code.wip2p.com/mwadmin/wip2p-go/autoinvites"
)

func peerAuth(session *Session, paramData []interface{}) (interface{}, error){

  params, ok := paramData[0].(map[string]interface{})
  if !ok {
    return nil, errors.New("param[0] expects object")
  }

  response := map[string]interface{}{}

  //if session.WeInitiated {
  sigIface, ok := params["signature"]
  if !ok {
    return nil, errors.New("missing signature")
  }

  restrictedIface, ok := params["restricted"]
  if ok {
    session.RemoteRestricted = restrictedIface.(bool)
    //log.Printf("Remote says restricted\n")
  }

  // verify received signature
  remoteHash, _ := signature.EthHashBytes(session.RemoteNaclPublicKey[:])
  remoteSigB, _ := hex.DecodeString(sigIface.(string)[2:])
  recAddressB, _ := signature.Recover(remoteHash, remoteSigB)

  session.RemotePeerId = common.BytesToAddress(recAddressB)

  // if in private mode, verify the account exists

  //if util.AllZero(globals.RootAccount[:]) {
  if globals.NextAuthIsRoot {
    copy(globals.RootAccount[:], recAddressB)
    tx := db.GetTx(true)
    doc := db.Config{}
    doc.Write("rootAccount", globals.RootAccount[:], tx)

    inviters := make([]invite.Inviter, 0)
    inviters = append(inviters, invite.Inviter{Timestamp: uint64(1)})
    rootAccount := account.AccountStruct{Address: session.RemotePeerId, Inviters: inviters, ActiveInviter: inviters[0], Enabled: true}
    rootAccount.SaveToDb(tx)

    globals.NextAuthIsRoot = false;
    conf := db.Config{}
    conf.Delete("nextAuthIsRoot", tx)

    db.CommitTx(tx)
    log.Printf("rootAccount set to 0x%v\n", hex.EncodeToString(globals.RootAccount[:]))
  } else if globals.NextAuthGetsInvite {

    tx := db.GetTx(true)
    tmpInvite := autoinvites.Invite{}
    autoinvites.CreateInvite(session.RemotePeerId, tmpInvite, tx)

    globals.NextAuthGetsInvite = false;
    conf := db.Config{}
    conf.Delete("nextAuthGetsInvite", tx)

    db.CommitTx(tx)

  } else if globals.GetRootFromNextPeer {

  } else {
    if !globals.PublicMode {
      if bytes.Equal(globals.RootAccount[:], recAddressB) {
        // looks like root, allow it
      } else {
        _, success := account.FetchAccountFromDb(session.RemotePeerId, nil, false)
        if !success {
          //log.Println("Unauthorized session from " + session.RemotePeerId.String())
          //err := errors.New("account not authorized")
          //session.LastError = err
          //return nil, err
          log.Println("session from unknown account " + session.RemotePeerId.String() + " (restricted)")
          response["restricted"] = true
          session.LocalRestricted = true
        }
      }
    }
  }

  wasAdded := OnAuthed(session)
  if !wasAdded {
    log.Println("duplicate session for " + session.RemotePeerId.String())
    return nil, errors.New("peer already has another session")
  }

  session.HasAuthed = true

  log.Println(session.RemoteEndpoint + " authed as " + session.RemotePeerId.String())

  // Now sign our nacl pubkey
  signedString := "0x" + hex.EncodeToString(session.LocalNaclKeyPair.PublicKey[:])
  signedString = "\x19Ethereum Signed Message:\n" + strconv.Itoa(len(signedString)) + signedString
  signedBytes := []byte(signedString)
	hash := crypto.Keccak256Hash(signedBytes)

  localSigB, err := crypto.Sign(hash.Bytes(), session.LocalPrivateKey)
  if err != nil {
    log.Fatal(err)
	}

  // convert v to normalized for solidity
  localSigB[64] = localSigB[64] + 27

  response["signature"] = "0x" + hex.EncodeToString(localSigB)

  return response, nil
}
