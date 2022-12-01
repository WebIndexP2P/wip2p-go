package core

import (
   "os"
   "fmt"
   "log"
   "time"
   "context"
   "net/http"

   "code.wip2p.com/mwadmin/wip2p-go/core/globals"
   "code.wip2p.com/mwadmin/wip2p-go/db"
   "code.wip2p.com/mwadmin/wip2p-go/peermanager"
   "code.wip2p.com/mwadmin/wip2p-go/public"
   "code.wip2p.com/mwadmin/wip2p-go/merklehead"
   "code.wip2p.com/mwadmin/wip2p-go/autoinvites"
   "code.wip2p.com/mwadmin/wip2p-go/deltalog"
   "code.wip2p.com/mwadmin/wip2p-go/names"
)

var httpServeHandler *http.Server

func Start() {

  var heading = "==================================\n"
  heading += "==       wip2p-go v" + globals.AppVersion + "        ==\n"
  heading += "=================================="
  fmt.Println(heading)
  if globals.AndroidCallback != nil {
    globals.AndroidCallback(heading)
  }

  var dbPath string
  if db.IsAndroidLib() {
    dbPath = db.GetDefaultPath()
  }
  dbPath += GetArgs().Db

  globals.FirstRun = db.IsFirstRun(dbPath)

  bShouldExit, err := ProcessArgs(1)
  if err != nil {
    log.Fatal(err)
  }
  if bShouldExit {
    os.Exit(0)
  }

  err = db.DbInit(dbPath)
  if err != nil {
    log.Fatal(err)
  }

  deltalog.Init(nil)
  names.Init(nil)

  bShouldExit, err = ProcessArgs(2)
  if err != nil {
    log.Fatal(err)
  }
  if bShouldExit {
    os.Exit(0)
  }

  // init the sequence values
  conf := db.Config{}
  globals.SequenceSeed = conf.ReadUint("sequenceSeed", nil)
  if globals.SequenceSeed == 0 {
    globals.SequenceSeed = uint(time.Now().UTC().Unix())
    conf := db.Config{}
    conf.WriteUint("sequenceSeed", globals.SequenceSeed, nil)
    globals.LatestSequenceNo = 0
  } else {
    // get latest sequenceNo from db
    nextSeq := deltalog.GetDeltaLogSequence()
    globals.LatestSequenceNo = uint(nextSeq) - 1
  }
  fmt.Printf("Sequence Seed = %v\n", globals.SequenceSeed)
  fmt.Printf("LatestSequenceNo = %v\n", globals.LatestSequenceNo)

  merklehead.Init()
  peermanager.Init(GetArgs().BootArray)

  // import the NextAuths
  nextAuthIsRoot := conf.Read("nextAuthIsRoot", nil)
  if nextAuthIsRoot != nil && nextAuthIsRoot[0] == 1 {
    globals.NextAuthIsRoot = true
  }
  nextAuthGetsInvite := conf.Read("nextAuthGetsInvite", nil)
  if nextAuthGetsInvite != nil && nextAuthGetsInvite[0] == 1 {
    globals.NextAuthGetsInvite = true
  }

  globals.NodePrivateKey = InitNodeKey()
  copy(globals.RootAccount[:], InitRootAccount())

  // determine PublicMode from rootAccount
  globals.PublicMode = public.GetModeFromDb()
  if globals.PublicMode {
    fmt.Println("Set access to \"public\" mode")
  } else {
    fmt.Println("Set access to \"private\" mode")
  }

  // process any autoinvite requests
  if len(GetArgs().AutoInvites) > 0 {
    autoinvites.ProcessInviteFlag(GetArgs().AutoInvites)
    return
  }

  if !GetArgs().Passive {
    go peermanager.StartOutbound()
  }

  fmt.Println("Listening on", GetArgs().Addr)
  if globals.AndroidCallback != nil {
    globals.AndroidCallback("Listening on " + GetArgs().Addr)
  }
  http.HandleFunc("/", serveWs)
  http.HandleFunc("/api/", serveHttpApi)
  http.HandleFunc("/serve/", serveContent)

  // broadcast new messages to all connections
  go peermanager.Broadcaster()

  // setup lan discovery
  if GetArgs().NoLanDiscovery == false {
    go peermanager.StartLanDiscovery()
  }

  httpServeHandler = &http.Server{Addr: GetArgs().Addr}
  if err := httpServeHandler.ListenAndServe(); err != http.ErrServerClosed {
    // unexpected error. port in use?
    log.Fatalf("ListenAndServe(): %v", err)
  }

}

func Stop() {
  fmt.Printf("Shutting down...\n")
  if globals.AndroidCallback != nil {
    globals.AndroidCallback("Shutting down...")
  }

  // will stop peermanager
  globals.StartShutDown = true

  peermanager.StopAllSessions()

  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()
  httpServeHandler.Shutdown(ctx)
}
