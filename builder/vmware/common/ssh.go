// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/sdk-internals/communicator/ssh"
	"golang.org/x/net/proxy"
)

// CommHost returns a function that determines the IP address of the guest that
// is ready to accept connections.
func CommHost(config *SSHConfig) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) {
		driver := state.Get("driver").(Driver)
		comm := config.Comm

		host := comm.Host()
		if host != "" {
			return host, nil
		}

		port := comm.Port()

		// Check if this is a bridged network (case-insensitive).
		network := state.Get("vmnetwork").(string)
		isBridged := strings.EqualFold(network, "bridged")

		var hosts []string
		var err error

		if isBridged {
			// For bridged networks, wait for VMware Tools to provide the IP address.
			if state.Get("vmtools_ip_attempt") == nil {
				log.Printf("[INFO] Waiting for guest IP address from VMware Tools...")
				state.Put("vmtools_ip_attempt", true)
			}

			vmxPath := state.Get("vmx_path").(string)
			if addr, vmrunErr := driver.GetGuestIPAddress(vmxPath); vmrunErr == nil && addr != "" {
				hosts = []string{addr}
			} else {
				return "", fmt.Errorf("waiting for VMware Tools to start: %s", vmrunErr)
			}
		} else {
			// For NAT/host-only networks, use DHCP leases as the primary method.
			hosts, err = driver.PotentialGuestIP(state)
			if err != nil {
				// Fallback: Check to see if VMware Tools can provide the IP address.
				vmxPath := state.Get("vmx_path").(string)
				if addr, vmrunErr := driver.GetGuestIPAddress(vmxPath); vmrunErr == nil && addr != "" {
					hosts = []string{addr}
				} else {
					return "", fmt.Errorf("failed to lookup guest IP address: %s", err)
				}
			}
		}

		if len(hosts) == 0 {
			return "", errors.New("connection not ready, no IP yet")
		}

		var pAddr string
		var pAuth *proxy.Auth
		if config.Comm.SSHProxyHost != "" {
			pAddr = fmt.Sprintf("%s:%d", config.Comm.SSHProxyHost, config.Comm.SSHProxyPort)
			if config.Comm.SSHProxyUsername != "" {
				pAuth = new(proxy.Auth)
				pAuth.User = config.Comm.SSHProxyUsername
				pAuth.Password = config.Comm.SSHProxyPassword
			}
		}

		// Test connectivity to each potential IP address to determine which one
		// is actively running the SSH/WinRM service on the expected port.
		var connFunc func() (net.Conn, error)
		for _, host := range hosts {
			if pAddr != "" {
				// Connect using a bastion host.
				connFunc = ssh.ProxyConnectFunc(pAddr, pAuth, "tcp", fmt.Sprintf("%s:%d", host, port))
			} else {
				// Connect directly.
				connFunc = ssh.ConnectFunc("tcp", fmt.Sprintf("%s:%d", host, port))
			}
			conn, err := connFunc()

			// If the connection is successful, use this IP address.
			if err == nil {
				err := conn.Close()
				if err != nil {
					return "", err
				}

				log.Printf("[INFO] Guest Operating System IP address: %s", host)
				return host, nil
			}
		}

		return "", errors.New("connection not ready")
	}
}
