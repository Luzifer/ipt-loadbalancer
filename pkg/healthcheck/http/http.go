// Package http contains a http health-check
package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/config"
	"git.luzifer.io/luzifer/ipt-loadbalancer/pkg/healthcheck/common"
	"github.com/Luzifer/go_helpers/v2/fieldcollection"
)

const (
	settingCode          = "code"
	settingExpectContent = "expectContent"
	settingHost          = "host"
	settingInsecureTLS   = "insecureTLS"
	settingMethod        = "method"
	settingPath          = "path"
	settingPort          = "port"
	settingTimeout       = "timeout"
	settingTLS           = "tls"
)

type (
	// Check represents the HTTP check
	Check struct{}
)

var (
	defCode          = http.StatusOK
	defExpectContent = ""
	defHost          = ""
	defInsecureTLS   = false
	defMethod        = http.MethodGet
	defPath          = "/"
	defTimeout       = time.Second
	defTLS           = false
)

// New returns a new HTTP check
func New() Check { return Check{} }

// Check executes the check
func (c Check) Check(settings *fieldcollection.FieldCollection, target config.Target) error {
	ctx, cancel := context.WithTimeout(context.Background(), settings.MustDuration(settingTimeout, &defTimeout))
	defer cancel()

	u := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", target.Addr, settings.MustInt64(settingPort, c.intToInt64Ptr(target.Port))),
		Path:   settings.MustString(settingPath, &defPath),
	}

	if settings.MustBool(settingTLS, &defTLS) {
		u.Scheme = "https"
	}

	req, err := http.NewRequestWithContext(ctx, settings.MustString(settingMethod, &defMethod), u.String(), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "ipt-loadbalancer/v1 (https://git.luzifer.io/luzifer/ipt-loadbalancer)")

	if hh := settings.MustString(settingHost, &defHost); hh != defHost {
		req.Header.Set("Host", hh)
	}

	client := http.Client{}
	if settings.MustBool(settingInsecureTLS, &defInsecureTLS) {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // The intention is to use insecure TLS
			},
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != int(settings.MustInt64(settingCode, c.intToInt64Ptr(defCode))) {
		return fmt.Errorf("unexpected status code %d != %d", resp.StatusCode, settings.MustInt64(settingCode, c.intToInt64Ptr(defCode)))
	}

	if settings.MustString(settingExpectContent, &defExpectContent) == defExpectContent {
		return nil
	}

	content, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if !strings.Contains(string(content), settings.MustString(settingExpectContent, &defExpectContent)) {
		return fmt.Errorf("expected content not found in body")
	}

	return nil
}

// Help returns the set of settings used in the check
func (Check) Help() (help []common.SettingHelp) {
	return []common.SettingHelp{
		{Name: settingCode, Default: defCode, Description: "HTTP Status-Code to expect from the request"},
		{Name: settingExpectContent, Default: defExpectContent, Description: "Content to search in the response body"},
		{Name: settingHost, Default: defHost, Description: "Host header to send with the request"},
		{Name: settingInsecureTLS, Default: defInsecureTLS, Description: "Skip TLS certificate validation"},
		{Name: settingMethod, Default: defMethod, Description: "Method to use for request"},
		{Name: settingPath, Default: defPath, Description: "Path to send the request to"},
		{Name: settingPort, Default: "target-port", Description: "Port to send the request to"},
		{Name: settingTimeout, Default: defTimeout, Description: "Timeout for the HTTP request"},
		{Name: settingTLS, Default: defTLS, Description: "Connect to port using TLS"},
	}
}

func (Check) intToInt64Ptr(i int) *int64 {
	i64 := int64(i)
	return &i64
}
