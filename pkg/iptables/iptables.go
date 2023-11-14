// Package iptables contains the logic to interact with the iptables
// system interface
package iptables

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	coreosIptables "github.com/coreos/go-iptables/iptables"
	"github.com/mitchellh/hashstructure/v2"
)

const (
	natTable      = "nat"
	probBitsize   = 64
	probPrecision = 3
)

type (
	// Client contains the required functions to create the loadbalancing
	Client struct {
		*coreosIptables.IPTables

		managedChain string

		lock     sync.RWMutex
		services map[string][]NATTarget
	}

	// NATTarget contains the configuration for a DNAT jump target
	// with random distribution and given probability
	NATTarget struct {
		Addr      string
		BindAddr  string
		BindPort  int
		LocalAddr string
		Port      int
		Proto     string
		Weight    float64
	}

	// ServiceChain contains the name of the chain and a definition
	// which IP/Port combination should be sent to that chain
	ServiceChain struct {
		Name  string
		Addr  string
		Port  int
		Proto string
	}

	chainType uint
)

const (
	chainTypeDNAT chainType = iota
	chainTypeSNAT
)

var disallowedChars = regexp.MustCompile(`[^A-Z0-9_]`)

// New creates a new IPTables client
func New(managedChain string) (c *Client, err error) {
	c = &Client{
		managedChain: managedChain,

		services: make(map[string][]NATTarget),
	}
	if c.IPTables, err = coreosIptables.New(); err != nil {
		return nil, fmt.Errorf("creating iptables client: %w", err)
	}

	return c, nil
}

// EnsureManagedChains creates the managed chain referring to the
// service chains while only leading the specified address / port
// to that service chain
func (c *Client) EnsureManagedChains() (err error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var (
		dnat [][]string
		snat [][]string
	)

	for s := range c.services {
		for chain, cType := range map[string]chainType{
			c.tableName(c.managedChain, s, "DNAT"): chainTypeDNAT,
			c.tableName(c.managedChain, s, "SNAT"): chainTypeSNAT,
		} {
			if err = c.ensureChainWithRules(chain, c.buildServiceTable(s, cType)); err != nil {
				return fmt.Errorf("creating chain %q: %w", chain, err)
			}
		}

		dnat = append(dnat, []string{"-j", c.tableName(c.managedChain, s, "DNAT")})
		snat = append(snat, []string{"-j", c.tableName(c.managedChain, s, "SNAT")})
	}

	dnat = append(dnat, []string{"-j", "RETURN"})
	snat = append(snat, []string{"-j", "RETURN"})

	if err = c.ensureChainWithRules(c.tableName(c.managedChain, "DNAT"), dnat); err != nil {
		return fmt.Errorf("creating managed DNAT chain: %w", err)
	}

	if err = c.ensureChainWithRules(c.tableName(c.managedChain, "SNAT"), snat); err != nil {
		return fmt.Errorf("creating managed SNAT chain: %w", err)
	}

	return nil
}

// EnableMangedRoutingChains inserts a jump to the given managed chains
// at position 1 of the PREROUTING and POSTROUTING chains if it does
// not already exist in the chain
func (c *Client) EnableMangedRoutingChains() (err error) {
	if err = c.InsertUnique(natTable, "PREROUTING", 1, "-j", c.tableName(c.managedChain, "DNAT")); err != nil {
		return fmt.Errorf("ensuring DNAT jump to managed chain: %w", err)
	}

	if err = c.InsertUnique(natTable, "POSTROUTING", 1, "-j", c.tableName(c.managedChain, "SNAT")); err != nil {
		return fmt.Errorf("ensuring SNAT jump to managed chain: %w", err)
	}

	return nil
}

// RegisterServiceTarget adds a new routing target to the given service
func (c *Client) RegisterServiceTarget(service string, t NATTarget) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	var found bool
	for _, et := range c.services[service] {
		found = found || et.equals(t)
	}

	if !found {
		c.services[service] = append(c.services[service], t)
		return true
	}

	return false
}

// UnregisterServiceTarget removes a routing target from the given service
func (c *Client) UnregisterServiceTarget(service string, t NATTarget) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	var tmp []NATTarget
	for _, et := range c.services[service] {
		if !et.equals(t) {
			tmp = append(tmp, et)
		}
	}

	if len(tmp) == len(c.services[service]) {
		return false
	}

	c.services[service] = tmp
	return true
}

func (c *Client) buildServiceTable(service string, cType chainType) (rules [][]string) {
	weightLeft := 0.0
	for _, nt := range c.services[service] {
		weightLeft += nt.Weight
	}

	for _, nt := range c.services[service] {
		switch cType {
		case chainTypeDNAT:
			rules = append(rules, []string{
				"-m", "statistic",
				"--mode", "random",
				"--probability", strconv.FormatFloat(nt.Weight/weightLeft, 'f', probPrecision, probBitsize),

				"-p", nt.Proto,
				"-d", nt.BindAddr,
				"--dport", strconv.Itoa(nt.BindPort),

				"-j", "DNAT",
				"--to-destination", fmt.Sprintf("%s:%d", nt.Addr, nt.Port),
			})

		case chainTypeSNAT:
			rules = append(rules, []string{
				"-p", nt.Proto,
				"-d", nt.Addr,
				"--dport", strconv.Itoa(nt.Port),

				"-j", "SNAT",
				"--to-source", nt.LocalAddr,
			})
		}

		weightLeft -= nt.Weight
	}

	rules = append(rules, []string{"-j", "RETURN"})

	return rules
}

func (c *Client) ensureChainWithRules(chain string, rules [][]string) error {
	chainExists, err := c.ChainExists(natTable, chain)
	if err != nil {
		return fmt.Errorf("checking for chain existence: %w", err)
	}

	if chainExists {
		if err = c.ClearChain(natTable, chain); err != nil {
			return fmt.Errorf("clearing existing chain: %w", err)
		}
	} else {
		if err = c.NewChain(natTable, chain); err != nil {
			return fmt.Errorf("creating tmp-chain: %w", err)
		}
	}

	for _, rule := range rules {
		if err = c.Append(natTable, chain, rule...); err != nil {
			return fmt.Errorf("adding rule to chain: %w", err)
		}
	}

	return nil
}

func (*Client) tableName(components ...string) string {
	var parts []string
	for _, c := range components {
		parts = append(parts, disallowedChars.ReplaceAllString(strings.ToUpper(c), "_"))
	}

	return strings.Join(parts, "_")
}

func (n NATTarget) equals(c NATTarget) bool {
	nh, _ := hashstructure.Hash(n, hashstructure.FormatV2, nil)
	ch, _ := hashstructure.Hash(c, hashstructure.FormatV2, nil)

	return nh == ch
}
