package erasurecode

/*
#cgo pkg-config: erasurecode-1
#include <liberasurecode/erasurecode.h>
*/
import "C"

import (
	"fmt"
	"io"
	"os"
)

type ECWriter struct {
	Backend *ErasureCodeBackend
	Writers []io.WriteCloser
}

type ECReader struct {
	Backend *ErasureCodeBackend
	Readers []io.Reader
	buffer  []byte
}

func getWriters(prefix string, n uint8, perm os.FileMode) ([]io.WriteCloser, error) {
	var i, j uint8
	writers := make([]io.WriteCloser, n)
	for i = 0; i < n; i++ {
		fname := fmt.Sprintf("%s#%d", prefix, i)
		file, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
		if err != nil {
			// Clean up the writers we *did* open
			for j = 0; i < j; j++ {
				// Ignoring any errors allong the way
				writers[i].Close()
			}
			return nil, err
		}
		writers[i] = file
	}
	return writers, nil
}

func getReaders(prefix string, n uint8) ([]io.ReadCloser, error) {
	var i, j uint8
	readers := make([]io.ReadCloser, n)
	for i = 0; i < n; i++ {
		file, err := os.Open(fmt.Sprintf("%s#%d", prefix, i))
		if err != nil {
			// Clean up the readers we *did* open
			for j = 0; i < j; j++ {
				// Ignoring any errors allong the way
				readers[i].Close()
			}
			return nil, err
		}
		readers[i] = file
	}
	return readers, nil
}

func (shim ECWriter) Write(p []byte) (int, error) {
	frags, err := shim.Backend.Encode(p)
	if err != nil {
		return 0, err
	}
	for i, writer := range shim.Writers {
		// TODO: check for errors
		writer.Write(frags[i])
	}
	return len(p), nil
}

func (shim ECWriter) Close() error {
	var firstErr error
	for _, writer := range shim.Writers {
		err := writer.Close()
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (shim ECReader) Read(p []byte) (int, error) {
	sz := len(p)
	frags := make([][]byte, len(shim.Readers))

	n := copy(p, shim.buffer)
	shim.buffer = shim.buffer[n:]
	p = p[n:]
	gotEOF := false
	for len(p) > 0 {
		for i, reader := range shim.Readers {
			frag, err := ReadFragment(reader)
			if err == io.EOF {
				gotEOF = true
				break
			}
			if err != nil {
				return sz - len(p), err
			}
			frags[i] = frag
		}
		if gotEOF {
			break
		}
		decoded, err := shim.Backend.Decode(frags)
		if err != nil {
			return sz - len(p), err
		}

		shim.buffer = decoded
		n := copy(p, shim.buffer)
		shim.buffer = shim.buffer[n:]
		p = p[n:]
	}
	if len(p) == sz && gotEOF {
		return 0, io.EOF
	}

	return sz - len(p), nil
}

func (backend *ErasureCodeBackend) GetFileWriter(prefix string, perm os.FileMode) (io.WriteCloser, error) {
	writers, err := getWriters(prefix, uint8(backend.K+backend.M), perm)
	if err != nil {
		return nil, err
	}
	return ECWriter{backend, writers}, nil
}

func ReadFragmentHeader(reader io.Reader) (FragmentInfo, []byte, error) {
	header := make([]byte, C.sizeof_struct_fragment_header_s)
	n, err := io.ReadFull(reader, header)
	if err != nil {
		return FragmentInfo{}, header[:n], err
	}
	info := GetFragmentInfo(header)

	if !info.IsValid {
		return FragmentInfo{}, header, fmt.Errorf("Metadata checksum failed")
	}
	return info, header, err
}

func ReadFragment(reader io.Reader) ([]byte, error) {
	info, header, err := ReadFragmentHeader(reader)
	if err != nil {
		return header, err
	}

	frag := make([]byte, len(header)+info.Size)
	copy(frag, header)
	n, err := io.ReadFull(reader, frag[len(header):])
	if err != nil {
		return frag[:len(header)+n], err
	}

	return frag, nil
}

func GuessParams(readers []io.Reader) (params ErasureCodeParams, err error) {
	var firstInfo *FragmentInfo
	for i, reader := range readers {
		info, _, headerErr := ReadFragmentHeader(reader)
		if headerErr != nil {
			err = headerErr
			return
		}

		if firstInfo == nil {
			firstInfo = &info
			params.Name = info.BackendName
			continue
		}

		// Validate that we seem to be talking about the same original data
		if info.BackendId != (*firstInfo).BackendId {
			err = fmt.Errorf("Backend mismatch on reader %d; got %d, expected %d", i, info.BackendName, (*firstInfo).BackendName)
			return
		}
		if info.Size != (*firstInfo).Size {
			err = fmt.Errorf("Size mismatch on reader %d; got %d, expected %d", i, info.Size, (*firstInfo).Size)
			return
		}
		if info.OrigDataSize != (*firstInfo).OrigDataSize {
			err = fmt.Errorf("OrigDataSize mismatch on reader %d; got %d, expected %d", i, info.OrigDataSize, (*firstInfo).OrigDataSize)
			return
		}

		if info.Index >= params.M {
			params.M = info.Index + 1
		}
	}
	return
}
