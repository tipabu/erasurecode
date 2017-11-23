package erasurecode

/*
#cgo pkg-config: erasurecode-1
#include <stdlib.h>
#include <liberasurecode/erasurecode.h>
// shim to make dereferencing frags easier
void * strArrayItem(char ** arr, int idx) { return arr[idx]; }
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

type Version struct {
	Major    uint
	Minor    uint
	Revision uint
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Revision)
}

func (v Version) Less(other Version) bool {
	if v.Major < other.Major {
		return true
	} else if v.Minor < other.Minor {
		return true
	} else if v.Revision < other.Revision {
		return true
	}
	return false
}

func GetVersion() Version {
	v := C.liberasurecode_get_version()
	return Version{
		Major:    uint(v>>16) & 0xffff,
		Minor:    uint(v>>8) & 0xff,
		Revision: uint(v) & 0xff,
	}
}

var KnownBackends = [...]string{
	"null",
	"jerasure_rs_vand",
	"jerasure_rs_cauchy",
	"flat_xor_hd",
	"isa_l_rs_vand",
	"shss",
	"liberasurecode_rs_vand",
	"isa_l_rs_cauchy",
	"libphazr",
}

func AvailableBackends() (avail []string) {
	for _, name := range KnownBackends {
		if BackendIsAvailable(name) {
			avail = append(avail, name)
		}
	}
	return
}

type ErasureCodeParams struct {
	Name string
	K    int
	M    int
	W    int
	HD   int
}

type ErasureCodeBackend struct {
	ErasureCodeParams
	libec_desc C.int
}

func BackendIsAvailable(name string) bool {
	id, err := nameToId(name)
	if err != nil {
		return false
	}
	return C.liberasurecode_backend_available(id) != 0
}

func InitBackend(params ErasureCodeParams) (ErasureCodeBackend, error) {
	backend := ErasureCodeBackend{params, 0}
	id, err := nameToId(backend.Name)
	if err != nil {
		return backend, err
	}
	desc := C.liberasurecode_instance_create(id, &C.struct_ec_args{
		k:  C.int(backend.K),
		m:  C.int(backend.M),
		w:  C.int(backend.W),
		hd: C.int(backend.HD),
		ct: C.CHKSUM_CRC32,
	})
	if desc < 0 {
		return backend, errors.New(fmt.Sprintf(
			"instance_create() returned %v", errToName(-desc)))
	}
	backend.libec_desc = desc
	return backend, nil
}

func (backend *ErasureCodeBackend) Close() error {
	if backend.libec_desc == 0 {
		return errors.New("backend already closed")
	}
	if rc := C.liberasurecode_instance_destroy(backend.libec_desc); rc != 0 {
		return errors.New(fmt.Sprintf(
			"instance_destroy() returned %v", errToName(-rc)))
	}
	backend.libec_desc = 0
	return nil
}

func (backend *ErasureCodeBackend) Encode(data []byte) ([][]byte, error) {
	var data_frags **C.char
	var parity_frags **C.char
	var frag_len C.uint64_t
	p_data := (*C.char)(unsafe.Pointer(&data[0]))
	if rc := C.liberasurecode_encode(
		backend.libec_desc, p_data, C.uint64_t(len(data)),
		&data_frags, &parity_frags, &frag_len); rc != 0 {
		return nil, errors.New(fmt.Sprintf(
			"encode() returned %v", errToName(-rc)))
	}
	defer C.liberasurecode_encode_cleanup(
		backend.libec_desc, data_frags, parity_frags)
	result := make([][]byte, backend.K+backend.M)
	for i := 0; i < backend.K; i++ {
		result[i] = C.GoBytes(C.strArrayItem(data_frags, C.int(i)), C.int(frag_len))
	}
	for i := 0; i < backend.M; i++ {
		result[i+backend.K] = C.GoBytes(C.strArrayItem(parity_frags, C.int(i)), C.int(frag_len))
	}
	return result, nil
}

func (backend *ErasureCodeBackend) Decode(frags [][]byte) ([]byte, error) {
	var data *C.char
	var data_len C.uint64_t
	if len(frags) == 0 {
		return nil, errors.New("decoding requires at least one fragment")
	}

	c_frags := (**C.char)(C.calloc(C.size_t(len(frags)), C.size_t(int(unsafe.Sizeof(data)))))
	defer C.free(unsafe.Pointer(c_frags))
	base := uintptr(unsafe.Pointer(c_frags))
	for index, frag := range frags {
		ptr := (**C.char)(unsafe.Pointer(base + uintptr(index)*unsafe.Sizeof(*c_frags)))
		*ptr = (*C.char)(unsafe.Pointer(&frag[0]))
	}

	if rc := C.liberasurecode_decode(
		backend.libec_desc, c_frags, C.int(len(frags)),
		C.uint64_t(len(frags[0])), C.int(1),
		&data, &data_len); rc != 0 {
		return nil, errors.New(fmt.Sprintf(
			"decode() returned %v", errToName(-rc)))
	}
	defer C.liberasurecode_decode_cleanup(backend.libec_desc, data)
	return C.GoBytes(unsafe.Pointer(data), C.int(data_len)), nil
}

func (backend *ErasureCodeBackend) Reconstruct(frags [][]byte, frag_index int) ([]byte, error) {
	if len(frags) == 0 {
		return nil, errors.New("reconstruction requires at least one fragment")
	}
	frag_len := len(frags[0])
	data := make([]byte, frag_len)
	p_data := (*C.char)(unsafe.Pointer(&data[0]))

	c_frags := (**C.char)(C.calloc(C.size_t(len(frags)), C.size_t(int(unsafe.Sizeof(p_data)))))
	defer C.free(unsafe.Pointer(c_frags))
	base := uintptr(unsafe.Pointer(c_frags))
	for index, frag := range frags {
		ptr := (**C.char)(unsafe.Pointer(base + uintptr(index)*unsafe.Sizeof(*c_frags)))
		*ptr = (*C.char)(unsafe.Pointer(&frag[0]))
	}

	if rc := C.liberasurecode_reconstruct_fragment(
		backend.libec_desc, c_frags, C.int(len(frags)),
		C.uint64_t(len(frags[0])), C.int(frag_index), p_data); rc != 0 {
		return nil, errors.New(fmt.Sprintf(
			"reconstruct_fragment() returned %v", errToName(-rc)))
	}
	return data, nil
}

func (backend *ErasureCodeBackend) IsInvalidFragment(frag []byte) bool {
	p_data := (*C.char)(unsafe.Pointer(&frag[0]))
	return 1 == C.is_invalid_fragment(backend.libec_desc, p_data)
}
