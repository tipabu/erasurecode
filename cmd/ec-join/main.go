package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/tipabu/erasurecode"
)

var numData = flag.Int("k", 0, "number of data fragments")

func init() {
	flag.Usage = func() {
		fmt.Printf("usage: %s -k K file1 [... fileN]\n\n", os.Args[0])
		fmt.Println("Decode N fragment archives back to the original content.")
		flag.PrintDefaults()
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

	files := make([]*os.File, len(flag.Args()))
	readers := make([]io.Reader, len(flag.Args()))
	for i, fname := range flag.Args() {
		file, err := os.Open(fname)
		checkErr(err)
		files[i] = file
		readers[i] = file
	}
	params, err := erasurecode.GuessParams(readers)
	checkErr(err)

	// can't use bufio.Reader as io.Reader (apparently), so... re-open all of those (and hope the underlying FS didn't change!)
	for _, file := range files {
		_, err = file.Seek(0, 0)
		checkErr(err)
		//readers[i] = file
	}

	params.M -= *numData
	params.K = *numData
	backend, err := erasurecode.InitBackend(params)
	checkErr(err)
	defer backend.Close()

	input := erasurecode.ECReader{Backend: &backend, Readers: readers}

	buf := make([]byte, 4096)
	n, err := io.CopyBuffer(os.Stdout, input, buf)
	checkErr(err)
	fmt.Fprintf(os.Stderr, "%v bytes copied\n", n)
}
