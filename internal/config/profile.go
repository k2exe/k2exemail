package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func (p ConnectionProfile) Normalized() ConnectionProfile {
	normalized := p

	if p.CMS != nil {
		cms := *p.CMS
		normalized.CMS = &cms
	}

	if p.TCP != nil {
		tcp := *p.TCP
		normalized.TCP = &tcp
	}

	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.Name = strings.TrimSpace(normalized.Name)
	normalized.Network = NetworkType(
		strings.ToLower(strings.TrimSpace(string(normalized.Network))),
	)
	normalized.Transport = TransportType(
		strings.ToLower(strings.TrimSpace(string(normalized.Transport))),
	)

	if normalized.CMS != nil {
		normalized.CMS.Mode = CMSMode(
			strings.ToLower(
				strings.TrimSpace(string(normalized.CMS.Mode)),
			),
		)
	}

	if normalized.TCP != nil {
		normalized.TCP.Address = strings.TrimSpace(
			normalized.TCP.Address,
		)
		normalized.TCP.TargetCall = strings.ToUpper(
			strings.TrimSpace(normalized.TCP.TargetCall),
		)
	}

	return normalized
}

func (p ConnectionProfile) Validate() error {
	p = p.Normalized()

	if p.ID == "" {
		return fmt.Errorf("profile ID is required")
	}
	if p.Name == "" {
		return fmt.Errorf("profile name is required")
	}

	switch p.Network {
	case NetworkInternet, NetworkLAN, NetworkAREDN, NetworkRadio:
	default:
		return fmt.Errorf("unsupported network %q", p.Network)
	}

	switch p.Transport {
	case TransportCMSTelnet:
		if p.Network != NetworkInternet {
			return fmt.Errorf(
				"CMS Telnet requires the Internet network",
			)
		}
		if p.CMS == nil {
			return fmt.Errorf("CMS settings are required")
		}
		if p.TCP != nil {
			return fmt.Errorf(
				"CMS Telnet profile must not contain TCP settings",
			)
		}

		switch p.CMS.Mode {
		case CMSModeTest, CMSModeProduction:
		default:
			return fmt.Errorf(
				"unsupported CMS mode %q",
				p.CMS.Mode,
			)
		}

	case TransportDirectTCP:
		if p.Network != NetworkInternet &&
			p.Network != NetworkLAN &&
			p.Network != NetworkAREDN {
			return fmt.Errorf(
				"direct TCP does not support network %q",
				p.Network,
			)
		}
		if p.TCP == nil {
			return fmt.Errorf("TCP settings are required")
		}
		if p.CMS != nil {
			return fmt.Errorf(
				"direct TCP profile must not contain CMS settings",
			)
		}
		if err := validateTCPAddress(p.TCP.Address); err != nil {
			return err
		}
		if p.TCP.TargetCall == "" {
			return fmt.Errorf("target callsign is required")
		}

	default:
		return fmt.Errorf(
			"unsupported transport %q",
			p.Transport,
		)
	}

	return nil
}

func validateTCPAddress(address string) error {
	address = strings.TrimSpace(address)
	if address == "" {
		return fmt.Errorf("TCP address is required")
	}

	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf(
			"TCP address must be host:port: %w",
			err,
		)
	}

	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("TCP host is required")
	}

	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf(
			"TCP port must be between 1 and 65535",
		)
	}

	return nil
}

func (c Config) Validate() error {
	c = c.Normalized()

	ids := make(map[string]struct{}, len(c.ConnectionProfiles))

	for i, profile := range c.ConnectionProfiles {
		profile = profile.Normalized()

		if err := profile.Validate(); err != nil {
			return fmt.Errorf(
				"connection profile %d (%q): %w",
				i+1,
				profile.ID,
				err,
			)
		}

		if _, exists := ids[profile.ID]; exists {
			return fmt.Errorf(
				"duplicate connection profile ID %q",
				profile.ID,
			)
		}

		ids[profile.ID] = struct{}{}
	}

	return nil
}
