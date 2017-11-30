package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/tipabu/erasurecode"
)

var backendName = flag.String("b", "", "the backend to use")
var numData = flag.Int("k", 0, "number of data fragments")
var numParity = flag.Int("m", 0, "number of parity fragments")
var wordSize = flag.Int("w", 0, "word size, in bits")
var hammingDistance = flag.Int("d", 0, "Hamming distance, for flat_xor_hd")
var bufferSize = flag.Int("s", 1<<20, "chunk size, in bytes")

func init() {
	flag.Usage = func() {
		fmt.Printf("usage: %s -b backend -k K -m M [-w W] [-d HD] [-s size] file\n\n", os.Args[0])
		fmt.Println("Split a file into K + M fragment archives.")
		flag.PrintDefaults()
		fmt.Println("\nAvailable backends:")
		for _, name := range erasurecode.AvailableBackends() {
			fmt.Println("    " + name)
		}
	}
}

func checkErr(err error) {
	if err != nil {
		flag.Usage()
		fmt.Println()
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	flag.Parse()
	buf := make([]byte, *bufferSize)

	if *backendName == "" {
		checkErr(fmt.Errorf("missing required flag -b"))
	}
	if !erasurecode.BackendIsAvailable(*backendName) {
		checkErr(fmt.Errorf("backend must be one of %v", erasurecode.AvailableBackends()))
	}
	if *numData == 0 {
		checkErr(fmt.Errorf("missing required flag -k"))
	}
	if *numParity == 0 && *backendName != "null" {
		checkErr(fmt.Errorf("missing required flag -m"))
	}
	if len(flag.Args()) != 1 {
		checkErr(fmt.Errorf("expected exactly one file to split"))
	}
	input := flag.Args()[0]
	prefix := &input

	backend, err := erasurecode.InitBackend(erasurecode.ErasureCodeParams{
		Name: *backendName,
		K:    *numData,
		M:    *numParity,
		W:    *wordSize,
		HD:   *hammingDistance,
	})
	checkErr(err)
	defer backend.Close()

	fd, err := os.Open(input)
	checkErr(err)
	defer fd.Close()

	info, err := fd.Stat()
	checkErr(err)

	output, err := backend.GetFileWriter(*prefix, info.Mode())
	checkErr(err)
	defer output.Close()

	n, err := io.CopyBuffer(output, fd, buf)
	checkErr(err)
	fmt.Printf("%v bytes copied\n", n)
}
