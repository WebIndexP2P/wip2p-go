package util

import (
  //"fmt"
  "bytes"
  "encoding/binary"
)

func RemoveIndex(s []int, index int) []int {
    return append(s[:index], s[index+1:]...)
}

func AllZero(s []byte) bool {
    for _, v := range s {
        if v != 0 {
            return false
        }
    }
    return true
}

func Itob(v uint64) []byte {
  seqB := make([]byte, 8)
  binary.BigEndian.PutUint64(seqB, v)

  return seqB
}

func Btoi(b []byte) uint64 {

  var result uint64

  buf := bytes.NewReader(b)
  binary.Read(buf, binary.BigEndian, &result)

  return result
}
