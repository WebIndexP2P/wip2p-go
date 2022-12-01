package peermanager

import (
  //"fmt"
  "log"
  "github.com/gorilla/websocket"

  "code.wip2p.com/mwadmin/wip2p-go/clientsession"
)

type broadcastStruct struct {
  session *clientsession.Session
  paramData []interface{}
}

var connections = make(map[*websocket.Conn]*clientsession.Session)
var broadcast = make(chan broadcastStruct)

func StopAllSessions() {
    for client := range connections {
      client.Close()
    }
}

func Broadcaster() {
  for {
    //fmt.Printf("%+v\n", connections)

    broadcastDetails := <- broadcast
    // send to every client that is currently connected
    for client := range connections {

      // dont sent it back to the person who sent it to us
      if client == broadcastDetails.session.Ws {
        continue
      }

      session := connections[client]

      if !session.HasAuthed || session.LocalRestricted {
        continue
      }

      log.Printf("Broadcasting new paste to %s", session.RemoteEndpoint)
      err := session.SendRPC("bundle_save", broadcastDetails.paramData, nil)

      if err != nil {
        log.Printf("Websocket error: %s", err)
        client.Close()
        delete(connections, client)
      }
    }
  }
}

func AddSession(newSession *clientsession.Session) {
  connections[newSession.Ws] = newSession
}

func RemoveSession(session *clientsession.Session) {
  delete(connections, session.Ws)
}
