package clientsession

import (
  //"fmt"
  "errors"
  "encoding/hex"
)

type helloResponseStruct struct {
  Version string
  Endpoints []string
  RootAccount string
  Merklehead string
  SeqNo uint
}

func peerHello(session *Session, paramData []interface{}) (interface{}, error) {

    response := map[string]interface{}{}

    if len(paramData) == 0 {
      return nil, errors.New("missing public key")
    }

    paramObj, ok := paramData[0].(map[string]interface{})
    if !ok {
      return nil, errors.New("missing public key")
    }

    remotePubKeyAuth, ok := paramObj["pubKeyAuth"].(string)

    if !ok {
      return nil, errors.New("missing public key")
    }

    pubKeyBytes, err := hex.DecodeString(remotePubKeyAuth[2:])
    if err != nil {
      return nil, errors.New("invalid public key")
    }
    copy(session.RemoteNaclPublicKey[:], pubKeyBytes)
    session.HaveRemotePublicKey = true

    pubKeyString := hex.EncodeToString(session.LocalNaclKeyPair.PublicKey[:])

    response["pubKeyAuth"] = "0x" + pubKeyString
    return response, nil
}
