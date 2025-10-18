
### Windows via msys2-ucrt (preferred)

* export PKG_CONFIG_PATH="{PATH_TO}/vcpkg/packages/libheif_x64-windows/lib/pkgconfig:$PKG_CONFIG_PATH"
* export PKG_CONFIG_PATH="{PATH_TO}/vcpkg/packages/libde265_x64-windows/lib/pkgconfig:$PKG_CONFIG_PATH"
* export PKG_CONFIG_PATH="{PATH_TO}/vcpkg/packages/x265_x64-windows/lib/pkgconfig:$PKG_CONFIG_PATH"

* export CGO_LDFLAGS="-O2 -g -L/{PATH_TO}/msys64/ucrt64/lib"
* export CGO_CFLAGS="-O2 -g -L/{PATH_TO}/msys64/ucrt64/lib"

* windres thumbnailer.rc -O coff -o thumbnailer.syso

* go build -ldflags "-s -w"


### libheif

* see build instructions via https://github.com/strukturag/libheif
