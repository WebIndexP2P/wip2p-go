package db

import (
  //"fmt"
  "bytes"
  "encoding/binary"

  bolt "go.etcd.io/bbolt"
)

type Config struct {}

func (c *Config) Write(key string, value []byte, tx *bolt.Tx) {
  if tx == nil {
    tx = GetTx(true)
    defer CommitTx(tx)
  }

  cb := tx.Bucket([]byte("config"))
  cb.Put([]byte(key), value)
}

func (c *Config) Read(key string, tx *bolt.Tx) []byte {

  if tx == nil {
    tx = GetTx(false)
    defer RollbackTx(tx)
  }

  cb := tx.Bucket([]byte("config"))
  return cb.Get([]byte(key))
}

func (c *Config) WriteUint(key string, value uint, tx *bolt.Tx) {
  if tx == nil {
    tx = GetTx(true)
    defer CommitTx(tx)
  }

  cb := tx.Bucket([]byte("config"))

  buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint64(value))

  cb.Put([]byte(key), buf.Bytes())
}

func (c *Config) ReadUint(key string, tx *bolt.Tx) uint {

  if tx == nil {
    tx = GetTx(false)
    defer RollbackTx(tx)
  }

  cb := tx.Bucket([]byte("config"))
  valueB := cb.Get([]byte(key))

  buf := bytes.NewReader(valueB)
	var tmpVal uint64
	binary.Read(buf, binary.BigEndian, &tmpVal)

  return uint(tmpVal)
}


func (c *Config) Delete(key string, tx *bolt.Tx) {

  if tx == nil {
    tx = GetTx(true)
    defer CommitTx(tx)
  }

  cb := tx.Bucket([]byte("config"))
  cb.Delete([]byte(key))
}
