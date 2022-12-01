package core

import (
  "fmt"
  "crypto/ecdsa"
  "github.com/ethereum/go-ethereum/crypto"
  "github.com/tyler-smith/go-bip39"

  "code.wip2p.com/mwadmin/wip2p-go/db"
)

func InitNodeKey() *ecdsa.PrivateKey {
  var nodePrivateKey *ecdsa.PrivateKey

  tx := db.GetTx(true)
  cb := tx.Bucket([]byte("config"))
  key := cb.Get([]byte("nodeKey"))
  if key == nil {
    nodePrivateKey, _ = crypto.GenerateKey()
    address := crypto.PubkeyToAddress(nodePrivateKey.PublicKey)
    fmt.Printf("Generated private key for this node with address: %+v\n", address.String());
    cb.Put([]byte("nodeKey"), crypto.FromECDSA(nodePrivateKey))
    db.CommitTx(tx);
  } else {
    nodePrivateKey, _ = crypto.ToECDSA(key);
    address := crypto.PubkeyToAddress(nodePrivateKey.PublicKey)
    fmt.Printf("NodeAccount: %+v\n", address.String());
    db.RollbackTx(tx);
  }

  return nodePrivateKey
}

func ExportNodeKeySeed() string {
  tx := db.GetTx(false)
  defer db.RollbackTx(tx)

  cb := tx.Bucket([]byte("config"))
  key := cb.Get([]byte("nodeKey"))

  seed, _ := bip39.NewMnemonic(key)
  return seed
}
