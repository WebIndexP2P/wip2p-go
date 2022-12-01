package core

import (
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)

type CallbackIface interface {
    Forward(msg string)
}

var AndroidCallback CallbackIface

// save the callback instance
func RegisterCallback(c CallbackIface) {
    AndroidCallback = c
    globals.AndroidCallback = c.Forward
}
