package main

import (
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:   "release-cli",
	Short: "Useful tools for release",
}

func main() {
	root.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	f := root.PersistentFlags().Lookup("logtostderr")
	f.DefValue = "true"
	_ = f.Value.Set(f.DefValue)
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	_ = fs.Parse([]string{})
	flag.CommandLine = fs
	if err := root.Execute(); err != nil {
		glog.Fatalln(err)
	}
}
