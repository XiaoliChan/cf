package cmd

import (
	"os"

	"github.com/teamssix/cf/pkg/util"

	cc "github.com/ivanpirog/coloredcobra"
	"github.com/spf13/cobra"
)

var logLevel string

var RootCmd = &cobra.Command{
	Use: "cf",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		util.Init(logLevel)
	},
}

func init() {
	RootCmd.PersistentFlags().StringVar(&logLevel, "logLevel", "info", "设置日志等级 (Set log level) [trace|debug|info|warn|error|fatal|panic]")
	RootCmd.CompletionOptions.DisableDefaultCmd = true
}

func Execute() {
	cc.Init(&cc.Config{
		RootCmd:  RootCmd,
		Headings: cc.HiGreen + cc.Underline,
		Commands: cc.Cyan + cc.Bold,
		Example:  cc.Italic,
		ExecName: cc.Bold,
		Flags:    cc.Cyan + cc.Bold,
	})
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
