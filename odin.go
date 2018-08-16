package main

import (
	"fmt"
	"os"

	"github.com/coinbase/odin/client"
	"github.com/coinbase/odin/deployer"
	"github.com/coinbase/step/utils/is"
	"github.com/coinbase/step/utils/run"
	"github.com/coinbase/step/utils/to"
)

func main() {
	var arg, command string
	switch len(os.Args) {
	case 1:
		fmt.Println("Starting Lambda")
		run.LambdaTasks(deployer.TaskFunctions())
	case 2:
		command = os.Args[1]
		arg = ""
	case 3:
		command = os.Args[1]
		arg = os.Args[2]
	default:
		printUsage() // Print how to use and exit
	}

	stepFn := to.Strp(os.Getenv("ODIN_STEP"))

	if is.EmptyStr(stepFn) {
		stepFn = to.Strp("coinbase-odin")
	}

	switch command {
	case "json":
		run.JSON(deployer.StateMachine())
	case "deploy":
		// Send Configuration to the deployer
		// arg is a filename
		err := client.Deploy(stepFn, &arg)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	case "fails":
		// List the recent failures and their causes
		err := client.Failures(stepFn)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	case "halt":
		err := client.Halt(stepFn, &arg)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	default:
		printUsage() // Print how to use and exit
	}
}

func printUsage() {
	fmt.Println("Usage: odin <json|deploy|halt|fails> <release_file> (No args starts Lambda)")
	os.Exit(0)
}
