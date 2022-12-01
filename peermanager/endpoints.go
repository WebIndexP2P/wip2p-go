package peermanager

var uniqueEndpointsMap map[string]bool

func addEndpoint(ep string) bool {
  if ok := uniqueEndpointsMap[ep]; ok == true {
    return false
  } else {
    uniqueEndpointsMap[ep] = true
    return true
  }
}

func uniqueEndpointExists(ep string) bool {
  return uniqueEndpointsMap[ep]
}
