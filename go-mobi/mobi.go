package gomobi

/*
#cgo CFLAGS: -I${SRCDIR}/vendor/libmobi/src
#cgo LDFLAGS: -L${SRCDIR}/vendor/libmobi/lib -lmobi
#include <stdlib.h>
#include <stdio.h>
#include <mobi.h>

MOBI_RET load_mobi_from_path(MOBIData *doc, const char *path) {
    FILE *f = fopen(path, "rb");
    if (!f) return MOBI_ERROR;
    MOBI_RET ret = mobi_load_file(doc, f);
    fclose(f);
    return ret;
}
*/
import "C"
import (
	"errors"
	"unsafe"
)

type Mobi struct {
	doc *C.MOBIData
}

func Open(path string) (*Mobi, error) {
	doc := C.mobi_init()
	if doc == nil {
		return nil, errors.New("mobi_init failed")
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	ret := C.load_mobi_from_path(doc, cpath)
	if ret != C.MOBI_SUCCESS {
		C.mobi_free(doc)
		return nil, errors.New("load_mobi_from_path failed")
	}

	return &Mobi{doc: doc}, nil
}

func (m *Mobi) Close() {
	if m.doc != nil {
		C.mobi_free(m.doc)
		m.doc = nil
	}
}

func Version () string {
	v := C.mobi_version()
	return C.GoString(v)
}
