package core

import (
  "fmt"
  "log"
  //"errors"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/crypto"

  "code.wip2p.com/mwadmin/wip2p-go/defaults"
  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/public"
  "code.wip2p.com/mwadmin/wip2p-go/peermanager"
  "code.wip2p.com/mwadmin/wip2p-go/invite"
  "code.wip2p.com/mwadmin/wip2p-go/account"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)

func InitRootAccount() ([]byte) {

  rootAccountFlag := GetArgs().RootAccount
  newTree := GetArgs().New
  noRoot := GetArgs().NoRoot

  if !globals.FirstRun && (rootAccountFlag != "" || newTree) {
    log.Fatal("Tree cannot be modified, remove the database file")
  }

  if noRoot && !newTree {
    log.Fatal("-noroot requires -new")
  }

  var rootAccount []byte

  tx := db.GetTx(false)
  cb := tx.Bucket([]byte("config"))
  configRootAccountB := cb.Get([]byte("rootAccount"))
  db.RollbackTx(tx)

  if rootAccountFlag != "" {

    if configRootAccountB != nil {
      log.Fatal("cannot change root on already existing tree")
    }

    // verify account
    success := common.IsHexAddress(rootAccountFlag)
    if !success {
      log.Fatal("invalid root account")
    }

    // set the root Account
    rootAccountAddr := common.HexToAddress(rootAccountFlag)
    rootAccount = rootAccountAddr.Bytes()
    SaveRootAccount(rootAccount)
    fmt.Println("RootAccount:", rootAccountFlag)

  } else {

    //fmt.Println("no root account passed, check for existing")

    if configRootAccountB == nil {
      if newTree {

        public.SetMode(false, nil)

        if GetArgs().PeerBoot {
          fmt.Println("PeerBoot set, RootAccount will be obtained from first peer")
          globals.GetRootFromNextPeer = true;
          conf := db.Config{}
          conf.Write("getRootFromNextPeer", []byte{1}, nil)
          return nil
        } else if !noRoot {
          address := crypto.PubkeyToAddress(globals.NodePrivateKey.PublicKey)
          fmt.Println("RootAccount using this new node address", address.String())
          SaveRootAccount(address.Bytes())

          globals.NextAuthGetsInvite = true;
          conf := db.Config{}
          conf.Write("nextAuthGetsInvite", []byte{1}, nil)

          return address.Bytes()
        } else {
          fmt.Println("No RootAccount set, first account will be assigned as Root")
          globals.NextAuthIsRoot = true;
          conf := db.Config{}
          conf.Write("nextAuthIsRoot", []byte{1}, nil)
          return nil
        }

      } else if globals.FirstRun {

        generalSherman := defaults.GetTrees()[0]
        rootAccountAddr := common.HexToAddress(generalSherman.RootAccount)
        rootAccount = rootAccountAddr.Bytes()
        SaveRootAccount(rootAccount)
        fmt.Printf("RootAccount using default tree %s - \"%s\"\n", generalSherman.RootAccount, generalSherman.Name)
        for _, ep := range generalSherman.BootstrapPeers {
          peermanager.AddEndpointStringToQueue(ep)
        }
        public.SetMode(true, nil)

      } else {

        if globals.GetRootFromNextPeer {
          fmt.Println("PeerBoot set, RootAccount will be obtained from first peer")
        } else {
          // get status from db, either
          fmt.Println("No RootAccount set, first account will be assigned as Root")
        }

        public.SetMode(false, nil)
        return nil

      }
    } else {
      rootAccountAddr := common.BytesToAddress(configRootAccountB)
      rootAccount = configRootAccountB
      fmt.Println("RootAccount:", rootAccountAddr.String())
    }
  }

  return rootAccount
}

func SaveRootAccount(rootAccount []byte) {
  if len(rootAccount) != 20 {
    log.Fatal("rootAccount not 20 bytes")
  }

  tx := db.GetTx(true)
  cb := tx.Bucket([]byte("config"))
  cb.Put([]byte("rootAccount"), rootAccount)

  inviters := make([]invite.Inviter, 0)
  inviters = append(inviters, invite.Inviter{Timestamp: uint64(1)})
  rootAccountObj := account.AccountStruct{Address: common.BytesToAddress(rootAccount), Inviters: inviters, ActiveInviter: inviters[0], Enabled: true}
  rootAccountObj.SaveToDb(tx)

  db.CommitTx(tx)
}
