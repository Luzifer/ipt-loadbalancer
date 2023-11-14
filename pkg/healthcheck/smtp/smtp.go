// Package smtp contains a health-check to verify a SMTP server is
// indeed listening to SMTP protocol
package smtp

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/config"
	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/healthcheck/common"
	"github.com/Luzifer/go_helpers/v2/fieldcollection"
)

const (
	settingInsecureTLS = "insecureTLS"
	settingPort        = "port"
	settingTimeout     = "timeout"
	settingTLS         = "tls"
)

type (
	// Check represents the SMTP check
	Check struct{}
)

var (
	defInsecureTLS = false
	defTimeout     = time.Second
	defTLS         = false
)

// New returns a new SMTP check
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

	sc, err := smtp.NewClient(conn, "localhost")
	if err != nil {
		return fmt.Errorf("creating SMTP client: %w", err)
	}

	if settings.MustBool(settingTLS, &defTLS) {
		tc := &tls.Config{
			InsecureSkipVerify: settings.MustBool(settingInsecureTLS, &defInsecureTLS), //nolint:gosec // That's intended
		}

		if err = sc.StartTLS(tc); err != nil {
			return fmt.Errorf("starting TLS: %w", err)
		}
	}

	if err = sc.Hello("localhost"); err != nil {
		return fmt.Errorf("exchanging HELO: %w", err)
	}

	if err = sc.Close(); err != nil {
		return fmt.Errorf("closing connection: %w", err)
	}

	return nil
}

// Help returns the set of settings used in the check
func (Check) Help() (help []common.SettingHelp) {
	return []common.SettingHelp{
		{Name: settingInsecureTLS, Default: defInsecureTLS, Description: "Skip TLS certificate validation"},
		{Name: settingPort, Default: "target-port", Description: "Port to send the request to"},
		{Name: settingTimeout, Default: defTimeout, Description: "Timeout for the HTTP request"},
		{Name: settingTLS, Default: defTLS, Description: "Connect to port using TLS"},
	}
}

func (Check) intToInt64Ptr(i int) *int64 {
	i64 := int64(i)
	return &i64
}
