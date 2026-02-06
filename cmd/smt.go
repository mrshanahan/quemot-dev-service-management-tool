package main

import (
	"flag"
	"os"

	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/command"
	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/utils"
)

func rootUsage() {
	utils.PrintErrln("smt <command> <options>")
	utils.PrintErrln("")
	utils.PrintErrln("    Commands:")
	utils.PrintErrf("        new		Create a new project from a template\n")
	utils.PrintErrf("        deploy		Deploy an existing project to a remote server\n")
	utils.PrintErrf("        config		Configure connections to remove servers\n")
	utils.PrintErrln("")
}

func main() {
	code := Run(os.Args)
	os.Exit(code)
}

func Run(args []string) int {
	if len(args) < 2 || args[1] == "-h" || args[1] == "--help" || args[1] == "-?" {
		rootUsage()
		return 0
	}
	cmdStr := args[1]
	var spec command.CommandSpec
	switch cmdStr {
	case "new":
		spec = &command.NewCommandSpec{Args: args[2:]}
	case "deploy":
		spec = &command.DeployCommandSpec{Args: args[2:]}
	case "config":
		spec = &command.ConfigCommandSpec{Args: args[2:]}
	default:
		utils.PrintErrf("error: unrecognized command %s\n\n", cmdStr)
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
