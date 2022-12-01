// +build !androidlib

package db

import (
  "log"
)

func GetDefaultPath() string {
  log.Fatal("not supported")
  return ""
}

func IsAndroidLib() bool {
  return false
}
