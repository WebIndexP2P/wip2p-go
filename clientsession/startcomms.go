package clientsession

import (
  "fmt"
  "log"
  "errors"

  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)


func (s *Session) StartComms() {
  // this blocks
  // the peerId of this node will be assigned as root
  s.StartAuthProcess()
  if !s.HasAuthed {
    if globals.DebugLogging {
      fmt.Printf("HasAuthed = false, something gone wrong\n")
    }
    return
  }

  if s.LocalRestricted && s.RemoteRestricted {
    s.LastError = errors.New("neither account has invites")
    log.Printf("%s\n", s.LastError)
    s.Ws.Close()
    return
  }

  // let the other peer know they are restricted, dont proceed any further
  if s.LocalRestricted {
    c := make(chan error)
    params := make([]interface{}, 0)
    err := s.SendRPC("peer_restricted", params, func(result interface{}, err error){
      c <- nil
    })
    err = <- c
    if err != nil {
      log.Println("peer_restricted failed")
    }
    return
  }

  if s.RemoteRestricted {
    log.Printf("sending account recovery bundle to remote account\n")
    err := s.StartRecovery()
    if err != nil {
      s.LastError = err
    }
    s.Ws.Close()
    return
  }

  // do info swap, save peer to db
  _, err := s.StartInfo()
  if err != nil {
    if s.OnError != nil {
      s.OnError(err)
    }
    s.LastError = err
    s.Ws.Close()
    return
  }

  // do peer swap, save peer to db
  err = s.StartPeerSwap()
  if err != nil {
    log.Printf("%+v\n", err)
    return
  }

  /*if info.Version[:12] == "wip2p-go 0.4" {
    log.Printf("Cant sync with 0.4.x node\n")
    return
  }*/

  // start the sync
  s.StartSync()
  if !s.HasSynced {
    log.Printf("waiting for remote %+v to complete sync\n", s.RemotePeerId.String())
    return
  }

  s.StartLinkedDocsSwap()
}
