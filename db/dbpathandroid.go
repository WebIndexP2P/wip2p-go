// +build androidlib

package db

func GetDefaultPath() string {
  return "/data/data/com.example.wip2plauncher/"
}

func IsAndroidLib() bool {
  return true
}
