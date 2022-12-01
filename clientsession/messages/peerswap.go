package messages

import (
  //"errors"
  "encoding/json"
)

func ParsePeerSwap(data interface{}) ([]string, error) {

  var endpoints []string
  responseB, _ := json.Marshal(data)
  err := json.Unmarshal(responseB, &endpoints)

  if err != nil {
    return nil, err
  }

  return endpoints, nil
}
