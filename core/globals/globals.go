package globals

import (
  "crypto/ecdsa"
)

var NodePrivateKey *ecdsa.PrivateKey
var RootAccount [20]byte
var PublicMode bool
var DebugLogging bool
//var LatestActivePeerByLevel []uint // [timestamp,...] - timestamp can be last authed connection or last signed bundle
var FirstRun bool
var Endpoints string // comma delimited list of endpoints
var SequenceSeed uint // unix timestamp of when sequence started, used to reset sync checkpoint with peers
var LatestSequenceNo uint
var StartShutDown bool
var NextAuthIsRoot bool
var NextAuthGetsInvite bool
var GetRootFromNextPeer bool
var ListenPort uint
var AppName string = "wip2p-go"
var AppVersion = "0.7.8"
var AccountDataSizeLimit = 256 * 1024
var AndroidCallback func(string)
var EnableApi bool
