
### Windows via msys2-ucrt (preferred)

* For reference see build instructions via https://github.com/strukturag/libheif

#### libheif via vcpkg

```
./bootstrap-vcpkg.sh
./vcpkg.exe integrate install

EDIT ports/libheif/portfile.cmake ->
  "default-features": [
  ],

./vcpkg install libheif:x64-windows --editable
```

#### libheif manually

```
pacman -S mingw-w64-ucrt-x86_64-libde265
	.. or mingw-w64-ucrt-x86_64-libheif for all dependencies

pacman -S mingw-w64-ucrt-x86_64-cmake

git clone https://github.com/strukturag/libheif.git
cd libheif
mkdir build
cd build

cmake -G Ninja \
-DCMAKE_C_FLAGS="-O2 -ffunction-sections -fdata-sections" \
-DCMAKE_CXX_FLAGS="-O2 -ffunction-sections -fdata-sections" \
-DCMAKE_SHARED_LINKER_FLAGS="-Wl,--gc-sections -Wl,--strip-all" \
-DCMAKE_EXE_LINKER_FLAGS="-Wl,--gc-sections -Wl,--strip-all" \
-DCMAKE_BUILD_TYPE=MinSizeRel \
-DBUILD_SHARED_LIBS=ON \
-DBUILD_DOCUMENTATION=OFF \
-DWITH_FUZZERS=OFF \
-DBUILD_TESTING=OFF \
-DCMAKE_COMPILE_WARNING_AS_ERROR=OFF \
-DWITH_EXAMPLES=OFF \
-DPLUGIN_DIRECTORY= \
-DENABLE_PLUGIN_LOADING=OFF \
-DWITH_LIBDE265=ON \
-DWITH_X265=OFF \
-DWITH_DAV1D=OFF \
-DWITH_AOM_ENCODER=OFF \
-DWITH_AOM_DECODER=OFF \
-DWITH_LIBSHARPYUV=OFF \
-DWITH_OpenH264_DECODER=OFF \
-DWITH_GDK_PIXBUF=OFF \
-DWITH_OpenJPEG_ENCODER=OFF \
-DWITH_OpenJPEG_DECODER=OFF \
-DWITH_OPENJPH_ENCODER=OFF \
-DWITH_SvtEnc=OFF \
-DWITH_RAV1E=OFF \
..

ninja -j$(nproc)
```

#### libavif

* /gen2brain/avif : The library will first try to use a dynamic/shared library (if installed) via purego and will fall back to WASM.
> WASM, in this case at least, is very slow and inefficient. For this reason prefer building native libs. 
> Following builds a shared lib for decoding only.

```
pacman -S mingw-w64-ucrt-x86_64-nasm

git clone https://aomedia.googlesource.com/aom
cd aom
mkdir build
cd build

cmake -G Ninja \
-DCMAKE_C_FLAGS="-O2 -ffunction-sections -fdata-sections" \
-DCMAKE_CXX_FLAGS="-O2 -ffunction-sections -fdata-sections" \
-DCMAKE_SHARED_LINKER_FLAGS="-Wl,--gc-sections -Wl,--strip-all" \
-DCMAKE_EXE_LINKER_FLAGS="-Wl,--gc-sections -Wl,--strip-all" \
-DCMAKE_BUILD_TYPE=MinSizeRel \
-DBUILD_SHARED_LIBS=0 \
-DCONFIG_PIC=1 \
-DENABLE_CCACHE=0 \
-DENABLE_DOCS=0 \
-DENABLE_EXAMPLES=0 \
-DENABLE_TESTDATA=0 \
-DENABLE_TESTS=0 \
-DENABLE_TOOLS=0 \
-DCONFIG_MULTITHREAD=1 \
-DCONFIG_AV1_DECODER=1 \
-DCONFIG_AV1_ENCODER=0 \
..

ninja -j$(nproc)

git clone https://github.com/AOMediaCodec/libavif.git
cd libavif
mkdir build

cd ext
# refer to libyuv.cmd
git clone --single-branch https://chromium.googlesource.com/libyuv/libyuv
cd libyuv
git checkout {commit_revision}
mkdir build
cd build

cmake -G Ninja \
-DCMAKE_C_FLAGS="-O2 -ffunction-sections -fdata-sections" \
-DCMAKE_CXX_FLAGS="-O2 -ffunction-sections -fdata-sections" \
-DCMAKE_SHARED_LINKER_FLAGS="-Wl,--gc-sections -Wl,--strip-all" \
-DCMAKE_EXE_LINKER_FLAGS="-Wl,--gc-sections -Wl,--strip-all" \
-DCMAKE_BUILD_TYPE=MinSizeRel \
-DBUILD_SHARED_LIBS=OFF \
-DCMAKE_POSITION_INDEPENDENT_CODE=ON \
..

ninja -j$(nproc)

# local build searches for libs here:
	libavif/ext/libyuv/build/libyuv.a
	libavif/ext/aom/build.libavif/libaom.a

cd libavif/build

cmake -G Ninja \
-DCMAKE_C_FLAGS="-O2 -ffunction-sections -fdata-sections" \
-DCMAKE_CXX_FLAGS="-O2 -ffunction-sections -fdata-sections" \
-DCMAKE_SHARED_LINKER_FLAGS="-Wl,--gc-sections -Wl,--strip-all" \
-DCMAKE_EXE_LINKER_FLAGS="-Wl,--gc-sections -Wl,--strip-all" \
-DCMAKE_BUILD_TYPE=MinSizeRel \
-DBUILD_SHARED_LIBS=1 \
-DAVIF_CODEC_AOM=LOCAL \
-DAVIF_LOCAL_AOM=1 \
-DAVIF_CODEC_AOM_DECODE=1 \
-DAVIF_CODEC_AOM_ENCODE=0 \
-DAVIF_LIBYUV=LOCAL \
..

ninja -j$(nproc)
```

#### executable

```
windres thumbnailer.rc -O coff -o thumbnailer.syso

export CGO_CFLAGS="-O2 -s -L/{PATH_TO}/msys64/ucrt64/lib"
export CGO_LDFLAGS="-O2 -s -L/{PATH_TO}/msys64/ucrt64/lib"

go build -ldflags="-s -w"

# couple more dlls may be needed, copy from msys64/ucrt64/bin
ldd thumbnailer
	libgcc_s_seh-1.dll
	libstdc++-6.dll
	libwinpthread-1.dll

# check via thumbnailer -vv
	thumbnailer: {version}
	FzVersion: {version}
	libheif: {version}
	libavif: wasm | shared
```
