// Package healthcheck contains the interface checks have to implement
// and a registry to get them by name
package healthcheck

import (
	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/config"
	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/healthcheck/common"
	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/healthcheck/http"
	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/healthcheck/tcp"
	"github.com/Luzifer/go_helpers/v2/fieldcollection"
)

type (
	// Checker defines the interface a healthcheck must support
	Checker interface {
		Check(settings *fieldcollection.FieldCollection, target config.Target) error
		Help() []common.SettingHelp
	}
)

// ByName returns the Checker for the given name or nil if that name
// is not registered
func ByName(name string) Checker {
	switch name {
	case "http":
		return http.New()

	case "tcp":
		return tcp.New()

	default:
		return nil
	}
}
