// protoc-gen-sol by Celer Network Team

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/celer-network/protoc-gen-sol/generator"
	"github.com/golang/protobuf/proto"
)

var showver = flag.Bool("v", false, "Show version and exit")

func main() {
	flag.Parse()
	if *showver {
		printver()
		os.Exit(0)
	}
	// Begin by allocating a generator. The request and response structures are stored there
	// so we can do error handling easily - the response structure contains the field to
	// report failure.
	g := generator.New()

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		generator.Error(err, "reading input")
	}

	if err := proto.Unmarshal(data, g.Request); err != nil {
		generator.Error(err, "parsing input proto")
	}
	g.ParseParams()
	g.GenerateAllFiles()

	// Send back the results.
	data, err = proto.Marshal(g.Response)
	if err != nil {
		generator.Error(err, "failed to marshal output proto")
	}
	_, err = os.Stdout.Write(data)
	if err != nil {
		generator.Error(err, "failed to write output proto")
	}
}

var (
	version string
	commit  string
)

func printver() {
	fmt.Println("Version:", version)
	fmt.Println("Commit:", commit)
}
