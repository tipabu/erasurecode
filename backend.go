package erasurecode

/*
#cgo pkg-config: erasurecode-1
#include <stdlib.h>
#include <liberasurecode/erasurecode.h>
*/
import "C"
import "errors"
import "fmt"
import "unsafe"

type ErasureCodeParams struct {
	Name string
	K    int
	M    int
	HD   int
}

type ErasureCodeBackend struct {
	ErasureCodeParams
	libec_desc C.int
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

func bytesFromCharArray(ptr **C.char, index int, read_length C.uint64_t) []byte {
	base := uintptr(unsafe.Pointer(ptr))
	offset := uintptr(index) * unsafe.Sizeof(*ptr)
	item := *(*unsafe.Pointer)(unsafe.Pointer(base + offset))
	return C.GoBytes(item, C.int(read_length))
}

func (backend *ErasureCodeBackend) Encode(data []byte) ([][]byte, error) {
	var data_frags **C.char
	var parity_frags **C.char
	var frag_len C.uint64_t
	c_data := C.CString(string(data))
	defer C.free(unsafe.Pointer(c_data))
	rc := C.liberasurecode_encode(
		backend.libec_desc, c_data, C.uint64_t(len(data)),
		&data_frags, &parity_frags, &frag_len)
	if rc != 0 {
		return nil, errors.New(fmt.Sprintf(
			"encode() returned %v", errToName(-rc)))
	}
	defer C.liberasurecode_encode_cleanup(
		backend.libec_desc, data_frags, parity_frags)
	result := make([][]byte, backend.K+backend.M)
	for i := 0; i < backend.K; i++ {
		result[i] = bytesFromCharArray(data_frags, i, frag_len)
	}
	for i := 0; i < backend.M; i++ {
		result[i+backend.K] = bytesFromCharArray(parity_frags, i, frag_len)
	}
	return result, nil
}

func (backend *ErasureCodeBackend) Decode(frags [][]byte) ([]byte, error) {
	var data *C.char
	var data_len C.uint64_t
	if len(frags) == 0 {
		return nil, errors.New("decoding requires at least one fragment")
	}
	c_frags := (**C.char)(C.malloc(C.size_t(int(unsafe.Sizeof(data)) * len(frags))))
	defer C.free(unsafe.Pointer(c_frags))
	base := uintptr(unsafe.Pointer(c_frags))
	for index, frag := range frags {
		ptr := (**C.char)(unsafe.Pointer(base + uintptr(index)*unsafe.Sizeof(*c_frags)))
		*ptr = C.CString(string(frag))
		defer C.free(unsafe.Pointer(*ptr))
	}
	rc := C.liberasurecode_decode(
		backend.libec_desc, c_frags, C.int(len(frags)),
		C.uint64_t(len(frags[0])), C.int(1),
		&data, &data_len)
	if rc != 0 {
		return nil, errors.New(fmt.Sprintf(
			"decode() returned %v", errToName(-rc)))
	}
	defer C.liberasurecode_decode_cleanup(backend.libec_desc, data)
	return C.GoBytes(unsafe.Pointer(data), C.int(data_len)), nil
}

func (backend *ErasureCodeBackend) Reconstruct(frags [][]byte, frag_index int) ([]byte, error) {
	var data *C.char
	if len(frags) == 0 {
		return nil, errors.New("reconstruction requires at least one fragment")
	}
	frag_len := len(frags[0])
	data = (*C.char)(C.malloc(C.size_t(int(unsafe.Sizeof(*data)) * frag_len)))
	defer C.free(unsafe.Pointer(data))
	c_frags := (**C.char)(C.malloc(C.size_t(int(unsafe.Sizeof(data)) * len(frags))))
	defer C.free(unsafe.Pointer(c_frags))
	base := uintptr(unsafe.Pointer(c_frags))
	for index, frag := range frags {
		ptr := (**C.char)(unsafe.Pointer(base + uintptr(index)*unsafe.Sizeof(*c_frags)))
		*ptr = C.CString(string(frag))
		defer C.free(unsafe.Pointer(*ptr))
	}

	if rc := C.liberasurecode_reconstruct_fragment(
		backend.libec_desc, c_frags, C.int(len(frags)),
		C.uint64_t(len(frags[0])), C.int(frag_index), data); rc != 0 {
		return nil, errors.New(fmt.Sprintf(
			"reconstruct_fragment() returned %v", errToName(-rc)))
	}
	return C.GoBytes(unsafe.Pointer(data), C.int(len(frags[0]))), nil
}

func (backend *ErasureCodeBackend) IsInvalidFragment(frag []byte) bool {
	c_data := C.CString(string(frag))
	defer C.free(unsafe.Pointer(c_data))
	return 1 == C.is_invalid_fragment(backend.libec_desc, c_data)
}
