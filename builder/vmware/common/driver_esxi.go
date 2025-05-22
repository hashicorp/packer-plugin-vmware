// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/hashicorp/go-getter/v2"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	helperssh "github.com/hashicorp/packer-plugin-sdk/communicator/ssh"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/sdk-internals/communicator/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// EsxiDriver is a driver for building virtual machines on an ESXi host.
type EsxiDriver struct {
	base VmwareDriver

	Host           string
	Port           int
	Username       string
	Password       string
	PrivateKeyFile string
	Datastore      string
	CacheDatastore string
	CacheDirectory string
	VMName         string
	CommConfig     communicator.Config

	ctx    context.Context
	client *govmomi.Client
	finder *find.Finder

	comm      packersdk.Communicator
	outputDir string
	vmId      string
}

func NewEsxiDriver(dconfig *DriverConfig, config *SSHConfig, vmName string) (Driver, error) {
	ctx := context.TODO()

	vsphereUrl, err := url.Parse(fmt.Sprintf("https://%v/sdk", dconfig.RemoteHost))
	if err != nil {
		return nil, err
	}
	credentials := url.UserPassword(dconfig.RemoteUser, dconfig.RemotePassword)
	vsphereUrl.User = credentials

	soapClient := soap.NewClient(vsphereUrl, true)
	vimClient, err := vim25.NewClient(ctx, soapClient)
	if err != nil {
		return nil, err
	}

	vimClient.RoundTripper = session.KeepAlive(vimClient.RoundTripper, 10*time.Minute)
	client := &govmomi.Client{
		Client:         vimClient,
		SessionManager: session.NewManager(vimClient),
	}

	err = client.SessionManager.Login(ctx, credentials)
	if err != nil {
		return nil, err
	}

	finder := find.NewFinder(client.Client, false)
	datacenter, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		return nil, err
	}
	finder.SetDatacenter(datacenter)

	return &EsxiDriver{
		Host:           dconfig.RemoteHost,
		Port:           dconfig.RemotePort,
		Username:       dconfig.RemoteUser,
		Password:       dconfig.RemotePassword,
		PrivateKeyFile: dconfig.RemotePrivateKey,
		Datastore:      dconfig.RemoteDatastore,
		CacheDatastore: dconfig.RemoteCacheDatastore,
		CacheDirectory: dconfig.RemoteCacheDirectory,
		VMName:         vmName,
		CommConfig:     config.Comm,
		ctx:            ctx,
		client:         client,
		finder:         finder,
	}, nil
}

func (d *EsxiDriver) Clone(dst, src string, linked bool, snapshot string) error {

	linesToArray := func(lines string) []string { return strings.Split(strings.Trim(lines, "\r\n"), "\n") }

	d.SetOutputDir(path.Dir(filepath.ToSlash(dst)))
	srcVmx := d.datastorePath(src)
	dstVmx := d.datastorePath(dst)
	srcDir := path.Dir(srcVmx)
	dstDir := path.Dir(dstVmx)

	log.Printf("Source: %s\n", srcVmx)
	log.Printf("Destination: %s\n", dstVmx)

	err := d.MkdirAll()
	if err != nil {
		return fmt.Errorf("ferror creatig the destination directory %s: %s", d.outputDir, err)
	}

	err = d.sh("cp", strconv.Quote(srcVmx), strconv.Quote(dstVmx))
	if err != nil {
		return fmt.Errorf("error copying the .vmx file %s: %s", srcVmx, err)
	}

	filesToClone, err := d.run(nil, "find", strconv.Quote(srcDir), "! -name '*.vmdk' ! -name '*.vmx' ! -name '*.vmxf' -type f ! -size 0")
	if err != nil {
		return fmt.Errorf("error returning the file list to copy: %s", err)
	}

	for _, f := range linesToArray(filesToClone) {
		// TODO: Should return [] if the string is empty, but returns [""].
		if f == "" {
			continue
		}
		err := d.sh("cp", strconv.Quote(f), strconv.Quote(dstDir))
		if err != nil {
			return fmt.Errorf("error copying %s to %s: %s", f, dstDir, err)
		}
	}

	disksToClone, err := d.run(nil, "sed -ne 's/.*file[Nn]ame = \"\\(.*vmdk\\)\"/\\1/p'", strconv.Quote(srcVmx))
	if err != nil {
		return fmt.Errorf("error returning the .vmdk file list to clone:%s", err)
	}
	for _, disk := range linesToArray(disksToClone) {
		srcDisk := path.Join(srcDir, disk)
		if path.IsAbs(disk) {
			srcDisk = disk
		}
		destDisk := path.Join(dstDir, path.Base(disk))
		err = d.sh("vmkfstools", "-d thin", "-i", strconv.Quote(srcDisk), strconv.Quote(destDisk))
		if err != nil {
			return fmt.Errorf("error cloning disk %s: %s", srcDisk, err)
		}
	}
	log.Printf("Successfully cloned %s to %s\n", src, dst)
	return nil
}

func (d *EsxiDriver) CompactDisk(diskPathLocal string) error {
	diskPath := d.datastorePath(diskPathLocal)
	return d.sh("vmkfstools", "--punchzero", strconv.Quote(diskPath))
}

func (d *EsxiDriver) CreateDisk(diskPathLocal string, size string, adapterType string, typeId string) error {
	diskPath := strconv.Quote(d.datastorePath(diskPathLocal))
	return d.sh("vmkfstools", "-c", size, "-d", typeId, "-a", adapterType, diskPath)
}

func (d *EsxiDriver) CreateSnapshot(vmxPath string, snapshotName string) error {
	_, err := d.run(nil, "vim-cmd", "vmsvc/snapshot.create", d.vmId, snapshotName)
	return err
}

func (d *EsxiDriver) IsRunning(string) (bool, error) {
	state, err := d.run(nil, "vim-cmd", "vmsvc/power.getstate", d.vmId)
	if err != nil {
		return false, err
	}
	return strings.Contains(state, "Powered on"), nil
}

func (d *EsxiDriver) ReloadVM() error {
	if d.vmId != "" {
		return d.sh("vim-cmd", "vmsvc/reload", d.vmId)
	} else {
		return nil
	}
}

func (d *EsxiDriver) Start(vmxPathLocal string, headless bool) error {
	for i := 0; i < 20; i++ {
		// Do not check for error. The power on operation may fail after the
		// initial virtual machine registration.
		_ = d.sh("vim-cmd", "vmsvc/power.on", d.vmId)
		time.Sleep((time.Duration(i) * time.Second) + 1)
		running, err := d.IsRunning(vmxPathLocal)
		if err != nil {
			return err
		}
		if running {
			return nil
		}
	}
	return errors.New("exceeded maximum retries to start the virtual machine")
}

func (d *EsxiDriver) Stop(vmxPathLocal string) error {
	return d.sh("vim-cmd", "vmsvc/power.off", d.vmId)
}

func (d *EsxiDriver) Register(vmxPathLocal string) error {
	vmxPath := filepath.ToSlash(filepath.Join(d.outputDir, filepath.Base(vmxPathLocal)))
	if err := d.upload(vmxPath, vmxPathLocal, nil); err != nil {
		return err
	}
	r, err := d.run(nil, "vim-cmd", "solo/registervm", strconv.Quote(vmxPath))
	if err != nil {
		return err
	}
	d.vmId = strings.TrimRight(r, "\n")
	return nil
}

func (d *EsxiDriver) SuppressMessages(vmxPath string) error {
	return nil
}

func (d *EsxiDriver) Unregister(vmxPathLocal string) error {
	return d.sh("vim-cmd", "vmsvc/unregister", d.vmId)
}

func (d *EsxiDriver) Destroy() error {
	return d.sh("vim-cmd", "vmsvc/destroy", d.vmId)
}

func (d *EsxiDriver) IsDestroyed() (bool, error) {
	err := d.sh("test", "!", "-e", strconv.Quote(d.outputDir))
	if err != nil {
		return false, err
	}
	return true, err
}

func (d *EsxiDriver) UploadISO(localPath string, checksum string, ui packersdk.Ui) (string, error) {
	finalPath := d.CachePath(localPath)
	if err := d.mkdir(filepath.ToSlash(filepath.Dir(finalPath))); err != nil {
		return "", err
	}

	log.Printf("Verifying checksum of %s", finalPath)
	if d.VerifyChecksum(checksum, finalPath) {
		log.Println("Checksum matched. Skipping upload.")
		return finalPath, nil
	}
	log.Println("Checksum did not match. Uploading...")

	if err := d.upload(finalPath, localPath, ui); err != nil {
		return "", err
	}

	if !d.VerifyChecksum(checksum, finalPath) {
		e := fmt.Errorf("error verifying checksum for %s", finalPath)
		log.Println(e)
		return "", e
	}

	return finalPath, nil
}

func (d *EsxiDriver) RemoveCache(localPath string) error {
	finalPath := d.CachePath(localPath)
	log.Printf("Removing remote cache path %s (local %s)...", finalPath, localPath)
	return d.sh("rm", "-f", strconv.Quote(finalPath))
}

func (d *EsxiDriver) ToolsIsoPath(string) string {
	return ""
}

func (d *EsxiDriver) ToolsInstall() error {
	return d.sh("vim-cmd", "vmsvc/tools.install", d.vmId)
}

func (d *EsxiDriver) Verify() error {
	// Ensure that NetworkMapper is nil. The mapping of device<->network
	// are handled by ESXi and thus can't be performed unless queried.

	// TODO: To expose the network devices to the user, esxcli may need to to be
	// used to enumerate the portgroup and switchId.
	d.base.NetworkMapper = nil

	// Be safe/friendly and overwrite the rest of the utility functions with
	// log functions despite the fact that these should not be called.
	d.base.DhcpLeasesPath = func(device string) string {
		log.Printf("unexpected error, ESXi driver attempted to call DhcpLeasesPath(%#v)\n", device)
		return ""
	}

	d.base.DhcpConfPath = func(device string) string {
		log.Printf("unexpected error: ESXi driver attempted to call DhcpConfPath(%#v)\n", device)
		return ""
	}

	d.base.VmnetnatConfPath = func(device string) string {
		log.Printf("unexpected error: ESXi driver attempted to call VmnetnatConfPath(%#v)\n", device)
		return ""
	}

	checks := []func() error{
		d.connect,
		d.checkSystemVersion,
		d.checkGuestIPHackEnabled,
	}

	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

func (d *EsxiDriver) VerifyOvfTool(SkipExport, skipValidateCredentials bool) error {
	// If we aren't exporting a virtual machine, we don't need to verify the
	// ovftool is installed.
	if SkipExport {
		return nil
	}

	err := d.base.VerifyOvfTool(SkipExport, skipValidateCredentials)
	if err != nil {
		return err
	}

	if skipValidateCredentials {
		return nil
	}

	log.Printf("Verifying credentials for ovftool...")
	ovftool := GetOvfTool()

	if d.Password == "" {
		return fmt.Errorf("exporting the virtual machine from esxi with " +
			"ovftool requires a value for 'remote_password'")
	}

	// Generate the uri of the host, with embedded credentials
	ovftoolUri := fmt.Sprintf("vi://%s", d.Host)
	u, err := url.Parse(ovftoolUri)
	if err != nil {
		return fmt.Errorf("error generating uri for ovftool: %s", err)
	}
	u.User = url.UserPassword(d.Username, d.Password)

	ovfToolArgs := []string{"--noSSLVerify", "--verifyOnly", u.String()}

	var out bytes.Buffer
	cmdCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, ovftool, ovfToolArgs...)
	cmd.Stdout = &out

	// Manually close stdin. If not, the process will hang if a user provides an
	// invalid password or username.
	stdin, _ := cmd.StdinPipe()
	defer stdin.Close()

	if err := cmd.Run(); err != nil {
		outString := out.String()
		// The command *should* fail with the following error if it
		// authenticates properly.
		if !strings.Contains(outString, "Found wrong kind of object") {
			err := fmt.Errorf("ovftool validation error: %s; %s",
				err, outString)
			if strings.Contains(outString,
				"Enter login information for source") {
				err = fmt.Errorf("invalid username or password for ovftool")
			}
			return err
		}
	}

	return nil
}

func (d *EsxiDriver) HostIP(multistep.StateBag) (string, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", d.Host, d.Port))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	host, _, err := net.SplitHostPort(conn.LocalAddr().String())
	return host, err
}

func (d *EsxiDriver) PotentialGuestIP(multistep.StateBag) ([]string, error) {
	// Validate the user-defined address for the host.
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", d.Host, d.Port))
	if err != nil {
		return []string{}, err
	}
	defer conn.Close()

	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	return []string{host}, err
}

func (d *EsxiDriver) HostAddress(multistep.StateBag) (string, error) {
	// make a connection
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", d.Host, d.Port))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// get the local address (the host)
	host, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return "", fmt.Errorf("error determining host address: %v", err)
	}

	// Interate through the interfaces.
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("error enumerating host interfaces: %v", err)
	}

	for _, intf := range interfaces {
		addrs, err := intf.Addrs()
		if err != nil {
			continue
		}

		// Check if the addresses match the host address
		for _, addr := range addrs {
			if addr.String() == host {
				return intf.HardwareAddr.String(), nil
			}
		}
	}

	// If an interface is not found, return an error.
	return "", fmt.Errorf("error locating an interface matching host address: %v", host)
}

func (d *EsxiDriver) GuestAddress(multistep.StateBag) (string, error) {
	// Return all the interfaces on the host.
	r, err := d.esxcli("network", "ip", "interface", "list")
	if err != nil {
		return "", fmt.Errorf("error retrieving host interfaces: %v", err)
	}

	// Extract the interface name and MAC address from the output.
	addrs := make(map[string]string)
	for record, err := r.read(); record != nil && err == nil; record, err = r.read() {
		if strings.ToUpper(record["Enabled"]) != "TRUE" {
			continue
		}
		addrs[record["Name"]] = record["MAC Address"]
	}

	// Return all the IP addresses of the host.
	r, err = d.esxcli("network", "ip", "interface", "ipv4", "get")
	if err != nil {
		return "", fmt.Errorf("error retrieving the host addresses : %v", err)
	}

	// Find the interface name that matches the IP address of the host.
	var intf string
	intf = ""
	for record, err := r.read(); record != nil && err == nil; record, err = r.read() {
		if record["IPv4 Address"] == d.Host && record["Name"] != "" {
			intf = record["Name"]
			break
		}
	}
	if intf == "" {
		return "", fmt.Errorf("error matching address for guest")
	}

	// Find the MAC address for the interface.
	result, ok := addrs[intf]
	if !ok {
		return "", fmt.Errorf("error finding MAC address for interface %s", intf)
	}

	return result, nil
}

func (d *EsxiDriver) VNCAddress(ctx context.Context, _ string, portMin, portMax int) (string, int, error) {
	var vncPort int

	// Process ports ESXi is listening on to determine which are available.
	// This process detects which are unavailable ignore any ports listened to
	// by only localhost.
	r, err := d.esxcli("network", "ip", "connection", "list")
	if err != nil {
		err = fmt.Errorf("error retrieving network information for host: %v", err)
		return "", 0, err
	}

	listenPorts := make(map[string]bool)
	for record, err := r.read(); record != nil && err == nil; record, err = r.read() {
		if record["State"] == "LISTEN" {
			splitAddress := strings.Split(record["LocalAddress"], ":")
			if splitAddress[0] != "127.0.0.1" {
				port := splitAddress[len(splitAddress)-1]
				log.Printf("ESXi listening on address %s, port %s unavailable for VNC", record["LocalAddress"], port)
				listenPorts[port] = true
			}
		}
	}

	vncTimeout := 15 * time.Second
	envTimeout := os.Getenv("PACKER_ESXI_VNC_PROBE_TIMEOUT")
	if envTimeout != "" {
		if parsedTimeout, err := time.ParseDuration(envTimeout); err != nil {
			log.Printf("Error parsing PACKER_ESXI_VNC_PROBE_TIMEOUT. Using the default timeout of %s: %v", vncTimeout, err)
		} else {
			vncTimeout = parsedTimeout
		}
	}

	for port := portMin; port <= portMax; port++ {
		if _, ok := listenPorts[fmt.Sprintf("%d", port)]; ok {
			log.Printf("Port %d in use", port)
			continue
		}
		address := fmt.Sprintf("%s:%d", d.Host, port)
		log.Printf("Trying IP address: %s...", address)
		l, err := net.DialTimeout("tcp", address, vncTimeout)

		if err != nil {
			var opError *net.OpError
			if errors.As(err, &opError) {
				if opError.Timeout() {
					log.Printf("Timeout connecting to: %s (check firewall rules)", address)
				} else {
					vncPort = port
					break
				}
			}
		} else {
			defer l.Close()
		}
	}

	if vncPort == 0 {
		err := fmt.Errorf("error finding an available VNC port between %d and %d",
			portMin, portMax)
		return d.Host, vncPort, err
	}

	return d.Host, vncPort, nil
}

// UpdateVMX configures the VMX file with VNC parameters such as enablement, port, and optional password.
func (EsxiDriver) UpdateVMX(_, password string, port int, data map[string]string) {
	// Important: Do not set 'remotedisplay.vnc.ip' on ESXi hosts.
	data["remotedisplay.vnc.enabled"] = "TRUE"
	data["remotedisplay.vnc.port"] = fmt.Sprintf("%d", port)
	if len(password) > 0 {
		data["remotedisplay.vnc.password"] = password
	}
}

func (d *EsxiDriver) CommHost(state multistep.StateBag) (string, error) {
	sshc := state.Get("sshConfig").(*SSHConfig).Comm
	port := sshc.Port()

	if address, ok := state.GetOk("vm_address"); ok {
		return address.(string), nil
	}

	if address := d.CommConfig.Host(); address != "" {
		state.Put("vm_address", address)
		return address, nil
	}

	r, err := d.esxcli("network", "vm", "list")
	if err != nil {
		return "", err
	}

	// The value in the name field returned by 'esxcli network vm list'
	// corresponds directly to the value of displayName set in the .vmx file.
	var displayName string
	if v, ok := state.GetOk("display_name"); ok {
		displayName = v.(string)
	} else {
		displayName = strings.ReplaceAll(d.VMName, " ", "_")
		log.Printf("No 'display_name' set; using 'vmname' %s "+
			"to look for an IP address for SSH", displayName)
	}

	record, err := r.find("Name", displayName)
	if err != nil {
		return "", err
	}

	wid := record["WorldID"]
	if wid == "" {
		return "", errors.New("unable to find worldid")
	}

	r, err = d.esxcli("network", "vm", "port", "list", "-w", wid)
	if err != nil {
		return "", err
	}

	// Loop through interfaces.
	for {
		record, err = r.read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if record["IPAddress"] == "0.0.0.0" {
			continue
		}

		// if SSH is going through a bastion, it's not easy to check if the NIC
		// is reachable on the network. Select the first NIC that is not equal
		// to 0.0.0.0.
		if sshc.SSHBastionHost != "" {
			address := record["IPAddress"]
			state.Put("vm_address", address)
			return address, nil
		}

		// When multiple NICs are connected to the same network, choose
		// one that has a route back.
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", record["IPAddress"], port), 2*time.Second)
		if err != nil {
			if e, ok := err.(*net.OpError); ok {
				if e.Timeout() {
					log.Printf("Timeout connecting to %s", record["IPAddress"])
					continue
				}
			}
		} else {
			defer conn.Close()
			address := record["IPAddress"]
			state.Put("vm_address", address)
			return address, nil
		}
	}
	return "", errors.New("no interface ready with an IP address")
}

//-------------------------------------------------------------------
// OutputDir implementation
//-------------------------------------------------------------------

func (d *EsxiDriver) DirExists() (bool, error) {
	err := d.sh("test", "-e", strconv.Quote(d.outputDir))
	return err == nil, nil
}

func (d *EsxiDriver) ListFiles() ([]string, error) {
	stdout, err := d.ssh("ls -1p "+strconv.Quote(d.outputDir), nil)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, 10)
	reader := bufio.NewReader(stdout)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if line[len(line)-1] == '/' {
			continue
		}

		files = append(files, filepath.ToSlash(filepath.Join(d.outputDir, string(line))))
	}

	return files, nil
}

func (d *EsxiDriver) MkdirAll() error {
	return d.mkdir(d.outputDir)
}

func (d *EsxiDriver) Remove(path string) error {
	return d.sh("rm", strconv.Quote(path))
}

func (d *EsxiDriver) RemoveAll() error {
	return d.sh("rm", "-rf", strconv.Quote(d.outputDir))
}

func (d *EsxiDriver) SetOutputDir(path string) {
	d.outputDir = d.datastorePath(path)
}

func (d *EsxiDriver) String() string {
	return d.outputDir
}

func (d *EsxiDriver) datastorePath(path string) string {
	dirPath := filepath.Dir(path)
	return filepath.ToSlash(filepath.Join("/vmfs/volumes", d.Datastore, dirPath, filepath.Base(path)))
}

func (d *EsxiDriver) CachePath(path string) string {
	return filepath.ToSlash(filepath.Join("/vmfs/volumes", d.CacheDatastore, d.CacheDirectory, filepath.Base(path)))
}

func (d *EsxiDriver) connect() error {
	address := fmt.Sprintf("%s:%d", d.Host, d.Port)

	auth := []gossh.AuthMethod{
		gossh.Password(d.Password),
		gossh.KeyboardInteractive(
			ssh.PasswordKeyboardInteractive(d.Password)),
	}

	if d.PrivateKeyFile != "" {
		signer, err := helperssh.FileSigner(d.PrivateKeyFile)
		if err != nil {
			return err
		}

		auth = append(auth, gossh.PublicKeys(signer))
	}

	sshConfig := &ssh.Config{
		Connection: ssh.ConnectFunc("tcp", address),
		SSHConfig: &gossh.ClientConfig{
			User:            d.Username,
			Auth:            auth,
			HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		},
	}

	comm, err := ssh.New(address, sshConfig)
	if err != nil {
		return err
	}

	d.comm = comm
	return nil
}

func (d *EsxiDriver) checkSystemVersion() error {
	r, err := d.esxcli("system", "version", "get")
	if err != nil {
		return err
	}

	record, err := r.read()
	if err != nil {
		return err
	}

	log.Printf("Connected to %s %s %s", record["Product"],
		record["Version"], record["Build"])
	return nil
}

func (d *EsxiDriver) checkGuestIPHackEnabled() error {
	r, err := d.esxcli("system", "settings", "advanced", "list", "-o", "/Net/GuestIPHack")
	if err != nil {
		return err
	}

	record, err := r.read()
	if err != nil {
		return err
	}

	if record["IntValue"] != "1" {
		return errors.New(
			"guestiphack is required, enable by running this on the host:\n" +
				"esxcli system settings advanced set -o /Net/GuestIPHack -i 1")
	}

	return nil
}

func (d *EsxiDriver) mkdir(path string) error {
	return d.sh("mkdir", "-p", strconv.Quote(path))
}

func (d *EsxiDriver) upload(dst, src string, ui packersdk.Ui) error {
	// Get the size to set up the progress tracker.
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	if ui != nil {
		pf := ui.TrackProgress(filepath.Base(src), 0, info.Size(), f)
		defer pf.Close()

		return d.comm.Upload(dst, pf, &info)
	}

	return d.comm.Upload(dst, f, nil)
}

func (d *EsxiDriver) Download(src, dst string) error {
	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer file.Close()
	return d.comm.Download(d.datastorePath(src), file)
}

func (d *EsxiDriver) Export(args []string) error {
	return d.base.Export(args)
}

// VerifyChecksum checks that file on the esxi instance matches hash
func (d *EsxiDriver) VerifyChecksum(hash string, file string) bool {
	if hash == "none" {
		if err := d.sh("stat", strconv.Quote(file)); err != nil {
			return false
		}
		return true
	}

	// parse user checksum
	fcksum, err := getter.DefaultClient.GetChecksum(context.TODO(), &getter.Request{
		Src: file + "?checksum=" + hash,
	})
	if err != nil {
		log.Printf("[ERROR] Error parsing the checksum: %s", err)
		return false
	}

	checksumEntry := fmt.Sprintf("%s  %s", hex.EncodeToString(fcksum.Value), file)
	checksumCommand := []string{fmt.Sprintf("%ssum", fcksum.Type), "-c"}

	log.Printf("running: %s | %s", checksumEntry, checksumCommand)

	_, err = d.run(bytes.NewBufferString(checksumEntry), checksumCommand...)
	if err != nil {
		log.Printf("Checksum failed: %s", err)
	}

	return err == nil
}

func (d *EsxiDriver) ssh(command string, stdin io.Reader) (*bytes.Buffer, error) {
	ctx := context.TODO()
	var stdout, stderr bytes.Buffer

	cmd := &packersdk.RemoteCmd{
		Command: command,
		Stdout:  &stdout,
		Stderr:  &stderr,
		Stdin:   stdin,
	}

	err := d.comm.Start(ctx, cmd)
	if err != nil {
		return nil, err
	}

	cmd.Wait()

	if cmd.ExitStatus() != 0 {
		err = fmt.Errorf("'%s'\n\nStdout: %s\n\nStderr: %s",
			cmd.Command, stdout.String(), stderr.String())
		return nil, err
	}

	return &stdout, nil
}

func (d *EsxiDriver) run(stdin io.Reader, args ...string) (string, error) {
	stdout, err := d.ssh(strings.Join(args, " "), stdin)
	if err != nil {
		return "", err
	}
	return stdout.String(), nil
}

func (d *EsxiDriver) sh(args ...string) error {
	_, err := d.run(nil, args...)
	return err
}

func (d *EsxiDriver) esxcli(args ...string) (*esxcliReader, error) {
	stdout, err := d.ssh("esxcli --formatter csv "+strings.Join(args, " "), nil)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(bytes.NewReader(stdout.Bytes()))
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	return &esxcliReader{r, header}, nil
}

func (d *EsxiDriver) GetVmwareDriver() VmwareDriver {
	return d.base
}

type esxcliReader struct {
	cr     *csv.Reader
	header []string
}

func (r *esxcliReader) read() (map[string]string, error) {
	fields, err := r.cr.Read()

	if err != nil {
		return nil, err
	}

	record := map[string]string{}
	for i, v := range fields {
		record[r.header[i]] = v
	}

	return record, nil
}

func (r *esxcliReader) find(key, val string) (map[string]string, error) {
	for {
		record, err := r.read()
		if err != nil {
			return nil, err
		}
		if record[key] == val {
			return record, nil
		}
	}
}

func (d *EsxiDriver) AcquireVNCOverWebsocketTicket() (*types.VirtualMachineTicket, error) {
	vm, err := d.finder.VirtualMachine(d.ctx, d.VMName)
	if err != nil {
		return nil, err
	}
	return vm.AcquireTicket(d.ctx, string(types.VirtualMachineTicketTypeWebmks))
}
