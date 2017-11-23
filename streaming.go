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

func (backend *ErasureCodeBackend) GetFileWriter(prefix string, perm os.FileMode) (io.WriteCloser, error) {
	writers, err := getWriters(prefix, uint8(backend.K+backend.M), perm)
	if err != nil {
		return nil, err
	}
	return ECWriter{backend, writers}, nil
}
