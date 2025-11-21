package gomobi

/*
#include <mobi.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

// refer to https://github.com/bfabiszewski/libmobi/blob/public/tools/mobitool.c :: dump_cover()

// Cover returns (coverBytes, extension, error)
// Extension is "jpg", "png", "gif", "bmp", or "raw".
func (m *Mobi) Cover() ([]byte, string, error) {
	if m.doc == nil {
		return nil, "", errors.New("not initialized")
	}

	// EXTH record for the cover offset
	exth := C.mobi_get_exthrecord_by_tag(m.doc, C.EXTH_COVEROFFSET)
	if exth == nil {
		return nil, "", errors.New("no EXTH_COVEROFFSET record found")
	}

	// Decode uint32 value
	offset := C.mobi_decode_exthvalue(
		(*C.uchar)(exth.data), // cast unsafe.Pointer -> *C.uchar
		C.size_t(exth.size),   // cast uint32_t -> size_t
	)

	// First resource record index
	first := C.mobi_get_first_resource_record(m.doc)
	if first == C.MOBI_NOTSET {
		return nil, "", errors.New("no resource records")
	}

	// UID = first_resource + offset
	uid := first + C.size_t(offset)

	// Fetch the PDB record by sequence number
	rec := C.mobi_get_record_by_seqnumber(m.doc, uid)
	if rec == nil || rec.size < 4 {
		return nil, "", errors.New("cover not found")
	}

	// Convert to Go slice
	data := C.GoBytes(unsafe.Pointer(rec.data), C.int(rec.size))

	// Detect file extension by magic header
	ext := "raw"

	// JPEG
	if len(data) >= 3 &&
		data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		ext = "jpg"
	} else if len(data) >= 4 &&
		data[0] == 'G' && data[1] == 'I' && data[2] == 'F' && data[3] == '8' {
		ext = "gif"
	} else if len(data) >= 8 &&
		data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 &&
		data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A {
		ext = "png"
	} else if len(data) >= 2 &&
		data[0] == 0x42 && data[1] == 0x4D {
		ext = "bmp"
	}

	return data, ext, nil
}