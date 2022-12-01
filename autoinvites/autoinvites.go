package autoinvites

import (
  "log"
  "fmt"
  "time"
  "bytes"
  "errors"
  "crypto/ecdsa"
  "encoding/binary"
  "encoding/hex"
  "strconv"
  "strings"
  bolt "go.etcd.io/bbolt"
  "github.com/ipfs/go-ipld-cbor"
  mh "github.com/multiformats/go-multihash"
  "github.com/ethereum/go-ethereum/crypto"
  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/account"
  "code.wip2p.com/mwadmin/wip2p-go/signature"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/sigbundle"
  "code.wip2p.com/mwadmin/wip2p-go/sigbundle/sigbundlestruct"
)

type Invite struct {
  Lvl uint16
  Days uint16
  Num uint16
  Max uint16
}

func ProcessInviteFlag(flag string) {

  // save invite params to db
  tx := db.GetTx(true)
  defer db.CommitTx(tx)
  bucket := tx.Bucket([]byte("invites"))

  // check for clear flag
  if flag == "clear" {
    bucket.ForEach(func(k, v []byte) error {
      bucket.Delete(k)
      return nil
    })
    fmt.Println("\nAll invites cleared")
    return
  }

  fmt.Println("\nCreating autoinvite...")

  address := crypto.PubkeyToAddress(globals.NodePrivateKey.PublicKey)
  _, success := account.FetchAccountFromDb(address, nil, false)

  if !success {
    fmt.Println("Warning! NodeAccount does not appear to be an active account in this tree.")
  }

  key := []byte{0}
  newInvite := Invite{
    Lvl: 0,
    Days: 30,
    Num: 1,
    Max: 0,
  }

  params := strings.Split(flag, ",")
  for _, param := range params {
    if param == "key" {
      key = make([]byte, 16)
      privKey, _ := crypto.GenerateKey()
      copy(key, crypto.FromECDSA(privKey))
    } else if param == "nokey" {

    } else if strings.HasPrefix(param, "num") {
      tmpNum, _ := strconv.Atoi(param[3:])
      newInvite.Num = uint16(tmpNum)
    } else if strings.HasPrefix(param, "lvl") {
      tmpLvl, _ := strconv.Atoi(param[3:])
      newInvite.Lvl = uint16(tmpLvl)
    } else if strings.HasPrefix(param, "days") {
      tmpDays, _ := strconv.Atoi(param[4:])
      newInvite.Days = uint16(tmpDays)
    } else if strings.HasPrefix(param, "max") {
      tmpMax, _ := strconv.Atoi(param[3:])
      newInvite.Max = uint16(tmpMax)
    } else {
      log.Fatal("Unknown autoinvite tag")
    }
  }

  if newInvite.Num < 0 || newInvite.Num >= 10000 {
    log.Fatal("num invites must be between 0 and 9999")
  }

  if newInvite.Max < 0 || newInvite.Max >= 10000 {
    log.Fatal("max invites must be between 0 and 9999")
  }

  if newInvite.Lvl < 0 || newInvite.Lvl > 10 {
    log.Fatal("lvl must be between 0 and 10")
  }

  var qtyString string
  if newInvite.Num == 0 {
    qtyString = "unlimited"
  } else {
    qtyString = strconv.Itoa(int(newInvite.Num))
  }

  if len(key) == 1 {
    fmt.Printf("Added %v invites with no key\n", qtyString)
  } else {
    fmt.Printf("Added %v invites with key %s\n", qtyString, hex.EncodeToString(key))
  }

  buf := new(bytes.Buffer)
  binary.Write(buf, binary.LittleEndian, uint16(newInvite.Num))
  binary.Write(buf, binary.LittleEndian, uint16(newInvite.Lvl))
  binary.Write(buf, binary.LittleEndian, uint16(newInvite.Days))
  binary.Write(buf, binary.LittleEndian, uint16(newInvite.Max))

  err := bucket.Put(key, buf.Bytes())
  if err != nil {
    log.Fatal("error saving invite")
  }

}

func GetStatus() (bool, bool) {
  nokey := false
  key := false

  // save invite params to db
  tx := db.GetTx(false)
  defer db.RollbackTx(tx)
  bucket := tx.Bucket([]byte("invites"))

  bucket.ForEach(func(k, v []byte) error {
    if len(k) == 1 {
      nokey = true
    } else {
      key = true
    }
    return nil
  })

  return nokey, key
}

func RedeemInviteKey(key string, tx *bolt.Tx) (inviteDetails *Invite, err error) {
  bucket := tx.Bucket([]byte("invites"))
  var keyB []byte

  if key == "" {
    keyB = []byte{0}
  } else {
    var err error
    keyB, err = hex.DecodeString(key)
    if err != nil {
      return nil, err
    }
  }

  inviteB := bucket.Get(keyB)
  if inviteB == nil {
    return nil, errors.New("invite not found")
  }

  tInvite := Invite{}
  buf := bytes.NewReader(inviteB)
  binary.Read(buf, binary.LittleEndian, &tInvite.Num)
  binary.Read(buf, binary.LittleEndian, &tInvite.Lvl)
  binary.Read(buf, binary.LittleEndian, &tInvite.Days)
  binary.Read(buf, binary.LittleEndian, &tInvite.Max)

  // decrement invite counter
  if tInvite.Num == 0 {
    // unlimited, dont do anything
  } else if tInvite.Num == 1 {
    bucket.Delete(keyB)
  } else {
    tInvite.Num--

    // save the update
    buf := new(bytes.Buffer)
    binary.Write(buf, binary.LittleEndian, uint16(tInvite.Num))
    binary.Write(buf, binary.LittleEndian, uint16(tInvite.Lvl))
    binary.Write(buf, binary.LittleEndian, uint16(tInvite.Days))

    err := bucket.Put(keyB, buf.Bytes())
    if err != nil {
      log.Fatal("error updating invite")
    }
  }

  return &tInvite, nil
}

func CreateInvite(requestingAccount common.Address, inviteDetails Invite, tx *bolt.Tx) (*sigbundlestruct.SigBundle, error) {

  //bucket := tx.Bucket([]byte("invites"))

  nodeAddress := crypto.PubkeyToAddress(globals.NodePrivateKey.PublicKey)

  var rootDoc interface{}
  unixNow := uint(time.Now().UTC().Unix())
  acct, success := account.FetchAccountFromDb(nodeAddress, tx, false)

  if !success {
    return nil, errors.New("inviter account not found")
  }

  if success && acct.RootMultihash != nil {
    // fetch document
    doc := db.Doc{}
    docWithRefs, _ := doc.Get(acct.RootMultihash, tx)

    var err error
    var nd *cbornode.Node

    nd, err = cbornode.Decode(docWithRefs.Data, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
    if err != nil {
      log.Fatal(err)
    }

    rootDoc, _, _ = nd.Resolve(nil)

    wip2p := rootDoc.(map[string]interface{})["wip2p"].(map[string]interface{})
    invites := wip2p["i"].([]interface{})

    oldestExpiredIdx := -1

    // make sure account is not already invited
    for inviteIdx, tmpInviteIFace := range invites {
      tmpInvite := tmpInviteIFace.(map[string]interface{})
      invitedAccountB := tmpInvite["account"].([]byte)

      if bytes.Equal(invitedAccountB, requestingAccount.Bytes()) {
        return nil, errors.New("account already invited")
      }

      // check for expired invites
      timestamp := uint(tmpInvite["timestamp"].(int))
      expireDaysIface, success := tmpInvite["expire"]
      var expireDays uint
      if success {
        expireDays = uint(expireDaysIface.(int))
      }
      if expireDays > 0 {
        expiry := timestamp + (expireDays * 24 * 60 * 60)
        if expiry < unixNow {
          if oldestExpiredIdx == -1 {
            oldestExpiredIdx = inviteIdx
          } else {
            oldestInvite := invites[oldestExpiredIdx].(map[string]interface{})
            oldTimestamp := uint(oldestInvite["timestamp"].(int))
            oldExpireDays := uint(oldestInvite["expire"].(int))
            oldExpiry := oldTimestamp + (oldExpireDays * 24 * 60 * 60)
            if expiry < oldExpiry {
              oldestExpiredIdx = inviteIdx
            }
          }
        }
      }
    }

    if oldestExpiredIdx >= 0 {
      //fmt.Printf("removing expired invite %+v\n", oldestExpiredIdx)
      invites = append(invites[:oldestExpiredIdx], invites[oldestExpiredIdx + 1:]...)
    }

    newInvite := map[string]interface{}{"account": requestingAccount, "timestamp": unixNow}
    if inviteDetails.Lvl > 0 {
      newInvite["lvlgap"] = inviteDetails.Lvl
    }
    if inviteDetails.Days > 0 {
      newInvite["expire"] = inviteDetails.Days
    }
    invites = append(invites, newInvite)
    wip2p["i"] = invites

  } else {

    tmpInvite := map[string]interface{}{"account": requestingAccount, "timestamp": unixNow}
    if inviteDetails.Lvl > 0 {
      tmpInvite["lvlgap"] = inviteDetails.Lvl
    }
    if inviteDetails.Days > 0 {
      tmpInvite["expire"] = inviteDetails.Days
    }

    invites := []interface{}{tmpInvite}
    inviteDoc := map[string]interface{}{"i": invites}
    rootDoc = map[string]interface{}{"wip2p": inviteDoc}

  }

  // save the new doc
  sigBundle, err := createAndSaveDoc(unixNow, rootDoc, globals.NodePrivateKey, tx)
  if err != nil {
    return nil, err
  }

  return sigBundle, nil
}

func createAndSaveDoc(timestamp uint, objData interface{}, privateKey *ecdsa.PrivateKey, tx *bolt.Tx) (sigBundle *sigbundlestruct.SigBundle, err error) {
  nd, err := cbornode.WrapObject(objData, mh.SHA2_256, mh.DefaultLengths[mh.SHA2_256])
  if err != nil {
    return nil, err
  }

  rootMultihash := "0x" + hex.EncodeToString(nd.Cid().Hash())

  signedString := fmt.Sprintf("[%v,\"%v\"]", timestamp, rootMultihash)
  hash, _ := signature.EthHashString(signedString)
  sigB, _ := signature.Sign(hash, privateKey)

  // construct paste_save payload
  author := crypto.PubkeyToAddress(privateKey.PublicKey).Bytes()
  bundle := sigbundlestruct.SigBundle{
    CborData: [][]byte{ nd.RawData() },
    Signature: sigB,
    Timestamp: uint64(timestamp),
    RootMultihash: nd.Cid().Hash(),
    Account: author,
  }

  _, _, err = sigbundle.ValidateAndSave(bundle, tx)
  if err != nil {
    return nil, err
  }

  return &bundle, nil
}
