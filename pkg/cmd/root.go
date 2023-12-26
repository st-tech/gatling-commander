/*
Copyright &copy; ZOZO, Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the “Software”), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included
in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// Package cmd implements root command of gatling-commander.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/st-tech/gatling-commander/pkg/cmd/exec"
	cfg "github.com/st-tech/gatling-commander/pkg/config"
)

type gatlingCommanderOptions struct {
	Arguments []string
}

var configFile string
var config cfg.Config

const rootCmdName = "gatling-commander"

// NewDefaultGatlingCommanderCommand creates the 'gatling-commander' command with default arguments.
func NewDefaultGatlingCommanderCommand() *cobra.Command {
	return NewDefaultGatlingCommanderCommandWithArgs(gatlingCommanderOptions{
		Arguments: os.Args,
	})
}

// NewDefaultGatlingCommanderCommandWithArgs creates the 'gatling-commander' command with arguments.
func NewDefaultGatlingCommanderCommandWithArgs(o gatlingCommanderOptions) *cobra.Command {
	cmd := NewGatlingCommanderCommand(o)

	if len(o.Arguments) > 1 {
		cmdPathPieces := o.Arguments[1:]
		var cmdName string
		if _, _, err := cmd.Find(cmdPathPieces); err != nil {
			for _, arg := range cmdPathPieces {
				if !strings.HasPrefix(arg, "-") {
					cmdName = arg
					break
				}
			}
			switch cmdName {
			case "help":
				// Avoid unsupported command error.
				// The help command display default help message generated by cobra.
			default:
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
	}

	return cmd
}

// NewGatlingCommanderCommand creates the 'gatling-commander' command and its nested children.
func NewGatlingCommanderCommand(o gatlingCommanderOptions) *cobra.Command {
	// Parent Command to which all subcommands are added.
	cmds := &cobra.Command{
		Use:   rootCmdName,
		Short: "gatling-commander automates the execution of load test using Gatling Operator",
		Long: `gatling-commander is a CLI tool that automates a series of tasks
				in the execution of load test using Gatling Operator.
				Complete documentation is available at https://github.com/st-tech/gatling-commander/docs`,
	}

	cmds.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file name")

	cobra.OnInitialize(func() {
		// avoid to return error when run help command without config flag
		if configFile == "" {
			if len(o.Arguments) > 1 {
				cmdName := o.Arguments[1]
				if cmdName == "help" {
					_ = cmds.Help()
					os.Exit(0)
				}
			}
			fmt.Fprintf(os.Stderr, "Error: config file not provided\n")
			os.Exit(1)
		}

		viper.SetConfigFile(configFile)

		if err := viper.ReadInConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := viper.Unmarshal(&config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := config.ValidateFieldValue(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid config param %v\n", err)
			os.Exit(1)
		}
	})
	cmds.CompletionOptions.DisableDefaultCmd = true
	cmds.AddCommand(exec.NewCmdExec(rootCmdName, &config))
	return cmds
}