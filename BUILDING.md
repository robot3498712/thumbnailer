
# Dependencies

```
libvips and its dependencies cross-compiled for all supported Windows architectures: https://github.com/libvips/build-win64-mxe 
extract dlls to thumbnailer directory
include the modules as needed (as of writing, "vips-modules-8.18" subfolder)
```


## vips: avif support

```
compile custom libheif with dav1d decoder via vcpkg
move/replace dlls in thumbnailer directory
```


### vcpkg: ports/libheif

```
vcpkg_cmake_configure(
    SOURCE_PATH "${SOURCE_PATH}"
    OPTIONS
        -DBUILD_TESTING=OFF
        -DCMAKE_COMPILE_WARNING_AS_ERROR=OFF 
        "-DCMAKE_PROJECT_INCLUDE=${CURRENT_PORT_DIR}/cmake-project-include.cmake" 
        -DPLUGIN_DIRECTORY= # empty
        -DCMAKE_BUILD_TYPE=MinSizeRel 
        -DWITH_DAV1D=ON 
        -DVCPKG_LOCK_FIND_PACKAGE_LIBDE265=ON   # feature candidate
        -DBUILD_DOCUMENTATION=OFF 
        -DWITH_FUZZERS=OFF 
        -DWITH_EXAMPLES=OFF
        -DWITH_LIBSHARPYUV=OFF
        -DWITH_OpenH264_DECODER=OFF
        -DENABLE_PLUGIN_LOADING=OFF 
        -DWITH_X265=OFF 
		-DWITH_AOM_ENCODER=OFF 
		-DWITH_AOM_DECODER=OFF 
		-DWITH_LIBSHARPYUV=OFF 
		-DWITH_GDK_PIXBUF=OFF 
		-DWITH_OpenJPEG_ENCODER=OFF 
		-DWITH_OpenJPEG_DECODER=OFF 
		-DWITH_OPENJPH_ENCODER=OFF 
		-DWITH_SvtEnc=OFF 
		-DWITH_RAV1E=OFF 
        -DVCPKG_LOCK_FIND_PACKAGE_Brotli=OFF
        -DVCPKG_LOCK_FIND_PACKAGE_Doxygen=OFF
        -DVCPKG_LOCK_FIND_PACKAGE_PNG=OFF
        -DVCPKG_LOCK_FIND_PACKAGE_TIFF=OFF
        ${FEATURE_OPTIONS}
    OPTIONS_RELEASE
        "-DPLUGIN_INSTALL_DIRECTORY=${CURRENT_PACKAGES_DIR}/plugins/libheif"
    OPTIONS_DEBUG
        "-DPLUGIN_INSTALL_DIRECTORY=${CURRENT_PACKAGES_DIR}/debug/plugins/libheif"
    MAYBE_UNUSED_VARIABLES
        VCPKG_LOCK_FIND_PACKAGE_AOM
        VCPKG_LOCK_FIND_PACKAGE_Brotli
        VCPKG_LOCK_FIND_PACKAGE_OpenJPEG
        VCPKG_LOCK_FIND_PACKAGE_X265
        VCPKG_LOCK_FIND_PACKAGE_ZLIB
)

vcpkg install dav1d:x64-windows
vcpkg install libheif[core]:x64-windows --editable
```


## vips: produce go bindings

```
Update the build env:

go get -u ./...
go mod tidy


pkg-config --modversion vips

pacman -Syu
pacman -S mingw-w64-ucrt-x86_64-libvips
```

```
follow instructions via https://github.com/cshum/vipsgen and output to vips directory
typically the pre-generated latest library revision (./vips folder) will work
```


## libmobi (minimal; nodeps)

```
git clone https://github.com/bfabiszewski/libmobi.git
cd libmobi && ./autogen.sh
CFLAGS="-O2 -s" LDFLAGS="-s" ./configure --prefix="/{PATH_TO}/libmobi" --disable-encryption --with-zlib=no --with-libxml2=no && make -j$(nproc)

strip --strip-all libmobi.dll
copy libmobi.dll to thumbnailer directory
```


### libmobi: go bindings

```
copy source files into thumbnailer/go-mobi/vendor/libmobi/src
copy libmobi.dll.a into thumbnailer/go-mobi/vendor/libmobi/lib
```


# executable (Windows via msys2-ucrt)

```
windres thumbnailer.rc -O coff -o thumbnailer.syso

export CGO_CFLAGS="-O2 -I/{PATH_TO}/vips-dev-8.18/lib"
export CGO_LDFLAGS="-O2 -L/{PATH_TO}/vips-dev-8.18/lib""

go build -trimpath -ldflags="-s -w"


# check via thumbnailer -vv
	thumbnailer: 0.2.3
	libvips: 8.18.0
	FzVersion: 1.24.9
```
