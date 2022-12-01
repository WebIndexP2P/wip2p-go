package sigbundlestruct

import (
	"fmt"
	"crypto/ecdsa"
	"encoding/hex"

	"github.com/ipfs/go-ipld-cbor"
	mh "github.com/multiformats/go-multihash"
	"github.com/ethereum/go-ethereum/crypto"

	"code.wip2p.com/mwadmin/wip2p-go/signature"
)

type SigBundle struct {
	Account []byte
	CborData [][]byte
	Car []byte
	RootMultihash []byte
	Signature []byte
	Timestamp uint64
}

func CreateBundle(timestamp uint, objData interface{}, privateKey *ecdsa.PrivateKey) (sigBundle *SigBundle, err error) {

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
  bundle := SigBundle{
    CborData: [][]byte{ nd.RawData() },
    Signature: sigB,
    Timestamp: uint64(timestamp),
    RootMultihash: nd.Cid().Hash(),
    Account: author,
  }

	return &bundle, nil
}
