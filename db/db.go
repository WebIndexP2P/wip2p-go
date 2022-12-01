package db

import (
  "os"
  "fmt"
  "log"
  "encoding/hex"
  bolt "go.etcd.io/bbolt"
  "github.com/ipfs/go-cid"
)

var db *bolt.DB

func IsInit() bool {
  if db == nil {
    return false
  } else {
    return true
  }
}

func IsFirstRun(dbname string) bool {
  if _, err := os.Stat(dbname); err == nil {
    return false
  } else {
    return true
  }
}

func DbInit(dbname string) (retErr error) {

  // bring up the db
  var err error
  db, err = bolt.Open(dbname, 0600, nil)
  if err != nil {
    log.Fatal(err)
  }

  tx := GetTx(true)
  defer CommitTx(tx)

  b, err := tx.CreateBucketIfNotExists([]byte("accounts"))
  if err != nil {
    return fmt.Errorf("create bucket: %s", err)
  }

  b, err = tx.CreateBucketIfNotExists([]byte("pastes"))
  if err != nil {
    return fmt.Errorf("create bucket: %s", err)
  }

  b, err = tx.CreateBucketIfNotExists([]byte("config"))
  if err != nil {
    return fmt.Errorf("create bucket: %s", err)
  }

  b, err = tx.CreateBucketIfNotExists([]byte("bootstrap"))
  if err != nil {
    return fmt.Errorf("create bucket: %s", err)
  }

  b, err = tx.CreateBucketIfNotExists([]byte("peers"))
  if err != nil {
    return fmt.Errorf("create bucket: %s", err)
  }

  b, err = tx.CreateBucketIfNotExists([]byte("invites"))
  if err != nil {
    return fmt.Errorf("create bucket: %s", err)
  }

  b = b

  if err != nil {
    log.Fatal(err)
  }

  fmt.Println("DB initialized")
  return nil
}

func Dump(tx *bolt.Tx) {
  fmt.Println("Accounts:")
  b := tx.Bucket([]byte("accounts"))
  b.ForEach(func(k, v []byte) error {
    fmt.Printf("key=%s, value=%s\n", "0x" + hex.EncodeToString(k), v)
    return nil
  })
  fmt.Println()

  fmt.Println("Pastes:")
  b = tx.Bucket([]byte("pastes"))
  b.ForEach(func(k, v []byte) error {
    shortv := v
    if len(shortv) > 32 {
      shortv = v[0:32]
    }
    tmpCid := cid.NewCidV1(0x71, k).String()
    fmt.Printf("key=%s, value=%v\n", tmpCid, shortv)
    return nil
  })
  fmt.Println()

  fmt.Println("Config:")
  b = tx.Bucket([]byte("config"))
  b.ForEach(func(k, v []byte) error {
    fmt.Printf("key=%s, value=%v\n", k, v)
    return nil
  })
  fmt.Println()

  fmt.Println("Bootstrap peers:")
  b = tx.Bucket([]byte("bootstrap"))
  b.ForEach(func(k, v []byte) error {
    fmt.Printf("key=%s, value=%v\n", k, v)
    return nil
  })
  fmt.Println()

  fmt.Println("Peers:")
  b = tx.Bucket([]byte("peers"))
  b.ForEach(func(k, v []byte) error {
    fmt.Printf("key=0x%v, value=%s\n", hex.EncodeToString(k), v)
    return nil
  })
  fmt.Println()

  fmt.Println("Invites:")
  b = tx.Bucket([]byte("invites"))
  b.ForEach(func(k, v []byte) error {
    fmt.Printf("key=%v, value=%v\n", hex.EncodeToString(k), v)
    return nil
  })
  fmt.Println()

}

func GetTx(writable bool) (tx *bolt.Tx) {
  //fmt.Println("db.GetTx")
  tx, err := db.Begin(writable)
  if err != nil {
    fmt.Println(err)
    return nil
  }
  //defer tx.Rollback()
  return tx
}

func CommitTx(tx *bolt.Tx) {
  //fmt.Println("db.CommitTx")
  err := tx.Commit()
  if err != nil {
    fmt.Println(err)
  }
}

func RollbackTx(tx *bolt.Tx) {
  //fmt.Println("db.RollbackTx")
  tx.Rollback()
}
