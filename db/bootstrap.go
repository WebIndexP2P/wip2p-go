package db

import (
  "log"
  "encoding/binary"
  bolt "go.etcd.io/bbolt"
)

type BootstrapPeer struct {}

func (b *BootstrapPeer) Add(endpoint string, tx *bolt.Tx) {

  if tx == nil {
    tx = GetTx(true)
    defer CommitTx(tx)
  }

  bucket := tx.Bucket([]byte("bootstrap"))

  countB := make([]byte, 4)
  binary.LittleEndian.PutUint32(countB, 0)
  err := bucket.Put([]byte(endpoint), countB)

  if err != nil {
    log.Fatal("problem saving bootstrap peer")
  }
}

func (b *BootstrapPeer) Delete(endpoint string, tx *bolt.Tx) {
  if tx == nil {
    tx = GetTx(true)
    defer CommitTx(tx)
  }

  bucket := tx.Bucket([]byte("bootstrap"))
  err := bucket.Delete([]byte(endpoint))

  if err != nil {
    log.Fatal("problem deleting bootstrap peer")
  }

}
