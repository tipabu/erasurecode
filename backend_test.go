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
	{Name: "jerasure_rs_cauchy", K: 15, M: 4, W: 5},
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
		if !BackendIsAvailable(params.Name) {
			continue
		}
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
		params ErasureCodeParams
		want   string
	}{
		{ErasureCodeParams{Name: "liberasurecode_rs_vand", K: -1, M: 1},
			"instance_create() returned EINVALIDPARAMS"},
		{ErasureCodeParams{Name: "liberasurecode_rs_vand", K: 10, M: -1},
			"instance_create() returned EINVALIDPARAMS"},
		{ErasureCodeParams{Name: "non-existent-backend", K: 10, M: 4},
			"unsupported backend \"non-existent-backend\""},
		{ErasureCodeParams{Name: "", K: 10, M: 4},
			"unsupported backend \"\""},
		{ErasureCodeParams{Name: "liberasurecode_rs_vand", K: 20, M: 20},
			"instance_create() returned EINVALIDPARAMS"},
	}
	for _, args := range cases {
		backend, err := InitBackend(args.params)
		if err == nil {
			t.Errorf("Expected error when calling InitBackend(%v)",
				args.params)
			backend.Close()
			continue
		}
		if err.Error() != args.want {
			t.Errorf("InitBackend(%v) produced error %q, want %q",
				args.params, err, args.want)
		}
		if backend.libec_desc != 0 {
			t.Errorf("InitBackend(%v) produced backend with descriptor %v, want 0",
				args.params, backend.libec_desc)
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

func TestIsInvalidFragment(t *testing.T) {
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
				if backend.IsInvalidFragment(frag) {
					t.Errorf("%v: frag %v unexpectedly invalid for pattern %d", backend, index, patternIndex)
				}
				fragCopy := make([]byte, len(frag))
				copy(fragCopy, frag)

				// corrupt the frag
				corruptedByte := rand.Intn(len(frag))
				for 71 <= corruptedByte && corruptedByte < 80 {
					// in the alignment padding -- try again
					corruptedByte = rand.Intn(len(frag))
				}
				frag[corruptedByte] ^= 0xff
				if !backend.IsInvalidFragment(frag) {
					t.Errorf("%v: frag %v unexpectedly valid after inverting byte %d for pattern %d", backend, index, corruptedByte, patternIndex)
				}
				if corruptedByte < 4 || 8 <= corruptedByte && corruptedByte <= 59 {
					/** corruption is in metadata; claim we were created by a version of
					 *  libec that predates metadata checksums. Note that
					 *  Note that a corrupted fragment size (bytes 4-7) will lead to a
					 *  segfault when we try to verify the fragment -- there's a reason
					 *  we added metadata checksums!
					 */
					copy(frag[63:67], []byte{9, 1, 1, 0})
					if 20 <= corruptedByte && corruptedByte <= 53 {
						/** Corrupted data checksum type or data checksum
						 *  We may or may not detect this type of error; in particular,
						 *      - if data checksum type is not in ec_checksum_type_t,
						 *        it is ignored
						 *      - if data checksum is mangled, we may still be valid
						 *        under the "alternative" CRC32; this seems more likely
						 *        with the byte inversion when the data is short
						 *  Either way, though, clearing the checksum type should make
						 *  it pass.
						 */
						frag[20] = 0
						if backend.IsInvalidFragment(frag) {
							t.Errorf("%v: frag %v unexpectedly invalid after clearing metadata crc and disabling data crc", backend, index)
						}
					} else if corruptedByte >= 54 || 0 <= corruptedByte && corruptedByte < 4 {
						/** Some corruptions of some bytes are still detectable. Since we're
						 *  inverting the byte, we can detect:
						 *      - frag index -- bytes 0-3
						 *      - data checksum type -- byte 20
						 *      - data checksum mismatch -- byte 54
						 *      - backend id -- byte 55
						 *      - backend version -- bytes 56-59
						 */
						if !backend.IsInvalidFragment(frag) {
							t.Errorf("%v: frag %v unexpectedly still valid after clearing metadata crc", backend, index)
						}
					} else {
						if backend.IsInvalidFragment(frag) {
							t.Errorf("%v: frag %v unexpectedly invalid after clearing metadata crc", backend, index)
						}
					}
				} else if corruptedByte >= 67 {
					copy(frag[20:25], []byte{1, 0, 0, 0, 0})
					// And since we've changed the metadata, roll back version as above...
					copy(frag[63:67], []byte{9, 1, 1, 0})
					if backend.IsInvalidFragment(frag) {
						t.Errorf("%v: frag %v unexpectedly invalid after clearing data crc", backend, index)
						t.FailNow()
					}
				}
				frag[corruptedByte] ^= 0xff
				copy(frag[63:67], fragCopy[63:67])
				copy(frag[20:25], fragCopy[20:25])

				if bytes.Compare(frag, fragCopy) != 0 {
					for i, orig := range fragCopy {
						if frag[i] != orig {
							t.Logf("%v != %v at index %v", frag[i], orig, i)
						}
					}
					t.Fatal(corruptedByte, frag, fragCopy)
				}

				frag[corruptedByte] += 1
				if !backend.IsInvalidFragment(frag) {
					t.Errorf("%v: frag %v unexpectedly valid after incrementing byte %d for pattern %d", backend, index, corruptedByte, patternIndex)
				}
				frag[corruptedByte] -= 2
				if corruptedByte >= 63 && corruptedByte < 67 && frag[corruptedByte] != 0xff {
					if backend.IsInvalidFragment(frag) {
						t.Errorf("%v: frag %v unexpectedly invalid after decrementing version byte %d for pattern %d", backend, index, corruptedByte, patternIndex)
					}
				} else {
					if !backend.IsInvalidFragment(frag) {
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
