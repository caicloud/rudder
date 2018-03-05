package main

import (
	"fmt"
	"os"

	"github.com/caicloud/rudder/cmd/controller/app"
	"github.com/caicloud/rudder/cmd/controller/app/options"
	"github.com/spf13/pflag"
)

func main() {
	s := options.NewReleaseServer()
	s.AddFlags(pflag.CommandLine, app.KnownControllers)

	InitFlags()
	InitLogs()
	defer FlushLogs()

	if err := app.Run(s); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
