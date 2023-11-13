// Package common contains some helpers used in multiple checks
package common

type (
	// SettingHelp is used to render a help for check config
	SettingHelp struct {
		Name        string
		Default     any
		Description string
	}
)
