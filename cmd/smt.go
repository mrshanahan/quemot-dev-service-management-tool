package main

import (
	"flag"
	"os"

	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/command"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/utils"
)

func rootUsage() {
	utils.PrintErrf("error: no command provided")
	utils.PrintErrf("usage: smt <command> <options>")
}

func main() {
	code := Run(os.Args)
	os.Exit(code)
}

func Run(args []string) int {
	if len(args) < 2 {
		rootUsage()
		return 1
	}
	cmdStr := args[1]
	var spec command.CommandSpec
	switch cmdStr {
	case "new":
		spec = &command.NewCommandSpec{Args: os.Args[2:]}
	default:
		utils.PrintErrf("error: unrecognized command %s\n", cmdStr)
		rootUsage()
		return 1
	}

	cmd, err := spec.Build()
	if err != nil {
		if err == flag.ErrHelp {
			// fs.Usage()
			// utils.PrintErrln(spec.Usage())
			return 0
		}
		utils.PrintErrf("error: %v\n\n", err)
		// utils.PrintErrln(spec.Usage())
		return 1
	}

	if err = cmd.Invoke(); err != nil {
		utils.PrintErrf("error: %v\n", err)
		return 1
	}

	return 0
}
