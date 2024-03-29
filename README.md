This repository contains a manager which health-checks given endpoints and generates a iptables DNAT/SNAT loadbalancer from active endpoints.

## Configuration

```console
# ipt-loadbalancer --help
Usage of ipt-loadbalancer:
  -c, --config string          Configuration file to load (default "config.yaml")
  -e, --enable-managed-chain   Modify PREROUTING / POSTROUTING chain to contain a jump to managed chain
      --log-level string       Log level (debug, info, warn, error, fatal) (default "info")
      --version                Prints current version and exits

# ipt-loadbalancer help
Supported sub-commands are:
  checkhelp <checkType>  Display available settings for a check

# ipt-loadbalancer checkhelp http
Setting        Default      Description
code           200          HTTP Status-Code to expect from the request
...
```

### Main Configuration File

```yaml
---

# Table prefix to manage (should not collide with existing tables in
# the system). Created tables in this case are named IPTLB_DNAT,
# IPTLB_SNAT and IPTLB_SERVICENAME_DNAT / IPTLB_SERVICENAME_SNAT.
managedChain: IPTLB

# Collection of services to expose on the host the ipt-loadbalancer
# runs on. Each service exposes one local port and forwards to N
# remote ports using DNAT/SNAT.
services:
  - name: ... # See below for the service definition

...
```

### Service Configuration

```yaml
# The name should consist only of [a-z0-9_] character set and must be
# unique for the prefix given in the main configuration file
name: https

# The healthcheck defines how to verify the targets are up to include
# them into the loadbalancing
healthCheck:
  # Type is required, currently supported: http, smtp, tcp
  type: http
  # Interval defines how often to check for the targets to be alive:
  # 2s means from the start of the LB the targets are checked every 2s
  # in parallel
  interval: 2s
  # Settings defines parameters for the given health-check and are
  # individual to the type. See the `checkhelp <type>` subcommand for
  # all supported settings and their default values.
  settings:
    insecureTLS: true
    path: /healthz
    tls: true

# Bind Address and Port describes the IP and Port to bind the service
# to.
bindAddr: 203.0.113.1
bindPort: 443

# Proto describes which protocol should be routed (defaults to tcp)
proto: tcp

# Targets is a list of routing targets which are checked for their
# liveness status and if they are live, they are included in the NAT
# rulesets.
# Each target consists of two Addresses, a Port and a Weight. The weight
# is defined as an integer and increases / decreases the percentage
# of the traffic that specific target will receive. (For example
# setting all weights to 1 will distribute the traffic equally between
# them, setting one to 2 will double the traffic to that target.)
# The localAddr is used for the SNAT to map the source IP.
targets:
  - addr: 10.1.2.4
    localAddr: 10.1.2.1
    port: 443
    weight: 1
  - addr: 10.1.2.5
    localAddr: 10.1.2.1
    port: 443
    weight: 1
  - addr: 10.1.2.6
    localAddr: 10.1.2.1
    port: 443
    weight: 1
```
