package public

import (
  "log"
  bolt "go.etcd.io/bbolt"
  "github.com/ipfs/go-ipld-cbor"
  mh "github.com/multiformats/go-multihash"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)

func GetModeFromDb() bool {
  conf := db.Config{}
  modeB := conf.Read("publicMode", nil)
  if len(modeB) == 0 {
    return false
  }
  if modeB[0] == byte(0) {
    return false
  } else {
    return true
  }
}

func SetMode(newMode bool, tx *bolt.Tx) {
  globals.PublicMode = newMode
  conf := db.Config{}
  if newMode {
    conf.Write("publicMode", []byte{1}, tx)
  } else {
    conf.Write("publicMode", []byte{0}, tx)
  }
}

func UpdatePublicMode(rootDocBytes []byte, tx *bolt.Tx) {
  origPublicMode := globals.PublicMode

  rootIpldNode, _ := cbornode.Decode(rootDocBytes, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
  newPublicMode := CheckIpldPublicMode(rootIpldNode)

  if origPublicMode != newPublicMode {
    if newPublicMode {
      log.Println("Enabled public mode")
    } else {
      log.Println("Disabled public mode")
    }
    SetMode(newPublicMode, tx)
  }
}

func CheckIpldPublicMode(ipldNode *cbornode.Node) bool {
	var newPublicMode bool
	valueIface, _, _ := ipldNode.Resolve([]string{"wip2p", "public"})
	if valueIface == nil {
    newPublicMode = false
  } else {
		pubMode := valueIface.(bool)
		newPublicMode = pubMode
  }
  return newPublicMode
}
