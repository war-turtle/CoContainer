package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/war-turtle/CoContainer/cocontainer"
)

var vethPrefix string

func init() {
	RunCmd.Flags().StringVarP(&vethPrefix, "vEthPrefix", "v", "vm", "pass prefix for the virtual ethernet name")
}

//RunCmd is run command for cocontainer
var RunCmd = &cobra.Command{
	Use: "run",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(args)
		fmt.Println(vethPrefix)
		if len(args) == 0 {
			args = append(args, "/bin/sh")
		}
		cocontainer.Run(args, vethPrefix)
	},
}
