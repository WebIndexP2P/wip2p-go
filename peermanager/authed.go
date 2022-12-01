package peermanager

import (
  "log"
  "sync"
  "strconv"
  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)

var authedPeers map[common.Address]bool

func AddAuthedPeer(address common.Address) bool {
  var mux sync.Mutex
  var bSuccess bool
  mux.Lock()
  _, ok := authedPeers[address]
  if !ok {
    if globals.DebugLogging {
      log.Printf("Added authed peer %s", address.String())
    }
    authedPeers[address] = true
    bSuccess = true
    if globals.AndroidCallback != nil {
      globals.AndroidCallback("peer count: " + strconv.Itoa(len(authedPeers)))
    }
  }
  mux.Unlock()
  return bSuccess
}

func RemoveAuthedPeer(address common.Address) {
  var mux sync.Mutex
  mux.Lock()
  _, ok := authedPeers[address]

  if ok {
    if globals.DebugLogging {
      log.Printf("Removing authed peer %s", address.String())
    }
    delete(authedPeers, address)
  } else {
    panic("authedPeers missing peer id")
  }
  mux.Unlock()
}
