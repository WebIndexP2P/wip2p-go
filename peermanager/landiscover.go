package peermanager

import (
  "log"
  "fmt"
  "net"
  "time"
  "errors"
  "strconv"
  "strings"

  "code.wip2p.com/mwadmin/wip2p-go/core/globals"
)

var ourIps []string

func StartLanDiscovery() {

//  go StartListen()
  fmt.Println("Starting LAN Discovery on UDP port 31458")

  var err error
  ourIps, err = findSystemIPs()

  pc, err := net.ListenPacket("udp4", ":31458")
  if err != nil {
    log.Println("Could not bind port, LAN discovery disabled")
    return
  }
  //defer pc.Close()

  // listen
  go func(){
    buf := make([]byte, 1024)

    for true {
      n, addr, err := pc.ReadFrom(buf)
      if err != nil {
        panic(err)
      }

      theirIpPort := strings.Split(addr.String(), ":")
      bItsOurs := false
      for _, ourIp := range(ourIps) {
        if theirIpPort[0] == ourIp {
          bItsOurs = true
          continue
        }
      }
      if bItsOurs {
        continue
      }

      msg := string(buf[:n])
      if strings.HasPrefix(msg, "wip2p ") == false {
        continue
      }

      theirWip2pPort := msg[6:]
      targetUrl := fmt.Sprintf("ws://%s:%s", theirIpPort[0], theirWip2pPort)
      wasAdded := AddEndpointStringToQueue(targetUrl)
      if wasAdded {
        log.Printf("Adding LAN peer @ %s\n", targetUrl)
      }
    }

  }()

  addr, err := net.ResolveUDPAddr("udp4", "255.255.255.255:31458")
  if err != nil {
    panic(err)
  }

  for true {
    payload := "wip2p " + strconv.FormatUint(uint64(globals.ListenPort), 10)
    _, err = pc.WriteTo([]byte(payload), addr)
    if err != nil {
      if strings.Contains(err.Error(), "network is unreachable") {
      } else {
        panic(err)
      }
    }

    time.Sleep(30 * time.Second)
  }

}

func findSystemIPs() ([]string, error) {

  ourIps := make([]string, 0)

	intfs, err := net.Interfaces()
	if err != nil {
		return ourIps, err
	}
	// mapping between network interface name and index
	// https://golang.org/pkg/net/#Interface
	for _, intf := range intfs {
		// skip down interface & check next intf
		if intf.Flags&net.FlagUp == 0 {
			continue
		}
		// skip loopback & check next intf
		if intf.Flags&net.FlagLoopback != 0 {
			continue
		}
		// list of unicast interface addresses for specific interface
		// https://golang.org/pkg/net/#Interface.Addrs
		addrs, err := intf.Addrs()
		if err != nil {
			return ourIps, err
		}
		// network end point address
		// https://golang.org/pkg/net/#Addr
		for _, addr := range addrs {
			var ip net.IP
			// Addr type switch required as a result of IPNet & IPAddr return in
			// https://golang.org/src/net/interface_windows.go?h=interfaceAddrTable
			switch v := addr.(type) {
			// net.IPNet satisfies Addr interface
			// since it contains Network() & String()
			// https://golang.org/pkg/net/#IPNet
			case *net.IPNet:
				ip = v.IP
			// net.IPAddr satisfies Addr interface
			// since it contains Network() & String()
			// https://golang.org/pkg/net/#IPAddr
			case *net.IPAddr:
				ip = v.IP
			}
			// skip loopback & check next addr
			if ip == nil || ip.IsLoopback() {
				continue
			}
			// convert IP IPv4 address to 4-byte
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			// return IP address as string
			ourIps = append(ourIps, ip.String())
		}
	}
	return ourIps, errors.New("no ip interface up")
}
