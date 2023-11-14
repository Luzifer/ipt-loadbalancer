package main

import (
	"fmt"

	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/healthcheck"
	"github.com/Luzifer/go_helpers/v2/cli"
	"github.com/fatih/color"
	"github.com/rodaine/table"
)

func init() {
	registry.Add(cli.RegistryEntry{
		Description: "Display available settings for a check",
		Name:        "checkhelp",
		Params:      []string{"<checkType>"},
		Run: func(args []string) error {
			if len(args[1:]) < 1 {
				return fmt.Errorf("usage: checkhelp <checkType>")
			}

			check := healthcheck.ByName(args[1])
			if check == nil {
				return fmt.Errorf("check %q not found", args[1])
			}

			headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
			columnFmt := color.New(color.FgYellow).SprintfFunc()

			tbl := table.New("Setting", "Default", "Description")
			tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

			for _, help := range check.Help() {
				tbl.AddRow(help.Name, help.Default, help.Description)
			}

			tbl.Print()

			return nil
		},
	})
}
