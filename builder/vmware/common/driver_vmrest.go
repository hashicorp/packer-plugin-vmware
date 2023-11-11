package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// VMRest driver talks to the VMWare Workstation Pro API
// tested against vmrest 1.3.1
type VMRestDriver struct {
	base VmwareDriver

	RemoteHost string
	BaseURL    string
	User       string
	Password   string
	SSHConfig  *SSHConfig
	VMName     string
	VMId       string

	//TODO unsure if i need these
	outputDir string
}

func NewVMRestDriver(dconfig *DriverConfig, config *SSHConfig, vmName string) (Driver, error) {
	baseURL := "http://" + dconfig.RemoteHost + ":" + strconv.Itoa(dconfig.RemotePort) + "/api"
	return &VMRestDriver{
		RemoteHost: dconfig.RemoteHost,
		BaseURL:    baseURL,
		User:       dconfig.RemoteUser,
		Password:   dconfig.RemotePassword,
		SSHConfig:  config,
		VMName:     vmName,
	}, nil
}

/*
Implement the Driver interface
*/

// Clone clones the VMX and the disk to the destination path. The
// destination is a path to the VMX file. The disk will be copied
// to that same directory.
func (d *VMRestDriver) Clone(dstVMX string, srcVMX string, linked bool, snapshot string) error {
	vmId, err := d.GetVMId(srcVMX)
	if err != nil {
		log.Print("Failed to retrieve the source VM Id")
		return err
	}
	body := `{"name":"` + d.VMName + `", "parentId":"` + vmId + `"}`
	response, err := d.MakeVMRestRequest("POST", "/vms", body)
	if err != nil {
		log.Print("Failed to retrieve VM configuration from the API")
		return err
	}
	type newVM struct {
		Id string `json:"id"`
	}
	var data newVM
	err = json.Unmarshal([]byte(response), &data)
	if err != nil {
		log.Print("Failed to parse the API response")
		return err
	}
	d.VMId = data.Id
	log.Printf("Successfully cloned VM; New VM ID is %v", data.Id)
	return nil
}

// CompactDisk compacts a virtual disk.
func (d *VMRestDriver) CompactDisk(string) error {
	return errors.New("Compacting disks is not supported by the VMRest API")
}

// CreateDisk creates a virtual disk with the given size.
func (d *VMRestDriver) CreateDisk(string, string, string, string) error {
	return errors.New("Creating disks is not supported by the VMRest API")
}

// CreateSnapshot creates a snapshot of the supplied .vmx file with
// the given name
func (d *VMRestDriver) CreateSnapshot(string, string) error {
	return errors.New("Creating snapshots is not supported by the VMRest API")
}

// Checks if the VMX file at the given path is running.
func (d *VMRestDriver) IsRunning(vmxPath string) (bool, error) {
	vmId, err := d.GetVMId(vmxPath)
	if err != nil {
		return false, err
	}
	response, err := d.MakeVMRestRequest("GET", "/vms/"+vmId+"/power", "")
	if err != nil {
		return false, err
	}
	state := ParsePowerResponse(response)
	// will never end up here
	return state, nil
}

// Start starts a VM specified by the path to the VMX given.
func (d *VMRestDriver) Start(vmxPath string, headless bool) error {
	vmId, err := d.GetVMId(vmxPath)
	if err != nil {
		return err
	}
	response, err := d.MakeVMRestRequest("PUT", "/vms/"+vmId+"/power", "on")
	if err != nil {
		return err
	}
	state := ParsePowerResponse(response)
	if state {
		return nil
	} else {
		return errors.New("API call was not successful in turning the VM on")
	}
}

// Stops a VM specified by the path to a VMX file.
func (d *VMRestDriver) Stop(vmxPath string) error {
	vmId, err := d.GetVMId(vmxPath)
	if err != nil {
		return err
	}
	response, err := d.MakeVMRestRequest("PUT", "/vms/"+vmId+"/power", "off")
	if err != nil {
		return err
	}
	state := ParsePowerResponse(response)
	if !state {
		return nil
	} else {
		return errors.New("API call was not successful in turning the VM off")
	}
}

// SuppressMessages modifies the VMX or surrounding directory so that
// VMware doesn't show any annoying messages.
func (d *VMRestDriver) SuppressMessages(string) error {
	return nil
}

// Get the path to the VMware ISO for the given flavor.
func (d *VMRestDriver) ToolsIsoPath(string) string {
	// return a string to avoid throwing any errors
	return ""
}

// Attach the VMware tools ISO
func (d *VMRestDriver) ToolsInstall() error {
	return errors.ErrUnsupported
}

// Verify checks to make sure that this driver should function
// properly. This should check that all the files it will use
// appear to exist and so on. If everything is okay, this doesn't
// return an error. Otherwise, this returns an error. Each vmware
// driver should assign the VmwareMachine callback functions for locating
// paths within this function.
func (d *VMRestDriver) Verify() error {
	// Be safe/friendly and overwrite the rest of the utility functions with
	// log functions despite the fact that these shouldn't be called anyways.
	d.base.DhcpLeasesPath = func(device string) string {
		log.Printf("Unexpected error, VMRest driver attempted to call DhcpLeasesPath(%#v)\n", device)
		return ""
	}
	d.base.DhcpConfPath = func(device string) string {
		log.Printf("Unexpected error, VMRest driver attempted to call DhcpConfPath(%#v)\n", device)
		return ""
	}
	d.base.VmnetnatConfPath = func(device string) string {
		log.Printf("Unexpected error, VMRest driver attempted to call VmnetnatConfPath(%#v)\n", device)
		return ""
	}

	// Make sure we can connect to the remote server
	response, err := d.MakeVMRestRequest("GET", "", "")
	if response == "404" {
		log.Print("Got expected response from remote server. Proceeding with VMRest driver.")
		return nil
	}

	if err != nil {
		log.Print(err.Error())
	}
	return errors.New("Did not receive expected response from remote server. VMRest driver verification failed.")
}

// This is to establish a connection to the guest
func (d *VMRestDriver) CommHost(state multistep.StateBag) (string, error) {
	return CommHost(d.SSHConfig)(state)
}

// These methods are generally implemented by the VmwareDriver
// structure within this file. A driver implementation can
// reimplement these, though, if it wants.
func (d *VMRestDriver) GetVmwareDriver() VmwareDriver {
	return d.base
}

// Get the guest hw address for the vm
func (d *VMRestDriver) GuestAddress(state multistep.StateBag) (string, error) {
	vmxPath := state.Get("vmx_path").(string)
	vmId, err := d.GetVMId(vmxPath)
	if err != nil {
		log.Print("Failed to retrieve VM Id")
		return "", err
	}
	response, err := d.MakeVMRestRequest("GET", "/vms/"+vmId+"/restrictions", "")
	if err != nil {
		log.Print("Failed to retrieve VM configuration from the API")
		return "", err
	}
	// attempt parsing the JSON response
	type nic struct {
		Index int    `json:"index"`
		Type  string `json:"type"`
		VMNet string `json:"vmnet"`
		MAC   string `json:"macAddress"`
	}
	type nicList struct {
		Nics []nic `json:"nics"`
	}
	type restrictions struct {
		NicList nicList `json:"nicList"`
	}
	var data restrictions
	err = json.Unmarshal([]byte(response), &data)
	if err != nil {
		log.Print("Failed to parse the API response")
		return "", err
	}
	for _, nic := range data.NicList.Nics {
		if nic.Index == 1 {
			log.Printf("Found the following MAC address for the VM: %v", nic.MAC)
			return nic.MAC, nil
		}
	}
	return "", errors.New("Failed to find a MAC address for the VM")
}

// Get the guest ip address for the vm
// It takes awhile for vmware to detect the IPs
// Not sure if 'timeout'/repeated attempts should be handled here or elsewhere
func (d *VMRestDriver) PotentialGuestIP(state multistep.StateBag) ([]string, error) {
	ips := make([]string, 0)
	vmxPath := state.Get("vmx_path").(string)
	vmId, err := d.GetVMId(vmxPath)
	if err != nil {
		log.Print("Failed to retrieve VM Id")
		return ips, err
	}
	response, err := d.MakeVMRestRequest("GET", "/vms/"+vmId+"/nicips", "")
	if err != nil {
		log.Print("Failed to retrieve VM NIC configuration(s) from the API")
		return ips, err
	}
	// attempt parsing the JSON response
	type nic struct {
		MAC    string   `json:"mac"`
		IpList []string `json:"ip"`
	}
	type nicInfo struct {
		NicList []nic `json:"nics"`
	}
	var data nicInfo
	err = json.Unmarshal([]byte(response), &data)
	if err != nil {
		log.Print("Failed to parse the API response")
		return ips, err
	}
	for _, nic := range data.NicList {
		if len(nic.IpList) > 0 {
			for _, ip := range nic.IpList {
				// strip subnet mask and ignore IPv6
				pattern, _ := regexp.Compile(`(\d{1,3}\.){3}\d{1,3}`)
				match := pattern.FindString(ip)
				if len(match) > 0 {
					log.Printf("Found the following IP address for the VM: %v", match)
					ips = append(ips, match)
				}
			}
		}
	}
	if len(ips) == 0 {
		return ips, errors.New("Failed to find an IP address for the VM")
	}
	return ips, nil
}

// Get the host hw address for the vm
// TODO - or not to do? not sure this is needed
func (d *VMRestDriver) HostAddress(state multistep.StateBag) (string, error) {
	/* get the requested network type
	network := state.Get("vmnetwork").(string)
	// get all vmware networks
	response, err := d.MakeVMRestRequest("GET", "/vmnet", "")
	if err != nil {
		log.Print("Failed to retrieve vmnet configuration(s) from the API")
		return "", err
	}


	// attempt parsing the JSON response
	type vmnet struct {
		Name string `json:""`
		Type string `json:""`
		DHCP string `json:""`
		Subnet string `json:""`
		Mask	string `json:""`
	}
	type nicInfo struct {
		NicList []nic `json:"nics"`
	}
	var data nicInfo
	err = json.Unmarshal([]byte(response), &data)
	if err != nil {
		log.Print("Failed to parse the API response")
		return ips, err
	}
	for _, nic := range data.NicList {
		if len(nic.IpList) > 0 {
			for _, ip := range nic.IpList {
				// strip subnet mask and ignore IPv6
				pattern, _ := regexp.Compile(`(\d{1,3}\.){3}\d{1,3}`)
				match := pattern.FindString(ip)
				if len(match) > 0 {
					log.Printf("Found the following IP address for the VM: %v", match)
					ips = append(ips, match)
				}
			}
		}
	}
	*/
	return "nil", nil
}

// Get the host ip address for the vm
func (d *VMRestDriver) HostIP(multistep.StateBag) (string, error) {
	// note that we want the local IP, as this is used http_ip
	// we do NOT want the vmnet's IP
	// StackOverflow seems to agree that dialing a connection is
	// the most reliable method of determining the 'primary' host IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// Export the vm to ovf or ova format using ovftool
func (d *VMRestDriver) Export([]string) error {
	return errors.ErrUnsupported
}

// OvfTool
func (d *VMRestDriver) VerifyOvfTool(skipExport bool, skipValidateCredentials bool) error {
	if skipExport {
		return nil
	} else {
		return errors.ErrUnsupported
	}
}

/*
Implement the OutputDir interface
VMWare will set the output dir and we have no control over it
We just need dummy interfaces to avoid errors
*/

func (d *VMRestDriver) DirExists() (bool, error) {
	return false, nil
}

func (d *VMRestDriver) ListFiles() ([]string, error) {
	return []string{}, nil
}

func (d *VMRestDriver) MkdirAll() error {
	return nil
}

func (d *VMRestDriver) Remove(string) error {
	return nil
}

func (d *VMRestDriver) RemoveAll() error {
	return nil
}

func (d *VMRestDriver) SetOutputDir(string) {
	log.Print(
		"Warning: the VMRest API does not support setting the output dir",
		"If an output dir was provided, it will be ignored",
	)
	return
}

func (d *VMRestDriver) String() string {
	return ""
}

/*
Implement the RemoteDriver interface
*/
//TODO

// UploadISO uploads a local ISO to the remote side and returns the
// new path that should be used in the VMX along with an error if it
// exists.
func (d *VMRestDriver) UploadISO(path string, checksum string, ui packersdk.Ui) (string, error) {
	return "", errors.New("The VMRest driver does not support uploading an ISO")
}

// RemoveCache deletes localPath from the remote cache.
func (d *VMRestDriver) RemoveCache(localPath string) error {
	return nil
}

// Adds a VM to inventory specified by the path to the VMX given.
func (d *VMRestDriver) Register(path string) error {
	return nil
}

// Removes a VM from inventory specified by the path to the VMX given.
func (d *VMRestDriver) Unregister(path string) error {
	return nil
}

// Destroys a VM
func (d *VMRestDriver) Destroy() error {
	return nil
}

// Checks if the VM is destroyed.
func (d *VMRestDriver) IsDestroyed() (bool, error) {
	return true, nil
}

// Uploads a local file to remote side.
func (d *VMRestDriver) upload(dst, src string, ui packersdk.Ui) error {
	/*
		Note: this WOULD be used to upload an edited vmx file in StepUploadVMX
		However, after reviewing StepConfigureVMX, all changes that might have
		been made to our VMX are either unnecessary or unsupported by the API. So,
		we will simply skip the upload step.
		StepUploadVMX only acts if RemoteType is equal to 'exs5', so we don't need
		to do anything here
	*/
	return nil
}

// Download a remote file to a local file.
func (d *VMRestDriver) Download(src, dst string) error {
	/*
		the API doesn't allow us to retrieve all values at once, so
		we have to request each value individually. We will only retrieve
		the attributes that are strictly necessary
	*/
	of, err := os.Create(dst)
	if err != nil {
		panic(err)
	}
	requiredAttributes := []string{
		"ethernet0.connectiontype",
		"displayname",
	}
	type vmAttr struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	for _, attr := range requiredAttributes {
		response, err := d.MakeVMRestRequest("GET", "/vms/"+d.VMId+"/params/"+attr, "")
		if err != nil {
			return err
		}
		var vmAttribute vmAttr
		err = json.Unmarshal([]byte(response), &vmAttribute)
		if err != nil {
			log.Print("API call succeeded, but the response could not be parsed")
			return err
		}
		of.WriteString(fmt.Sprintf("%v = %v\n", vmAttribute.Name, vmAttribute.Value))
	}
	// add a dummy disk config to prevent errors
	of.WriteString("scsi0:0.fileName = notARealDisk.vmdk\n")
	err = of.Close()
	if err != nil {
		panic(err)
	}
	return nil
}

// Reload VM on remote side.
func (d *VMRestDriver) ReloadVM() error {
	return nil
}

/*
Implementation of the VNCAddressFinder interface
*/

func (d *VMRestDriver) VNCAddress(ctx context.Context, BindAddress string, PortMin int, PortMax int) (string, int, error) {
	// returns the VNC Bind Address + Port to be used in the VMX file
	// we want the VNC Bind Address to be the same as the RemoteIP
	var bindIP string
	if BindAddress != "0.0.0.0" && BindAddress != "127.0.0.1" {
		bindIP = BindAddress
	} else {
		isIP, err := regexp.Match(`(\d{1,3}\.){3}\d{1,3}`, []byte(d.RemoteHost))
		if err != nil {
			return "", 0, err
		}
		if isIP {
			bindIP = d.RemoteHost
		} else {
			ips, err := net.LookupIP(d.RemoteHost)
			if err != nil {
				return "", 0, errors.New("Failed to get RemoteHost IP")
			}
			bindIP = ips[0].String()
		}
	}

	return "", 0, nil
}

// UpdateVMX, sets driver specific VNC values to VMX data.
func (d *VMRestDriver) UpdateVMX(vncAddress, vncPassword string, vncPort int, vmxData map[string]string) {
	return
}

/*
Helper Functions for working with the VMRest API
*/

func (d *VMRestDriver) MakeVMRestRequest(method string, path string, body string) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, d.BaseURL+path, bytes.NewReader([]byte(body)))
		req.Header.Add("Content-Type", "application/vnd.vmware.vmw.rest-v1+json")
	} else {
		req, err = http.NewRequest(method, d.BaseURL+path, nil)
	}
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(d.User, d.Password)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if resp.StatusCode < 300 {
		if err != nil {
			return "", err
		}
		return string(bodyBytes), nil
	}
	msg := "Response Status: " + resp.Status + "\nResponse Body: " + string(bodyBytes)
	return msg, err
}

// Retrieves the VM ID based on the VMX path provided
func (d *VMRestDriver) GetVMId(vmxPath string) (string, error) {
	response, err := d.MakeVMRestRequest("GET", "/vms", "")
	if err != nil {
		log.Print("API call to /vms failed")
		return "", errors.New("Could not retrieve the VM's Id")
	}
	type vm struct {
		Id   string `json:"id"`
		Path string `json:"path"`
	}
	var vmList []vm
	err = json.Unmarshal([]byte(response), &vmList)
	if err != nil {
		log.Print("API call to /vms succeeded, but the response could not be parsed")
		return "", errors.New("Could not retrieve the VM's Id")
	}
	for _, v := range vmList {
		if v.Path == vmxPath {
			return v.Id, nil
		}
	}
	return "", errors.New("Could not find a VM with the given path")
}

// Parses the response of VMRest power operations
func ParsePowerResponse(response string) bool {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(response), &data)
	if err != nil {
		log.Fatal(err)
	}
	rawState, ok := data["power_state"]
	if ok {
		state, ok := rawState.(string)
		if ok {
			// state may be "poweredOn" or "poweringOn"
			if strings.Contains(state, "On") {
				return true
			} else if strings.Contains(state, "Off") {
				return false
			} else {
				log.Fatal("API response contained 'power_state' but the value was unexpected")
			}
		} else {
			log.Fatal("API return value was not a string")
		}
	} else {
		log.Fatal("API did not return the expected value (power_state)")
	}
	// should never end up here
	return false
}
