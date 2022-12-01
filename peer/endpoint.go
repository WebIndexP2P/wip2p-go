package peer

import (
  //"fmt"
  "log"
  "time"
  "strings"
  "strconv"
  "net/url"

  "code.wip2p.com/mwadmin/wip2p-go/util"
)

type EndPoint struct {
  Host string
  Port uint
  Secure bool

  NetworkSegment string
  LastConnectAttempt time.Time
  ConsecutiveConnectFails uint
  IsReachable bool
}

func (ep *EndPoint) ToURL() (string) {

  if ep == nil {
    log.Fatal("endpoint.go -> pointer is nil")
  }

  if ep.Host == "" {
    log.Fatal("broken Endpoint")
  }

  var hostString string
  if strings.Index(ep.Host, ":") >= 0{
    hostString = "[" + ep.Host + "]"
  } else {
    hostString = ep.Host
  }

  if ep.Secure {
    if (ep.Port == 443) {
      return "wss://" + hostString
    } else {
      return "wss://" + hostString + ":" + strconv.FormatUint(uint64(ep.Port), 10)
    }
  } else {
    if (ep.Port == 80) {
      return "ws://" + hostString
    } else {
      return "ws://" + hostString + ":" + strconv.FormatUint(uint64(ep.Port), 10)
    }
  }

}

func (ep *EndPoint) Equals(endpoint *EndPoint) (bool) {
  if (ep.Host == endpoint.Host && ep.Port == endpoint.Port) {
    return true
  } else {
    return false
  }
}

func (ep *EndPoint) GetNextTryTime() time.Time {
  dur := time.Duration(ep.ConsecutiveConnectFails * 1000 * 1000 * 1000 * 60)
  nextTryTime := ep.LastConnectAttempt.Add(dur)
  return nextTryTime
}

func ParseEndpointString(endpoint string) EndPoint {

  if endpoint == "" {
    log.Fatal("ParseEndpointString error: endpoint is empty")
  }

  var newEP EndPoint
  var epUrl *url.URL
  var err error
  if strings.HasPrefix(endpoint, "wss://") || strings.HasPrefix(endpoint, "ws://") {
    epUrl, err = url.Parse(endpoint)
  } else {
    epUrl, err = url.Parse("ws://" + endpoint)
  }
  if err != nil {
    log.Fatal(err)
  }

  newEP = EndPoint{}
  if epUrl.Scheme == "wss" {
    newEP.Secure = true
  }

  hostStart := 0
  var hostEnd int
  var portColon int
  if epUrl.Host[0:1] == "[" {
    hostStart = 1
    hostEnd = strings.Index(epUrl.Host, "]")
    portColon = hostEnd + strings.Index(epUrl.Host[hostEnd:], ":")
  } else {
    hostEnd = strings.Index(epUrl.Host, ":")
    portColon = strings.Index(epUrl.Host, ":")
  }

  if hostEnd == -1 {
    hostEnd = len(epUrl.Host)
  }
  newEP.Host = epUrl.Host[hostStart:hostEnd]

  var port uint64
  if portColon == -1 {
    if newEP.Secure {
      port = 443
    } else {
      port = 80
    }
  } else {
    port, _ = strconv.ParseUint(epUrl.Host[portColon+1:], 10, 32)
  }
  newEP.Port = uint(port)

  //default to public internet
  newEP.NetworkSegment = util.GetNetworkSegment(newEP.Host)

  newEP.Port = uint(port)

  return newEP
}
