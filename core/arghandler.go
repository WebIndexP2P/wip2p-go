package core

import (
  "os"
  "fmt"
  "flag"
  "errors"
  "strings"
  "strconv"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/names"
  "code.wip2p.com/mwadmin/wip2p-go/deltalog"
  "code.wip2p.com/mwadmin/wip2p-go/peermanager"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)

type bootArrayFlags []string

func (i *bootArrayFlags) String() string {
	return "my string representation"
}

func (i *bootArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type AppArgs struct {
  BootArray bootArrayFlags
  Addr string
  Dump bool
  Version bool
  New bool
  RootAccount string
  Passive bool
  DebugFlag bool
  EndPoints string
  AutoInvites string
  Export bool
  ClearPeers bool
  Db string
  ClearContent bool
  NoRoot bool
  NoLanDiscovery bool
  PeerBoot bool
  EnableApi bool
}

var appArgs = AppArgs{}

func ParseArgs(args string) {

  flagset := flag.NewFlagSet("", 0)

  flagset.StringVar(&appArgs.Addr, "addr", "0.0.0.0:9472", "http service address")
  flagset.BoolVar(&appArgs.Dump, "dump", false, "dump db to output")
  flagset.BoolVar(&appArgs.Version, "version", false, "prints app version")
  flagset.BoolVar(&appArgs.New, "new", false, "start a new tree")
  flagset.StringVar(&appArgs.RootAccount, "root", "", "set the root account")
  flagset.BoolVar(&appArgs.Passive, "passive", false, "disable outbound peer connections")
  flagset.BoolVar(&appArgs.DebugFlag, "debug", false, "enable debug output")
  flagset.StringVar(&appArgs.EndPoints, "endpoints", "", "comma delimited list of accessible endpoints for this node")
  flagset.Var(&appArgs.BootArray, "boot", "bootstrap endpoint i.e. ws://yourwip2p.node:9472, can be used more than once")
  flagset.StringVar(&appArgs.AutoInvites, "autoinvite", "", "enable some autoinvites, example (nokey|key,num1,days30,lvl9,max50) or clear")
  flagset.BoolVar(&appArgs.Export, "export", false, "export node private key seed")
  flagset.BoolVar(&appArgs.ClearPeers, "clearpeers", false, "erase all saved peers")
  flagset.StringVar(&appArgs.Db, "db", "wip2p.db", "database filename")
  flagset.BoolVar(&appArgs.ClearContent, "clearcontent", false, "clear out bundles, sequence, sequence seed, merklehead")
  flagset.BoolVar(&appArgs.NoRoot, "noroot", false, "don't set the node account as the tree root, used with -new")
  flagset.BoolVar(&appArgs.NoLanDiscovery, "nolandiscovery", false, "dont bind udp 31458 and send broadcasts every 30 seconds")
  flagset.BoolVar(&appArgs.PeerBoot, "peerboot", false, "set tree root to match first peer that connects, used with -new")
  flagset.BoolVar(&appArgs.EnableApi, "enableapi", false, "allows open access to api even in private mode")

  if args == "" {
    flagset.Parse(os.Args[1:])
  } else {
    argArray := strings.Split(args, " ")
    flagset.Parse(argArray)
  }

}

func GetArgs() AppArgs {
  return appArgs
}


func ProcessArgs(phase int) (bool, error) {
  // process lanboot arg
  bShouldExit, err := processPeerBoot(phase)
  if bShouldExit {
    return bShouldExit, err
  }

  bShouldExit, err = processDump(phase)
  if bShouldExit {
    return bShouldExit, err
  }

  bShouldExit, err = clearContent(phase)
  if bShouldExit {
    return bShouldExit, err
  }

  bShouldExit, err = export(phase)
  if bShouldExit {
    return bShouldExit, err
  }

  bShouldExit, err = clearPeers(phase)
  if bShouldExit {
    return bShouldExit, err
  }

  bShouldExit, err = setListenPort(phase)
  if bShouldExit {
    return bShouldExit, err
  }

  bShouldExit, err = debugLogging(phase)
  if bShouldExit {
    return bShouldExit, err
  }

  bShouldExit, err = endpoints(phase)
  if bShouldExit {
    return bShouldExit, err
  }

  bShouldExit, err = enableApi(phase)
  if bShouldExit {
    return bShouldExit, err
  }

  if appArgs.Version {
    return true, nil
  }

  return false, nil
}

func processDump(phase int) (bool, error) {
  if !appArgs.Dump {
    return false, nil
  }

  if phase == 1 {
    if globals.FirstRun {
      return true, errors.New("no database found")
    }
    return false, nil
  }

  tx := db.GetTx(false)
  defer db.RollbackTx(tx)
  db.Dump(tx)
  deltalog.Dump()
  names.Dump()
  return true, nil
}

func processPeerBoot(phase int) (bool, error) {

  if appArgs.PeerBoot && phase == 1 {
    if !globals.FirstRun {
      return true, errors.New("cannot peerboot on an existing database")
    }

    if len(appArgs.RootAccount) > 0 {
      return true, errors.New("cannot use rootaccount in combination with peerboot")
    }

    globals.GetRootFromNextPeer = true
    appArgs.New = true
  } else if appArgs.PeerBoot && phase == 2 {

    conf := db.Config{}
    conf.Write("getRootFromNextPeer", []byte{1}, nil)

  } else if !appArgs.PeerBoot && phase == 2 {
    conf := db.Config{}
    getRootFromNextPeer := conf.Read("getRootFromNextPeer", nil)
    if len(getRootFromNextPeer) > 0 && getRootFromNextPeer[0] == 0x01 {
      globals.GetRootFromNextPeer = true
    }
  }

  return false, nil
}

func clearContent(phase int) (bool, error) {
  if !appArgs.ClearContent {
    return false, nil
  }

  if phase == 2 {
    db.ClearContent()
    conf := db.Config{}
    rootAccount := conf.Read("rootAccount", nil)
    SaveRootAccount(rootAccount)
    return true, nil
  }

  return false, nil
}

func export(phase int) (bool, error) {
  if appArgs.Export && phase == 2 {
    seed := ExportNodeKeySeed()
    fmt.Printf("%+v\n", seed)
    return true, nil
  }
  return false, nil
}

func clearPeers(phase int) (bool, error) {
  if appArgs.ClearPeers && phase == 2{
    peermanager.ClearAll()
    fmt.Printf("Erased all peers\n")
    return true, nil
  }
  return false, nil
}

func setListenPort(phase int) (bool, error) {
  // get the listen port
  if phase == 1 {
    return false, nil
  }

  addrArray := strings.Split(appArgs.Addr, ":")
  if len(addrArray) == 2 {
    tmpPort, _ := strconv.ParseUint(addrArray[1], 10, 64)
    globals.ListenPort = uint(tmpPort)
  } else {
    globals.ListenPort = 9472
  }
  return false, nil
}

func debugLogging(phase int) (bool, error) {
  if appArgs.DebugFlag && phase == 2 {
    globals.DebugLogging = true
  }

  return false, nil
}

func enableApi(phase int) (bool, error) {
  if (appArgs.EnableApi && phase == 2) {
    globals.EnableApi = true
  }
  return false, nil
}

func endpoints(phase int) (bool, error) {
  if phase == 1 {
    return false, nil
  }

  if appArgs.EndPoints == "clear" {
    globals.Endpoints = ""
    conf := db.Config{}
    conf.Delete("endpoints", nil)
  } else if len(appArgs.EndPoints) > 0 {
    globals.Endpoints = appArgs.EndPoints
    conf := db.Config{}
    conf.Write("endpoints", []byte(appArgs.EndPoints), nil)
  } else {
    conf := db.Config{}
    dbEndpoints := string(conf.Read("endpoints", nil))
    if len(dbEndpoints) == 0 {
      globals.Endpoints = ":" + strconv.FormatUint(uint64(globals.ListenPort), 10)
    } else {
      globals.Endpoints = dbEndpoints
    }
  }

  return false, nil
}
