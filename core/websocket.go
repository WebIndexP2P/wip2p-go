package core

import (
  "log"
  "net/http"
  "github.com/gorilla/websocket"

  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/clientsession"
  "code.wip2p.com/mwadmin/wip2p-go/peermanager"
)

func serveWs(w http.ResponseWriter, r *http.Request) {

  var upgrader = websocket.Upgrader{
     CheckOrigin: func(r *http.Request) bool {
        return true
     },
  }

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		//log.Print("upgrade:", err)
		return
	}

  newSession := clientsession.Create()
  newSession.Ws = c
  newSession.LocalPrivateKey = globals.NodePrivateKey

  forwardHeaderString := "x-forwarded-for";
  remoteAddr := r.Header.Get(forwardHeaderString)
  if len(remoteAddr) > 0 {
    newSession.SetRemoteEndpoint(remoteAddr)
  } else {
    newSession.SetRemoteEndpoint(c.RemoteAddr().String())
  }

  log.Println("connection from", newSession.RemoteEndpoint)
  if globals.AndroidCallback != nil {
    globals.AndroidCallback("connection from " + newSession.RemoteEndpoint)
  }

  peermanager.AddSession(newSession)
  newSession.HandleMessages()
}
