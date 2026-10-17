// Command lazyreview-terminal is lazyreview in a terminal: the same reviews
// as the web app, with Vim keys, as a second client of the same server.
// lazyreview terminal builds it and runs it.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/connorb645/lazyreview/terminal/internal/app"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

func main() {
	tr := i18n.EnglishTranslationSet()
	serverFile := flag.String("server-file", "", tr.ServerFileFlag)
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, tr.Usage)
	}
	flag.Parse()
	if *serverFile == "" || flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := app.Run(*serverFile, flag.Arg(0)); err != nil {
		fmt.Fprintln(os.Stderr, tr.TerminalFailed+":", err)
		os.Exit(1)
	}
}
