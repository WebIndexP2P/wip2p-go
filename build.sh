export CGO_ENABLED=0

if [ "$2" = "linux" ]; then
  go build -a -ldflags "-s -w" -trimpath -o release/wip2p-go_$1_linux_amd64
  upx release/wip2p-go_$1_linux_amd64
fi

if [ "$2" = "darwin" ]; then
  export GOOS=darwin
  go build -a -ldflags "-s -w" -trimpath -o release/wip2p-go_$1_darwin_amd64
  upx release/wip2p-go_$1_darwin_amd64
fi

if [ "$2" = "windows" ]; then
  export GOOS=windows
  go build -a -ldflags "-s -w" -trimpath -o release/wip2p-go_$1_windows_amd64.exe
  upx release/wip2p-go_$1_windows_amd64.exe
fi

if [ "$2" = "riscv" ]; then
  export GOARCH=riscv64
  go build -a -ldflags "-s -w" -trimpath -o release/wip2p-go_$1_linux_riscv64
fi

if [ "$2" = "android" ]; then
  export GOARCH=arm
  export GOARM=7
  go build -a -ldflags "-s -w" -trimpath -o release/wip2p-go_$1_linux_armv7
  upx release/wip2p-go_$1_linux_armv7
fi

if [ "$2" = "androidlib" ]; then
  gomobile bind -androidapi 19 --tags androidlib
fi
