package messages

import (
  //"fmt"
  //"errors"
  "encoding/json"
)

type SequenceListItem struct {
  SeqNo uint `json:"seqNo"`
  Account string `json:"account"`
  Timestamp uint `json:"timestamp,omitempty"`
  Removed bool `json:"removed,omitempty"`
}

func ParseSequenceListItem(data interface{}) (SequenceListItem, error) {

  var seqListItem SequenceListItem
  responseB, _ := json.Marshal(data)
  json.Unmarshal(responseB, &seqListItem)

  return seqListItem, nil
}
