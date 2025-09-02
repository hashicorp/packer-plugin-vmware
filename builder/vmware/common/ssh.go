// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/sdk-internals/communicator/ssh"
	"golang.org/x/net/proxy"
)

// CommHost returns a function that determines the IP address of the guest that is ready to accept connections.
func CommHost(config *SSHConfig) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) {
		driver := state.Get("driver").(Driver)
		comm := config.Comm

		host := comm.Host()
		if host != "" {
			return host, nil
		}

		port := comm.Port()

		// Get the list of potential addresses that the guest might use.
		hosts, err := driver.PotentialGuestIP(state)
		if err != nil {
			return "", fmt.Errorf("failed to lookup IP address: %s", err)
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
				// Connect through a bastion host.
				connFunc = ssh.ProxyConnectFunc(pAddr, pAuth, "tcp", fmt.Sprintf("%s:%d", host, port))
			} else {
				// Connect directly to the host.
				connFunc = ssh.ConnectFunc("tcp", fmt.Sprintf("%s:%d", host, port))
			}
			conn, err := connFunc()

			// If we can connect, then we can use this IP address.
			if err == nil {
				err := conn.Close()
				if err != nil {
					return "", err
				}

				log.Printf("[INFO] IP address: %s", host)
				return host, nil
			}
		}

		return "", errors.New("connection not ready")
	}
}
