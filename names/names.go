package names

import (
  "fmt"
  //"log"
  "bytes"
  "errors"
  "encoding/json"
  bolt "go.etcd.io/bbolt"
  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/db"
)

func Init(tx *bolt.Tx) {
  if tx == nil {
    tx = db.GetTx(true)
    defer db.CommitTx(tx)
  }

  _, err := tx.CreateBucketIfNotExists([]byte("names"))
  if err != nil {
    fmt.Errorf("create bucket: %s", err)
  }

}

func Dump() {
  tx := db.GetTx(true)
  defer db.RollbackTx(tx)

  fmt.Println("names:")
  b := tx.Bucket([]byte("names"))
  b.ForEach(func(k, v []byte) error {
    var s []NameIndexStruct
    json.Unmarshal(v, &s)
    fmt.Printf("key=%s, value=%+v\n", k, s)
    return nil
  })
  fmt.Println()
}

type NameIndexStruct struct {
  OwnerAddress common.Address `json:"oa"`
  OwnerInviteTimestamp uint64 `json:"ot"`
  TargetAddress common.Address `json:"ta"`
  Conflict bool `json:"c"`
}

func RemoveAllForAddress(ownerAddress []byte, tx *bolt.Tx) {
  b := tx.Bucket([]byte("names"))
  c := b.Cursor()

  for name, v := c.First(); name != nil; name, v = c.Next() {
    var s []NameIndexStruct
    json.Unmarshal(v, &s)

    newNamesStruct := []NameIndexStruct{}
    dupTimestamps := 0
    prevTimestamp := uint64(0)
    stopDupCount := false
    for idx := range s {

      if !stopDupCount {
        if prevTimestamp == s[idx].OwnerInviteTimestamp {
          dupTimestamps++
        } else {
          dupTimestamps = 1
        }
        prevTimestamp = s[idx].OwnerInviteTimestamp
      }

      if bytes.Equal(s[idx].OwnerAddress[:], ownerAddress) {
        // found one to be removed
        stopDupCount = true
      } else {
        newNamesStruct = append(newNamesStruct, s[idx])
      }
    }
    if (len(newNamesStruct) == 0) {
      b.Delete(name)
      continue
    }

    // update any conflicts
    if dupTimestamps == 1 {
      for idx := range newNamesStruct {
        if newNamesStruct[idx].OwnerInviteTimestamp == prevTimestamp {
          newNamesStruct[idx].Conflict = false
          break
        }
      }
    }

    nameStructBuf, _ := json.Marshal(newNamesStruct)
    b.Put(name, nameStructBuf)
  }
}

func Update(nameOwner []byte, ownerInviteTimestamp uint64, newNames map[string]common.Address, tx *bolt.Tx) (int, int) {

  // Maybe loop through existing names this account has registered and remove them
  RemoveAllForAddress(nameOwner, tx)

  b := tx.Bucket([]byte("names"))

  count := 0
  for name, address := range newNames {
    // check if name already exists
    v := b.Get([]byte(name))

    var newNamesStruct []NameIndexStruct
    ni := NameIndexStruct{
      OwnerAddress: common.BytesToAddress(nameOwner),
      OwnerInviteTimestamp: ownerInviteTimestamp,
      TargetAddress: address,
    }
    if v == nil {
      // create a newbie
      newNamesStruct = []NameIndexStruct{ ni }
      nameStructBuf, _ := json.Marshal(newNamesStruct)
      b.Put([]byte(name), nameStructBuf)
    } else {
      // add to array
      var s []NameIndexStruct
      json.Unmarshal(v, &s)

      // we need to check at which point in the array to inject the new record
      // we also need to remove the record if it already exists elsewhere
      newNamesStruct := []NameIndexStruct{}
      bAdded := false
      for idx := range s {
        if s[idx].OwnerAddress == ni.OwnerAddress {
          continue
        }
        if (s[idx].OwnerInviteTimestamp == ni.OwnerInviteTimestamp) {
          ni.Conflict = true
          s[idx].Conflict = true
          newNamesStruct = append(newNamesStruct, ni)
          bAdded = true
        } else if ni.OwnerInviteTimestamp < s[idx].OwnerInviteTimestamp {
          newNamesStruct = append(newNamesStruct, ni)
          bAdded = true
        }
        newNamesStruct = append(newNamesStruct, s[idx])
      }
      if !bAdded {
        newNamesStruct = append(newNamesStruct, ni)
      }

      nameStructBuf, _ := json.Marshal(newNamesStruct)
      b.Put([]byte(name), nameStructBuf)
    }

    count++
  }

  return count, 0
}

func Lookup(name string) (common.Address, error) {
  tx := db.GetTx(false)
  defer db.RollbackTx(tx)

  b := tx.Bucket([]byte("names"))
  v := b.Get([]byte(name))

  if v == nil {
    return common.Address{}, errors.New("not found")
  }
  var s []NameIndexStruct
  json.Unmarshal(v, &s)

  if s[0].Conflict {
    return common.Address{}, errors.New("name conflict")
  } else {
    return s[0].TargetAddress, nil
  }
}
