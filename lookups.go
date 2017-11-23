package erasurecode

/*
#cgo pkg-config: erasurecode-1
#include <stdlib.h>
#include <liberasurecode/erasurecode.h>
*/
import "C"
import "fmt"

func nameToId(name string) (C.ec_backend_id_t, error) {
	switch name {
	case "null":
		return C.EC_BACKEND_NULL, nil
	case "jerasure_rs_vand":
		return C.EC_BACKEND_JERASURE_RS_VAND, nil
	case "jerasure_rs_cauchy":
		return C.EC_BACKEND_JERASURE_RS_CAUCHY, nil
	case "flat_xor_hd":
		return C.EC_BACKEND_FLAT_XOR_HD, nil
	case "isa_l_rs_vand":
		return C.EC_BACKEND_ISA_L_RS_VAND, nil
	case "shss":
		return C.EC_BACKEND_SHSS, nil
	case "liberasurecode_rs_vand":
		return C.EC_BACKEND_LIBERASURECODE_RS_VAND, nil
	case "isa_l_rs_cauchy":
		return C.EC_BACKEND_ISA_L_RS_CAUCHY, nil
	case "libphazr":
		return C.EC_BACKEND_LIBPHAZR, nil
	default:
		return 0, fmt.Errorf("unsupported backend %q", name)
	}
}

func idToName(id C.ec_backend_id_t) string {
	switch id {
	case C.EC_BACKEND_NULL:
		return "null"
	case C.EC_BACKEND_JERASURE_RS_VAND:
		return "jerasure_rs_vand"
	case C.EC_BACKEND_JERASURE_RS_CAUCHY:
		return "jerasure_rs_cauchy"
	case C.EC_BACKEND_FLAT_XOR_HD:
		return "flat_xor_hd"
	case C.EC_BACKEND_ISA_L_RS_VAND:
		return "isa_l_rs_vand"
	case C.EC_BACKEND_SHSS:
		return "shss"
	case C.EC_BACKEND_LIBERASURECODE_RS_VAND:
		return "liberasurecode_rs_vand"
	case C.EC_BACKEND_ISA_L_RS_CAUCHY:
		return "isa_l_rs_cauchy"
	case C.EC_BACKEND_LIBPHAZR:
		return "libphazr"
	default:
		return fmt.Sprintf("<unknown backend id %v>", id)
	}
}

func errToName(errno C.int) string {
	switch errno {
	case C.EBACKENDNOTSUPP:
		return "EBACKENDNOTSUPP"
	case C.EECMETHODNOTIMPL:
		return "EECMETHODNOTIMPL"
	case C.EBACKENDINITERR:
		return "EBACKENDINITERR"
	case C.EBACKENDINUSE:
		return "EBACKENDINUSE"
	case C.EBACKENDNOTAVAIL:
		return "EBACKENDNOTAVAIL"
	case C.EBADCHKSUM:
		return "EBADCHKSUM"
	case C.EINVALIDPARAMS:
		return "EINVALIDPARAMS"
	case C.EBADHEADER:
		return "EBADHEADER"
	case C.EINSUFFFRAGS:
		return "EINSUFFFRAGS"
	default:
		return fmt.Sprintf("<unknown error code %v>", errno)
	}
}
