package deltalog

import (
  "fmt"
  "log"
  "encoding/hex"
  //"encoding/binary"
  bolt "go.etcd.io/bbolt"

  "github.com/ethereum/go-ethereum/common"

  "code.wip2p.com/mwadmin/wip2p-go/db"
  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
  "code.wip2p.com/mwadmin/wip2p-go/util"
)

// 0x01[seqNo,account]
// 0x02[account,seqNo]

func Init(tx *bolt.Tx) {

  if tx == nil {
    tx = db.GetTx(true)
    defer db.CommitTx(tx)
  }

  _, err := tx.CreateBucketIfNotExists([]byte("deltalog"))
  if err != nil {
    fmt.Errorf("create bucket: %s", err)
  }

}

func Update(account common.Address, isRemoval bool, tx *bolt.Tx) uint64 {
  // check this account not already in deltalog update and remove sequences
  // save the paste seq for the account
  b := tx.Bucket([]byte("deltalog"))
  seqNo, _ := b.NextSequence()
  seqNoBytes := util.Itob(seqNo)
  //binary.PutUvarint(seqNoBytes, seqNo)

  conf := db.Config{}
  globals.LatestSequenceNo = uint(seqNo)
  conf.WriteUint("latestSequenceNo", globals.LatestSequenceNo, tx)

  indexKey := make([]byte, 21)
  indexKey[0] = 0x02
  copy(indexKey[1:], account[:])

  existingSeqNoBytes := b.Get(indexKey)

  if existingSeqNoBytes != nil {
    //existingSeqNo, _ := binary.Uvarint(existingSeqNoBytes)
    oldLogKey := make([]byte, 9)
    oldLogKey[0] = 0x01
    copy(oldLogKey[1:], existingSeqNoBytes)
    b.Delete(oldLogKey)
  }

  newLogKey := make([]byte, 9)
  newLogKey[0] = 0x01
  copy(newLogKey[1:], seqNoBytes)
  //[account|isRemoval]
  dataBuf := make([]byte, 21)
  copy(dataBuf, account.Bytes())
  if isRemoval {
    copy(dataBuf[20:], []byte{0x01})
  } else {
    copy(dataBuf[20:], []byte{0x00})
  }
  b.Put(newLogKey, dataBuf)
  //fmt.Printf("Added log for %+v\n", account)

  newIndexKey := make([]byte, 21)
  newIndexKey[0] = 0x02
  copy(newIndexKey[1:], account.Bytes())
  b.Put(newIndexKey, seqNoBytes)
  //fmt.Printf("Added index for %+v\n", account)

  return seqNo
}


func GetDeltaLogSequence() uint64 {
  tx := db.GetTx(true)
  b := tx.Bucket([]byte("deltalog"))
  nextSeq, err := b.NextSequence()
  if err != nil {
    log.Fatal(err)
  }
  db.RollbackTx(tx)
  return nextSeq - 1
}

func Dump() {
  tx := db.GetTx(true)
  defer db.RollbackTx(tx)

  fmt.Println("deltalog:")
  b := tx.Bucket([]byte("deltalog"))
  b.ForEach(func(k, v []byte) error {
    if k[0] == 0x01 {
      seqNo := util.Btoi(k[1:])
      fmt.Printf("key=%v, value=0x%s (%v)\n", seqNo, hex.EncodeToString(v[:20]), v[20])
      return nil
    } else {
      acct := k[1:]
      seqNo := util.Btoi(v)
      fmt.Printf("key=0x%s, value=%v\n", hex.EncodeToString(acct), seqNo)
      return nil
    }
  })
  fmt.Println()
}

type deltaRow struct {
  SeqNo uint64
  Address common.Address
  Removed bool
}

func GetBatch(startSequence uint64, includeRemovals bool) []deltaRow {
  tx := db.GetTx(false)
  defer db.RollbackTx(tx)

  b := tx.Bucket([]byte("deltalog"))

  c := b.Cursor()

  results := make([]deltaRow, 0)

  startPrefix := make([]byte, 9)
  startPrefix[0] = 0x01
  copy(startPrefix[1:], util.Itob(startSequence))

  for k, v := c.Seek(startPrefix); k != nil; k, v = c.Next() {

    if k[0] == 0x02 {
      break
    }

    seqNo := util.Btoi(k[1:])
    isRemovalB := v[20]
    var isRemoval bool
    if isRemovalB == 0x01 {
      isRemoval = true
    } else {
      isRemoval = false
    }
    if includeRemovals == false && isRemoval {
      continue
    }
    reqAddress := common.BytesToAddress(v[:20])
    results = append(results, deltaRow{Address: reqAddress, SeqNo: seqNo, Removed: isRemoval})

    if len(results) >= 10 {
      break
    }
  }

  return results
}

func GetRecent(count uint, includeRemovals bool) []common.Address {
  tx := db.GetTx(false)
  defer db.RollbackTx(tx)

  b := tx.Bucket([]byte("deltalog"))

  c := b.Cursor()

  results := make([]common.Address, 0)

  c.Seek([]byte{0x02})

  for k, v := c.Prev(); k != nil; k, v = c.Prev() {
    isRemoval := v[20]
    if includeRemovals == false && isRemoval == 0x01 {
      continue
    }
    reqAddress := common.BytesToAddress(v[:20])
    results = append(results, reqAddress)
  }

  return results
}
