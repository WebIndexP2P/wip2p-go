package messages

import (
  //"log"
  //"errors"
  "encoding/json"

  "code.wip2p.com/mwadmin/wip2p-go/sigbundle/sigbundlestruct"
)

type AccountInfo struct {
	ActiveInviter string `json:"activeInviter"`
  ActiveLevel uint `json:"activeLevel"`
  ActiveTimestamp uint `json:"activeTimestamp"`
	ActiveSizeLimit uint `json:"activeSizeLimit"`
	DaysHeadstart uint `json:"daysHeadstart"`
	InvitersCount uint `json:"invitersCount"`
	Local_dateCreated uint `json:"local_dateCreated"`
	NextUpgradeDays uint `json:"nextUpgradeDays"`
	PostCount uint `json:"postcount"`

	Multihash string `json:"multihash,omitempty"`
	Signature string `json:"signature,omitempty"`
	Timestamp uint `json:"timestamp,omitempty"`
	CborData []string `json:"cborData,omitempty"`

	Invited []Invited `json:"invited,omitempty"`
	Inviters []Inviter `json:"inviters,omitempty"`
}

type Invited struct {
  Account string `json:"account"`
  Timestamp uint `json:"timestamp"`
  LvlGap uint `json:"lvlgap"`
}

type Inviter struct {
  Account string `json:"account"`
  //InviterTimestamp string `json:"inviterTimestamp"`
  InviterLevel uint `json:"inviterLevel"`
  Timestamp uint `json:"timestamp"`
  LvlGap uint `json:"lvlgap"`
}

func (a *AccountInfo) ToSigBundle(account string) (*sigbundlestruct.SigBundle, error) {

  b := Bundle{}
  b.Account = account
  b.Timestamp = uint64(a.Timestamp)
  b.Multihash = a.Multihash
  b.Signature = a.Signature
  b.CborData = a.CborData
  sigBundle, err := b.ToSigBundle()
  if err != nil {
    return nil, err
  }
  return sigBundle, nil
}

func ParseAccountInfo(data interface{}) (AccountInfo, error) {

  var info AccountInfo
  responseB, _ := json.Marshal(data)
  json.Unmarshal(responseB, &info)

  return info, nil
}
