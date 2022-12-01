package merklehead

import (
  "github.com/ipfs/go-cid"
)

type MerklePathNode struct {
  Name string
  Cid cid.Cid
}

type MerklePath struct {
  path []MerklePathNode
}
