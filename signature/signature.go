package signature

import (
  "fmt"
  "errors"
  "strconv"
  "encoding/hex"
  "crypto/ecdsa"
  "github.com/ethereum/go-ethereum/crypto"
)

func Recover(hash []byte, signature []byte) ([]byte, error) {


  tmpSig := signature
  // convert v from normalized back to ethereum compatible 0/1
  tmpSig[64] = tmpSig[64] - 27

  recPubKeyB, err := crypto.Ecrecover(hash, tmpSig)
  if err != nil {
    fmt.Println(err)
    return nil, errors.New("problem with signature")
  }

  recPubKey, _ := crypto.UnmarshalPubkey(recPubKeyB)
  recAddress := crypto.PubkeyToAddress(*recPubKey)

  return recAddress.Bytes(), nil
}

func Sign(hash []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
  signature, err := crypto.Sign(hash, privateKey)
  signature[64] = signature[64] + 27
  return signature, err
}

func EthHashBytes(data []byte) ([]byte, error) {
  signedString := "0x" + hex.EncodeToString(data)
  signedString = "\x19Ethereum Signed Message:\n" + strconv.Itoa(len(signedString)) + signedString
  signedBytes := []byte(signedString)
  hash := crypto.Keccak256Hash(signedBytes)
  return hash.Bytes(), nil
}

func EthHashString(msg string) ([]byte, error) {
  signedString := msg
  signedString = "\x19Ethereum Signed Message:\n" + strconv.Itoa(len(signedString)) + signedString
  signedBytes := []byte(signedString)
  hash := crypto.Keccak256Hash(signedBytes)
  return hash.Bytes(), nil
}
