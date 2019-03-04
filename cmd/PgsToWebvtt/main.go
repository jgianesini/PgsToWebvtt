package main

import (
	"captainpgs/pkg"
	"github.com/jawher/mow.cli"
	"log"
	"os"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	app := cli.App("pgstowebvtt", "PgsToWebvtt - PGS bitmap converter")
	// app.Version("v version", version.VERSION+"-"+version.GITCOMMIT)

	var (
		src = app.StringArg("FILE", "", "PGS file (*.sup)")
	)

	app.Action = func() {
		pkg.Convert(*src)
	}
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
