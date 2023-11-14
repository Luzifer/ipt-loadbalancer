package main

import (
	"os"

	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/config"
	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/iptables"
	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/servicemonitor"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/Luzifer/rconfig/v2"
)

var (
	cfg = struct {
		Config             string `flag:"config,c" default:"config.yaml" description:"Configuration file to load"`
		EnableManagedChain bool   `flag:"enable-managed-chain,e" default:"false" description:"Modify PREROUTING / POSTROUTING chain to contain a jump to managed chain"`
		LogLevel           string `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		VersionAndExit     bool   `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	version = "dev"
)

func initApp() error {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		return errors.Wrap(err, "parsing cli options")
	}

	l, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return errors.Wrap(err, "parsing log-level")
	}
	logrus.SetLevel(l)

	return nil
}

func main() {
	var err error
	if err = initApp(); err != nil {
		logrus.WithError(err).Fatal("initializing app")
	}

	if cfg.VersionAndExit {
		logrus.WithField("version", version).Info("ipt-loadbalancer")
		os.Exit(0)
	}

	confFile, err := config.Load(cfg.Config)
	if err != nil {
		logrus.WithError(err).Fatal("loading config file")
	}

	ipt, err := iptables.New(confFile.ManagedChain)
	if err != nil {
		logrus.WithError(err).Fatal("creating iptables client")
	}

	if err = ipt.EnsureManagedChains(); err != nil {
		logrus.WithError(err).Fatal("creating managed chain")
	}

	if cfg.EnableManagedChain {
		if err = ipt.EnableMangedRoutingChains(); err != nil {
			logrus.WithError(err).Fatal("enabling routing")
		}
	}

	svcErr := make(chan error, 1)
	for i := range confFile.Services {
		s := confFile.Services[i]

		sMon := servicemonitor.New(ipt, logrus.WithField("service", s.Name), s)
		go func() { svcErr <- sMon.Run() }()
	}

	logrus.WithFields(logrus.Fields{
		"services": len(confFile.Services),
		"version":  version,
	}).Info("ipt-loadbalancer started")

	for err := range svcErr {
		if err == nil {
			continue
		}

		logrus.WithError(err).Fatal("service monitor caused error")
	}
}
