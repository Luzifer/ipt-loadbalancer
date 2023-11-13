// Package tcp implements a simple TCP health-check
package tcp

import (
	"fmt"
	"net"
	"time"

	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/config"
	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/healthcheck/common"
	"github.com/Luzifer/go_helpers/v2/fieldcollection"
)

const (
	settingPort    = "port"
	settingTimeout = "timeout"
)

type (
	// Check represents the TCP check
	Check struct{}
)

var defTimeout = time.Second

// New returns a new TCP check
func New() Check { return Check{} }

// Check executes the check
func (c Check) Check(settings *fieldcollection.FieldCollection, target config.Target) error {
	conn, err := net.DialTimeout(
		"tcp",
		fmt.Sprintf("%s:%d", target.Addr, settings.MustInt64(settingPort, c.intToInt64Ptr(target.Port))),
		settings.MustDuration(settingTimeout, &defTimeout),
	)
	if err != nil {
		return fmt.Errorf("dialing tcp: %w", err)
	}

	if err = conn.Close(); err != nil {
		return fmt.Errorf("closing connection: %w", err)
	}

	return nil
}

// Help returns the set of settings used in the check
func (Check) Help() (help []common.SettingHelp) {
	return []common.SettingHelp{
		{Name: settingPort, Default: "target-port", Description: "Port to send the request to"},
		{Name: settingTimeout, Default: defTimeout, Description: "Timeout for the connect"},
	}
}

func (Check) intToInt64Ptr(i int) *int64 {
	i64 := int64(i)
	return &i64
}
