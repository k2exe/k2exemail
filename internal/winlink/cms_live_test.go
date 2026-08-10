//go:build integration

package winlink

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/la5nta/wl2k-go/transport/telnet"
)

func TestLiveCMSTelnetLogin(t *testing.T) {
	callsign := strings.ToUpper(
		strings.TrimSpace(
			os.Getenv("K2EXEMAIL_LIVE_CALLSIGN"),
		),
	)
	if callsign == "" {
		t.Skip(
			"K2EXEMAIL_LIVE_CALLSIGN is required",
		)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	conn, err := dialCMSTelnet(
		ctx,
		telnet.CMSAddress,
		callsign,
		telnet.CMSPassword,
	)
	if err != nil {
		t.Fatalf(
			"live CMS telnet login failed: %v",
			err,
		)
	}
	defer conn.Close()

	t.Logf(
		"CMS telnet login succeeded for %s via %s",
		callsign,
		conn.RemoteAddr(),
	)
}
