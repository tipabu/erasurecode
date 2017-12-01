package erasurecode

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func tempDir() string {
	base := strings.TrimRight(os.TempDir(), "/")
	dir := base + "/erasurecode_test/" // TODO: add random suffix
	fmt.Println(dir)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		panic(err)
	}
	return dir
}

func TestWriting(t *testing.T) {
	base := tempDir()
	defer os.RemoveAll(base)

	params := validParams[0]
	pattern := testPatterns[0]
	backend, err := InitBackend(params)
	if err != nil {
		t.Errorf("Error creating backend %v: %q", params, err)
	}
	defer backend.Close()

	writer, err := backend.GetFileWriter(base+"test_frags", 0640)
	if err != nil {
		t.Errorf("Error creating writer: %q", err)
	}

	for index := 0; index < params.K+params.M; index++ {
		fragPath := fmt.Sprintf("%stest_frags#%d", base, index)
		info, err := os.Stat(fragPath)
		if err != nil {
			t.Errorf("Error stat'ing %v: %v", fragPath, err)
			continue
		}
		if info.Size() != 0 {
			t.Errorf("%v: Expected size 0, got %v", fragPath, info.Size())
			continue
		}
		if info.Mode() != 0640 {
			t.Errorf("%v: Expected mode 0640, got 0%o", fragPath, info.Mode())
			continue
		}
	}

	writer.Write(pattern)
	var firstSize int64
	for index := 0; index < params.K+params.M; index++ {
		fragPath := fmt.Sprintf("%stest_frags#%d", base, index)
		info, err := os.Stat(fragPath)
		if err != nil {
			t.Errorf("Error stat'ing %v: %v", fragPath, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%v: Expected size to increase, but it's still 0", fragPath)
			continue
		}
		if firstSize == 0 {
			firstSize = info.Size()
		} else if info.Size() != firstSize {
			t.Errorf("%v: Expected all fragments to be the same size (%v), but got %v", fragPath, firstSize, info.Size())
		}

		fd, err := os.Open(fragPath)
		if err != nil {
			t.Errorf("%v: %v", fragPath, err)
			continue
		}
		defer fd.Close()
		frag, err := ReadFragment(fd)
		if err != nil {
			t.Errorf("%v: %v", fragPath, err)
			continue
		}

		// Only wrote the one frag
		junkData, err := ReadFragment(fd)
		if err != io.EOF {
			t.Errorf("%v: Expected EOF, got %v (and data %v)", fragPath, err, junkData)
			continue
		}

		fragInfo := GetFragmentInfo(frag)
		if !fragInfo.IsValid {
			t.Errorf("%v: Expected fragment to be valid", fragPath)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Error closing writer: %q", err)
	}
	if writer.Close() == nil {
		t.Fatal("Expected error when closing an already-closed writer.")
	}
}
