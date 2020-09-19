package main

import (
	gutowire "github.com/Just-maple/go-autowire"
	"github.com/spf13/cobra"
)

const command = "gutowire"

var Command = &cobra.Command{
	Use:   command,
	Short: "gen your wire files effective and simply",
	Run:   run,
}

var (
	filepath string
	scope    string
	pkg      string
	opt      []gutowire.Option
)

func init() {
	f := Command.Flags()
	f.StringVarP(&filepath, "wire path", "w", "", "your wire file path")
	f.StringVarP(&scope, "scope", "s", "", "your dependencies scope path")
	f.StringVarP(&pkg, "pkg", "p", "", "gen file pkg name")
}

func main() {
	_ = Command.Execute()

}

func run(_ *cobra.Command, args []string) {
	if len(pkg) > 0 {
		opt = append(opt, gutowire.WithPkg(pkg))
	}
	if len(scope) > 0 {
		opt = append(opt, gutowire.WithSearchPath(scope))
	}
	if len(filepath) == 0 && len(args) > 0 {
		filepath = args[0]
	}
	if len(filepath) == 0 {
		panic("arg required for your wire file path")
	}
	opt = append(opt, gutowire.InitWire())

	if err := gutowire.RunWire(filepath, opt...); err != nil {
		panic(err)
	}
}
