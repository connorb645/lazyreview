// Command lazyreview-terminal is lazyreview in a terminal: the same reviews
// as the web app, with Vim keys, as a second client of the same server.
// lazyreview terminal builds it and runs it.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/connorb645/lazyreview/terminal/internal/app"
)

func main() {
	serverFile := flag.String("server-file", "", "the lazyreview server's own record, server.json, holding its address and token")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: lazyreview-terminal --server-file <server.json> [<review id>]")
	}
	flag.Parse()
	if *serverFile == "" || flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := app.Run(*serverFile, flag.Arg(0)); err != nil {
		fmt.Fprintln(os.Stderr, "lazyreview terminal:", err)
		os.Exit(1)
	}
}
