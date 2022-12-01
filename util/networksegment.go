package util

import (
  "net"
  "strings"
)

func GetNetworkSegment(hostOrIP string) string {
  netSegment := "public"

  if strings.HasSuffix(hostOrIP, ".i2p") {
    netSegment = "i2p"
  } else {
    IP := net.ParseIP(hostOrIP)
    if (IP == nil) {
      // assume its a public internet host / IP
      addrs, err := net.LookupHost(hostOrIP)
      if err != nil {
        //assume its public (already set above)
        return netSegment
      }
      IP = net.ParseIP(addrs[0])
    }

    _, ygg200BitBlock, _ := net.ParseCIDR("200::/7")
    _, ygg300BitBlock, _ := net.ParseCIDR("300::/8")
    ygg := ygg200BitBlock.Contains(IP) || ygg300BitBlock.Contains(IP)
    if ygg {
      netSegment = "ygg"
    }
    _, private24BitBlock, _ := net.ParseCIDR("10.0.0.0/8")
    _, private20BitBlock, _ := net.ParseCIDR("172.16.0.0/12")
    _, private16BitBlock, _ := net.ParseCIDR("192.168.0.0/16")
    private := private24BitBlock.Contains(IP) || private20BitBlock.Contains(IP) || private16BitBlock.Contains(IP)
    if private {
      netSegment = "private"
    }
  }

  return netSegment
}
