package erasurecode

import "bytes"
import "math/rand"
import "testing"

var validArgs = []struct {
	name string
	k, m int
}{
	{"liberasurecode_rs_vand", 2, 1},
	{"liberasurecode_rs_vand", 10, 4},
	{"isa_l_rs_vand", 10, 4},
	{"jerasure_rs_vand", 10, 4},
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
	for _, args := range validArgs {
		backend, err := InitBackend(args.name, args.k, args.m)
		if err != nil {
			t.Errorf("%q", err)
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
		backend, err := InitBackend(args.name, args.k, args.m)
		if err == nil {
			t.Errorf("Expected error when calling InitBackend(%q, %v, %v)",
				args.name, args.k, args.m)
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
	for _, args := range validArgs {
		backend, err := InitBackend(args.name, args.k, args.m)
		if err != nil {
			t.Errorf("Error creating backend %v: %q", args, err)
			backend.Close()
			continue
		}
		for patternIndex, pattern := range testPatterns {
			frags, err := backend.Encode(pattern)
			if err != nil {
				t.Errorf("Error encoding %v: %q", args, err)
			}

			decode := func(frags [][]byte, description string) {
				data, err := backend.Decode(frags)
				if err != nil {
					t.Errorf("%v: %v: %q for pattern %d", description, backend, err, patternIndex)
				} else if bytes.Compare(data, pattern) != 0 {
					t.Errorf("%v: Expected %v to roundtrip pattern %d, got %q", description, backend, patternIndex, data)
				}
			}

			decode(frags, "all frags")
			decode(shuf(frags), "all frags, shuffled")
			decode(frags[:args.k], "data frags")
			decode(shuf(frags[:args.k]), "shuffled data frags")
			decode(frags[args.m:], "with parity frags")
			decode(shuf(frags[args.m:]), "shuffled parity frags")
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
	for _, args := range validArgs {
		backend, err := InitBackend(args.name, args.k, args.m)
		if err != nil {
			t.Errorf("Error creating backend %v: %q", args, err)
			backend.Close()
			continue
		}
		for patternIndex, pattern := range testPatterns {
			frags, err := backend.Encode(pattern)
			if err != nil {
				t.Errorf("Error encoding %v: %q", args, err)
			}

			reconstruct := func(recon_frags [][]byte, frag_index int, description string) {
				data, err := backend.Reconstruct(recon_frags, frag_index)
				if err != nil {
					t.Errorf("%v: %v: %q for pattern %d", description, backend, err, patternIndex)
				} else if bytes.Compare(data, frags[frag_index]) != 0 {
					t.Errorf("%v: Expected %v to roundtrip pattern %d, got %q", description, backend, patternIndex, data)
				}
			}

			reconstruct(shuf(frags[:args.k]), args.k+args.m-1, "last frag from data frags")
			reconstruct(shuf(frags[args.m:]), 0, "first frag with parity frags")
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
	for _, args := range validArgs {
		backend, err := InitBackend(args.name, args.k, args.m)
		if err != nil {
			t.Errorf("Error creating backend %v: %q", args, err)
			backend.Close()
			continue
		}
		for patternIndex, pattern := range testPatterns {
			frags, err := backend.Encode(pattern)
			if err != nil {
				t.Errorf("Error encoding %v: %q", args, err)
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
