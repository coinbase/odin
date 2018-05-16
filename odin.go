package main

import (
	"fmt"
	"os"

	"github.com/coinbase/odin/deployer"
	"github.com/coinbase/odin/deployer/client"
	"github.com/coinbase/step/utils/run"
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

	switch command {
	case "json":
		run.JSON(deployer.StateMachine())
	case "exec":
		run.Exec(deployer.StateMachineWithTaskHandlers(deployer.TaskFunctions()))(&arg)
	case "deploy":
		// Send Configuration to the deployer
		// arg is a filename OR a JSON string
		err := client.Deploy(&arg)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	case "halt":
		err := client.Halt(&arg)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	default:
		printUsage() // Print how to use and exit
	}
}

func printUsage() {
	fmt.Println("Usage: odin <json|exec|deploy> <arg> (No args starts Lambda)")
	os.Exit(0)
}
