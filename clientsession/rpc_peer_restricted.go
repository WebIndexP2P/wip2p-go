package clientsession

import (
  //"log"
  "time"
)

func peerRestricted(session *Session, paramData []interface{}) (interface{}, error) {

  session.RemoteRestricted = true
  //log.Printf("Remote says restricted\n")

  go func() {
    time.Sleep(1 * time.Second)
    session.StartRecovery()
    session.Disconnect()
  }()

  return "ok", nil
}
