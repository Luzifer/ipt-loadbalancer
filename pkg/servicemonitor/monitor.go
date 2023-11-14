// Package servicemonitor contains the monitoring logic which then
// triggers a rebuild of the chain in case there is a change
package servicemonitor

import (
	"fmt"
	"sync"
	"time"

	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/config"
	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/healthcheck"
	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/iptables"
	"github.com/sirupsen/logrus"
)

type (
	// Monitor contains the monitoring logic and state
	Monitor struct {
		ipt    *iptables.Client
		logger *logrus.Entry
		svc    config.Service
	}
)

// New creates a new monitor with empty rule set
func New(ipt *iptables.Client, logger *logrus.Entry, svc config.Service) *Monitor {
	return &Monitor{
		ipt:    ipt,
		logger: logger,
		svc:    svc,
	}
}

// Run contains the monitoring loop for the given service and should
// run in the background. When returning an error the loop is stopped.
func (m Monitor) Run() (err error) {
	for {
		itStart := time.Now()

		checker := healthcheck.ByName(m.svc.HealthCheck.Type)
		if checker == nil {
			return fmt.Errorf("checker %q not found", m.svc.HealthCheck.Type)
		}

		if err = m.updateRoutingTargets(checker); err != nil {
			return fmt.Errorf("updating healthy targets: %w", err)
		}

		time.Sleep(m.svc.HealthCheck.Interval - time.Since(itStart))
	}
}

func (m Monitor) updateRoutingTargets(checker healthcheck.Checker) (err error) {
	var (
		down, up []string

		changed bool
		wg      sync.WaitGroup
	)
	wg.Add(len(m.svc.Targets))

	for i := range m.svc.Targets {
		t := m.svc.Targets[i]
		logger := m.logger.WithField("target", fmt.Sprintf("%s:%d", t.Addr, t.Port))
		go func() {
			defer wg.Done()

			tgt := iptables.NATTarget{
				Addr:      t.Addr,
				BindAddr:  m.svc.BindAddr,
				BindPort:  m.svc.BindPort,
				LocalAddr: t.LocalAddr,
				Port:      t.Port,
				Weight:    float64(t.Weight),
				Proto:     m.svc.Protocol(),
			}

			if err := checker.Check(m.svc.HealthCheck.Settings, t); err != nil {
				logger.WithError(err).Debug("detected target down")
				changed = changed || m.ipt.UnregisterServiceTarget(m.svc.Name, tgt)
				down = append(down, t.String())
				return
			}

			logger.Debug("target up")
			changed = changed || m.ipt.RegisterServiceTarget(m.svc.Name, tgt)
			up = append(up, t.String())
		}()
	}

	wg.Wait()

	uplog := m.logger.WithFields(logrus.Fields{
		"down": down,
		"up":   up,
	})

	switch {
	case len(up) == len(up)+len(down):
		uplog.Debugf("%d/%d targets up", len(up), len(up)+len(down))
	case len(up) > 0:
		uplog.Warnf("%d/%d targets up", len(up), len(up)+len(down))
	case len(up) == 0:
		uplog.Errorf("%d/%d targets up", len(up), len(up)+len(down))
	}

	if !changed {
		return nil
	}

	if err = m.ipt.EnsureManagedChains(); err != nil {
		return fmt.Errorf("updating chains: %w", err)
	}

	return nil
}
