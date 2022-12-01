package db

import (
  "fmt"
)

func ClearContent() {
  fmt.Printf("Clear content\n")

  tx := GetTx(true)
  tx.DeleteBucket([]byte("pastes"))
  tx.DeleteBucket([]byte("accounts"))
  tx.DeleteBucket([]byte("deltalog"))
  tx.DeleteBucket([]byte("names"))

  _, err := tx.CreateBucketIfNotExists([]byte("accounts"))
  if err != nil {
    panic(err)
  }

  cb := tx.Bucket([]byte("config"))
  cb.Delete([]byte("latestSequenceNo"))
  cb.Delete([]byte("merklehead"))
  cb.Delete([]byte("sequenceSeed"))

  CommitTx(tx)
}
