package config

import (
	"strings"
	"testing"
)

func TestConnectionProfileValidate(t *testing.T) {
	tests := []struct {
		name    string
		profile ConnectionProfile
		wantErr string
	}{
		{
			name: "test CMS",
			profile: ConnectionProfile{
				ID:        "cms-test",
				Name:      "Winlink Test CMS",
				Network:   NetworkInternet,
				Transport: TransportCMSTelnet,
				CMS: &CMSProfile{
					Mode: CMSModeTest,
				},
			},
		},
		{
			name: "production CMS",
			profile: ConnectionProfile{
				ID:        "cms-production",
				Name:      "Winlink Production CMS",
				Network:   NetworkInternet,
				Transport: TransportCMSTelnet,
				CMS: &CMSProfile{
					Mode: CMSModeProduction,
				},
			},
		},
		{
			name: "Internet direct TCP",
			profile: ConnectionProfile{
				ID:        "internet-peer",
				Name:      "Internet Peer",
				Network:   NetworkInternet,
				Transport: TransportDirectTCP,
				TCP: &TCPProfile{
					Address:    "example.net:8772",
					TargetCall: "W2ABC",
				},
			},
		},
		{
			name: "LAN direct TCP",
			profile: ConnectionProfile{
				ID:        "lan-peer",
				Name:      "LAN Peer",
				Network:   NetworkLAN,
				Transport: TransportDirectTCP,
				TCP: &TCPProfile{
					Address:    "192.168.50.20:8772",
					TargetCall: "W2ABC",
				},
			},
		},
		{
			name: "AREDN direct TCP",
			profile: ConnectionProfile{
				ID:        "mesh-peer",
				Name:      "Mesh Peer",
				Network:   NetworkAREDN,
				Transport: TransportDirectTCP,
				TCP: &TCPProfile{
					Address:    "w2abc-node.local.mesh:8772",
					TargetCall: "W2ABC",
				},
			},
		},
		{
			name: "direct TCP mesh IP",
			profile: ConnectionProfile{
				ID:        "mesh-ip-peer",
				Name:      "Mesh IP Peer",
				Network:   NetworkAREDN,
				Transport: TransportDirectTCP,
				TCP: &TCPProfile{
					Address:    "10.42.1.15:8772",
					TargetCall: "W2ABC",
				},
			},
		},
		{
			name: "missing ID",
			profile: ConnectionProfile{
				Name:      "Peer",
				Network:   NetworkAREDN,
				Transport: TransportDirectTCP,
				TCP: &TCPProfile{
					Address:    "node.local.mesh:8772",
					TargetCall: "W2ABC",
				},
			},
			wantErr: "profile ID is required",
		},
		{
			name: "missing name",
			profile: ConnectionProfile{
				ID:        "peer",
				Network:   NetworkAREDN,
				Transport: TransportDirectTCP,
				TCP: &TCPProfile{
					Address:    "node.local.mesh:8772",
					TargetCall: "W2ABC",
				},
			},
			wantErr: "profile name is required",
		},
		{
			name: "unsupported network",
			profile: ConnectionProfile{
				ID:        "peer",
				Name:      "Peer",
				Network:   NetworkType("unknown"),
				Transport: TransportDirectTCP,
				TCP: &TCPProfile{
					Address:    "example.net:8772",
					TargetCall: "W2ABC",
				},
			},
			wantErr: "unsupported network",
		},
		{
			name: "CMS on AREDN",
			profile: ConnectionProfile{
				ID:        "cms",
				Name:      "CMS",
				Network:   NetworkAREDN,
				Transport: TransportCMSTelnet,
				CMS: &CMSProfile{
					Mode: CMSModeTest,
				},
			},
			wantErr: "CMS Telnet requires the Internet network",
		},
		{
			name: "CMS missing settings",
			profile: ConnectionProfile{
				ID:        "cms",
				Name:      "CMS",
				Network:   NetworkInternet,
				Transport: TransportCMSTelnet,
			},
			wantErr: "CMS settings are required",
		},
		{
			name: "CMS with TCP settings",
			profile: ConnectionProfile{
				ID:        "cms",
				Name:      "CMS",
				Network:   NetworkInternet,
				Transport: TransportCMSTelnet,
				CMS: &CMSProfile{
					Mode: CMSModeTest,
				},
				TCP: &TCPProfile{
					Address:    "example.net:8772",
					TargetCall: "W2ABC",
				},
			},
			wantErr: "must not contain TCP settings",
		},
		{
			name: "unsupported CMS mode",
			profile: ConnectionProfile{
				ID:        "cms",
				Name:      "CMS",
				Network:   NetworkInternet,
				Transport: TransportCMSTelnet,
				CMS: &CMSProfile{
					Mode: CMSMode("other"),
				},
			},
			wantErr: "unsupported CMS mode",
		},
		{
			name: "direct TCP on radio network",
			profile: ConnectionProfile{
				ID:        "radio-peer",
				Name:      "Radio Peer",
				Network:   NetworkRadio,
				Transport: TransportDirectTCP,
				TCP: &TCPProfile{
					Address:    "localhost:8772",
					TargetCall: "W2ABC",
				},
			},
			wantErr: "direct TCP does not support network",
		},
		{
			name: "direct TCP missing settings",
			profile: ConnectionProfile{
				ID:        "peer",
				Name:      "Peer",
				Network:   NetworkAREDN,
				Transport: TransportDirectTCP,
			},
			wantErr: "TCP settings are required",
		},
		{
			name: "direct TCP with CMS settings",
			profile: ConnectionProfile{
				ID:        "peer",
				Name:      "Peer",
				Network:   NetworkAREDN,
				Transport: TransportDirectTCP,
				CMS: &CMSProfile{
					Mode: CMSModeTest,
				},
				TCP: &TCPProfile{
					Address:    "node.local.mesh:8772",
					TargetCall: "W2ABC",
				},
			},
			wantErr: "must not contain CMS settings",
		},
		{
			name: "direct TCP missing port",
			profile: ConnectionProfile{
				ID:        "peer",
				Name:      "Peer",
				Network:   NetworkAREDN,
				Transport: TransportDirectTCP,
				TCP: &TCPProfile{
					Address:    "node.local.mesh",
					TargetCall: "W2ABC",
				},
			},
			wantErr: "TCP address must be host:port",
		},
		{
			name: "direct TCP invalid port",
			profile: ConnectionProfile{
				ID:        "peer",
				Name:      "Peer",
				Network:   NetworkAREDN,
				Transport: TransportDirectTCP,
				TCP: &TCPProfile{
					Address:    "node.local.mesh:70000",
					TargetCall: "W2ABC",
				},
			},
			wantErr: "TCP port must be between 1 and 65535",
		},
		{
			name: "direct TCP missing target",
			profile: ConnectionProfile{
				ID:        "peer",
				Name:      "Peer",
				Network:   NetworkAREDN,
				Transport: TransportDirectTCP,
				TCP: &TCPProfile{
					Address: "node.local.mesh:8772",
				},
			},
			wantErr: "target callsign is required",
		},
		{
			name: "unsupported transport",
			profile: ConnectionProfile{
				ID:        "peer",
				Name:      "Peer",
				Network:   NetworkInternet,
				Transport: TransportType("unknown"),
			},
			wantErr: "unsupported transport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf(
					"Validate() expected error containing %q",
					tt.wantErr,
				)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf(
					"Validate() error = %q, want substring %q",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestConnectionProfileNormalized(t *testing.T) {
	input := ConnectionProfile{
		ID:        "  mesh-peer  ",
		Name:      "  Mesh Peer  ",
		Network:   NetworkType(" AREDN "),
		Transport: TransportType(" DIRECT_TCP "),
		TCP: &TCPProfile{
			Address:    "  node.local.mesh:8772  ",
			TargetCall: " w2abc ",
		},
	}

	got := input.Normalized()

	if got.ID != "mesh-peer" {
		t.Fatalf("ID = %q, want mesh-peer", got.ID)
	}
	if got.Name != "Mesh Peer" {
		t.Fatalf("Name = %q, want Mesh Peer", got.Name)
	}
	if got.Network != NetworkAREDN {
		t.Fatalf("Network = %q, want %q", got.Network, NetworkAREDN)
	}
	if got.Transport != TransportDirectTCP {
		t.Fatalf(
			"Transport = %q, want %q",
			got.Transport,
			TransportDirectTCP,
		)
	}
	if got.TCP.Address != "node.local.mesh:8772" {
		t.Fatalf(
			"Address = %q, want node.local.mesh:8772",
			got.TCP.Address,
		)
	}
	if got.TCP.TargetCall != "W2ABC" {
		t.Fatalf("TargetCall = %q, want W2ABC", got.TCP.TargetCall)
	}

	if input.TCP.TargetCall != " w2abc " {
		t.Fatal("Normalized() mutated the source TCP profile")
	}
}
