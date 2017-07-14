package erasurecode

import "bytes"
import "math/rand"
import "testing"

var validParams = []ErasureCodeParams{
	{Name: "liberasurecode_rs_vand", K: 2, M: 1},
	{Name: "liberasurecode_rs_vand", K: 10, M: 4},
	{Name: "liberasurecode_rs_vand", K: 4, M: 3},
	{Name: "liberasurecode_rs_vand", K: 8, M: 4},
	{Name: "liberasurecode_rs_vand", K: 15, M: 4},
	{Name: "isa_l_rs_vand", K: 2, M: 1},
	{Name: "isa_l_rs_vand", K: 10, M: 4},
	{Name: "isa_l_rs_vand", K: 4, M: 3},
	{Name: "isa_l_rs_vand", K: 8, M: 4},
	{Name: "isa_l_rs_vand", K: 15, M: 4},
	{Name: "isa_l_rs_cauchy", K: 2, M: 1},
	{Name: "isa_l_rs_cauchy", K: 10, M: 4},
	{Name: "isa_l_rs_cauchy", K: 4, M: 3},
	{Name: "isa_l_rs_cauchy", K: 8, M: 4},
	{Name: "isa_l_rs_cauchy", K: 15, M: 4},
	{Name: "jerasure_rs_vand", K: 2, M: 1},
	{Name: "jerasure_rs_vand", K: 10, M: 4},
	{Name: "jerasure_rs_vand", K: 4, M: 3},
	{Name: "jerasure_rs_vand", K: 8, M: 4},
	{Name: "jerasure_rs_vand", K: 15, M: 4},
	{Name: "jerasure_rs_cauchy", K: 2, M: 1},
	{Name: "jerasure_rs_cauchy", K: 10, M: 4},
	{Name: "jerasure_rs_cauchy", K: 4, M: 3},
	{Name: "jerasure_rs_cauchy", K: 8, M: 4},
}

var testPatterns = [][]byte{
	bytes.Repeat([]byte{0x00}, 1),
	bytes.Repeat([]byte{0xff}, 1),
	bytes.Repeat([]byte{0x00}, 1<<10),
	bytes.Repeat([]byte{0xff}, 1<<10),
	bytes.Repeat([]byte{0x00}, 1<<20),
	bytes.Repeat([]byte{0xff}, 1<<20),
	bytes.Repeat([]byte{0xf0, 0x0f}, 512),
	bytes.Repeat([]byte{0xde, 0xad, 0xbe, 0xef}, 256),
	bytes.Repeat([]byte{0xaa}, 1024),
	bytes.Repeat([]byte{0x55}, 1024),
}

func shuf(src [][]byte) [][]byte {
	dest := make([][]byte, len(src))
	perm := rand.Perm(len(src))
	for i, v := range perm {
		dest[v] = src[i]
	}
	return dest
}

func TestInitBackend(t *testing.T) {
	for _, params := range validParams {
		backend, err := InitBackend(params)
		if err != nil {
			t.Errorf("%q", err)
			continue
		}
		if backend.libec_desc <= 0 {
			t.Errorf("Expected backend descriptor > 0, got %d", backend.libec_desc)
		}

		if err = backend.Close(); err != nil {
			t.Errorf("%q", err)
		}
		if err = backend.Close(); err == nil {
			t.Errorf("Expected error when closing an already-closed backend.")
		}
	}
}

func TestInitBackendFailure(t *testing.T) {
	cases := []struct {
		name string
		k, m int
		want string
	}{
		{"liberasurecode_rs_vand", -1, 1,
			"instance_create() returned EINVALIDPARAMS"},
		{"liberasurecode_rs_vand", 10, -1,
			"instance_create() returned EINVALIDPARAMS"},
		{"non-existent-backend", 10, 4,
			"unsupported backend \"non-existent-backend\""},
		{"", 10, 4,
			"unsupported backend \"\""},
		{"liberasurecode_rs_vand", 20, 20,
			"instance_create() returned EINVALIDPARAMS"},
	}
	for _, args := range cases {
		params := ErasureCodeParams{args.name, args.k, args.m, 0}
		backend, err := InitBackend(params)
		if err == nil {
			t.Errorf("Expected error when calling InitBackend(%v)",
				params)
			backend.Close()
			continue
		}
		if err.Error() != args.want {
			t.Errorf("InitBackend(%q, %v, %v) produced error %q, want %q",
				args.name, args.k, args.m, err, args.want)
		}
		if backend.libec_desc != 0 {
			t.Errorf("InitBackend(%q, %v, %v) produced backend with descriptor %v, want 0",
				args.name, args.k, args.m, backend.libec_desc)
			backend.Close()
		}
	}
}

func TestEncodeDecode(t *testing.T) {
	for _, params := range validParams {
		if !BackendIsAvailable(params.Name) {
			continue
		}
		backend, err := InitBackend(params)
		if err != nil {
			t.Errorf("Error creating backend %v: %q", params, err)
			continue
		}
		for patternIndex, pattern := range testPatterns {
			frags, err := backend.Encode(pattern)
			if err != nil {
				t.Errorf("Error encoding %v: %q", params, err)
				break
			}

			decode := func(frags [][]byte, description string) bool {
				data, err := backend.Decode(frags)
				if err != nil {
					t.Errorf("%v: %v: %q for pattern %d", description, backend, err, patternIndex)
					return false
				} else if bytes.Compare(data, pattern) != 0 {
					t.Errorf("%v: Expected %v to roundtrip pattern %d, got %q", description, backend, patternIndex, data)
					return false
				}
				return true
			}

			var good bool
			good = decode(frags, "all frags")
			good = good && decode(shuf(frags), "all frags, shuffled")
			good = good && decode(frags[:params.K], "data frags")
			good = good && decode(shuf(frags[:params.K]), "shuffled data frags")
			good = good && decode(frags[params.M:], "with parity frags")
			good = good && decode(shuf(frags[params.M:]), "shuffled parity frags")

			if !good {
				break
			}
		}

		if _, err := backend.Decode([][]byte{}); err == nil {
			t.Errorf("Expected error when decoding from empty fragment array")
		}

		err = backend.Close()
		if err != nil {
			t.Errorf("Error closing backend %v: %q", backend, err)
		}
	}
}

func TestReconstruct(t *testing.T) {
	for _, params := range validParams {
		if !BackendIsAvailable(params.Name) {
			continue
		}
		backend, err := InitBackend(params)
		if err != nil {
			t.Errorf("Error creating backend %v: %q", params, err)
			backend.Close()
			continue
		}
		for patternIndex, pattern := range testPatterns {
			frags, err := backend.Encode(pattern)
			if err != nil {
				t.Errorf("Error encoding %v: %q", params, err)
			}

			reconstruct := func(recon_frags [][]byte, frag_index int, description string) bool {
				data, err := backend.Reconstruct(recon_frags, frag_index)
				if err != nil {
					t.Errorf("%v: %v: %q for pattern %d", description, backend, err, patternIndex)
					return false
				} else if bytes.Compare(data, frags[frag_index]) != 0 {
					t.Errorf("%v: Expected %v to roundtrip pattern %d, got %q", description, backend, patternIndex, data)
					return false
				}
				return true
			}

			var good bool
			good = reconstruct(shuf(frags[:params.K]), params.K+params.M-1, "last frag from data frags")
			good = good && reconstruct(shuf(frags[params.M:]), 0, "first frag with parity frags")
			if !good {
				break
			}
		}

		if _, err := backend.Reconstruct([][]byte{}, 0); err == nil {
			t.Errorf("Expected error when reconstructing from empty fragment array")
		}

		err = backend.Close()
		if err != nil {
			t.Errorf("Error closing backend %v: %q", backend, err)
		}
	}
}

func TestIsInvalid(t *testing.T) {
	for _, params := range validParams {
		if !BackendIsAvailable(params.Name) {
			continue
		}
		backend, err := InitBackend(params)
		if err != nil {
			t.Errorf("Error creating backend %v: %q", params, err)
			backend.Close()
			continue
		}
		for patternIndex, pattern := range testPatterns {
			frags, err := backend.Encode(pattern)
			if err != nil {
				t.Errorf("Error encoding %v: %q", params, err)
				continue
			}
			for index, frag := range frags {
				res := backend.IsInvalidFragment(frag)
				if res {
					t.Errorf("%v: frag %v unexpectedly invalid for pattern %d", backend, index, patternIndex)
				}

				// corrupt the frag
				corruptedByte := rand.Intn(len(frag))
				for 71 <= corruptedByte && corruptedByte < 80 {
					// in the alignment padding -- try again
					corruptedByte = rand.Intn(len(frag))
				}
				frag[corruptedByte] ^= 0xff
				res = backend.IsInvalidFragment(frag)
				if !res {
					t.Errorf("%v: frag %v unexpectedly valid after inverting byte %d for pattern %d", backend, index, corruptedByte, patternIndex)
				}
				frag[corruptedByte] ^= 0xff
				frag[corruptedByte] += 1
				res = backend.IsInvalidFragment(frag)
				if !res {
					t.Errorf("%v: frag %v unexpectedly valid after incrementing byte %d for pattern %d", backend, index, corruptedByte, patternIndex)
				}
				frag[corruptedByte] -= 2
				res = backend.IsInvalidFragment(frag)
				if corruptedByte >= 63 && corruptedByte < 67 && frag[corruptedByte] != 0xff {
					if res {
						t.Errorf("%v: frag %v unexpectedly invalid after decrementing version byte %d for pattern %d", backend, index, corruptedByte, patternIndex)
					}
				} else {
					if !res {
						t.Errorf("%v: frag %v unexpectedly valid after decrementing byte %d for pattern %d", backend, index, corruptedByte, patternIndex)
					}
				}
			}
		}
		err = backend.Close()
		if err != nil {
			t.Errorf("Error closing backend %v: %q", backend, err)
		}
	}
}

func TestBackendIsAvailable(t *testing.T) {
	required_backends := []string{
		"null",
		"flat_xor_hd",
		"liberasurecode_rs_vand",
	}
	optional_backends := []string{
		"isa_l_rs_vand",
		"isa_l_rs_cauchy",
		"jerasure_rs_vand",
		"jerasure_rs_cauchy",
		"shss",
		"libphazr",
	}
	for _, name := range required_backends {
		if !BackendIsAvailable(name) {
			t.Fatalf("%v is not available", name)
		}
	}
	for _, name := range optional_backends {
		if !BackendIsAvailable(name) {
			t.Logf("%v is not available", name)
		}
	}
}
