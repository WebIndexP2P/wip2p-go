package messages

import (
  //"fmt"
  "errors"
)

type PeerInfo struct {
  Merklehead string `json:"merklehead"`
  RootAccount string `json:"rootAccount"`
  Endpoints string `json:"endpoints"`
  Version string `json:"version"`
  SequenceSeed uint `json:"sequenceSeed"`
  LatestSequenceNo uint `json:"latestSequenceNo"`
}

func ParsePeerInfo(info interface{}) (PeerInfo, error) {

  i := PeerInfo{}

  infoObj, ok := info.(map[string]interface{})
  if !ok {
    return i, errors.New("expects object")
  }

  var tmpFloat64 float64
  i.Merklehead, _ = infoObj["merklehead"].(string)
  i.RootAccount, _ = infoObj["rootAccount"].(string)
  i.Endpoints, _ = infoObj["endpoints"].(string)
  i.Version, _ = infoObj["version"].(string)

  tmpFloat64, _ = infoObj["sequenceSeed"].(float64)
  i.SequenceSeed = uint(tmpFloat64)

  tmpFloat64, _ = infoObj["latestSequenceNo"].(float64)
  i.LatestSequenceNo = uint(tmpFloat64)

  return i, nil
}
