
package iso

import (
	"errors"
	"fmt"
	"strings"
	"log"
	"strconv"
//	"os"
	"time"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"github.com/mitchellh/packer/common"
	"code.google.com/p/go-uuid/uuid"
	
	hypervcommon "github.com/mitchellh/packer/builder/hyperv/common"
)
	

const (

	BuilderIdHyperVISO = "jamesleech.hyperv-iso"
	DiskTypeFixed = "Fixed"
	DiskTypeDynamic = "Dynamic"
	
	DefaultDiskSize = 127 * 1024	// 127GB
	MinDiskSize = 10 * 1024			// 10GB
	MaxDiskSize = 65536 * 1024		// 64TB

	DefaultRamSize = 1024	// 1GB
	MinRamSize = 512		// 512MB
	MaxRamSize = 32768 		// 32GB

	LowRam = 512 // 512MB

	DefaultUsername = "vagrant"
	DefaultPassword = "vagrant"
)

type Builder struct {
	config config
	runner multistep.Runner
}


type config struct {
	common.PackerConfig      			`mapstructure:",squash"`
	hypervcommon.OutputConfig			`mapstructure:",squash"`

	DiskName        	string   		`mapstructure:"vmdk_name"`
	DiskSize        	uint     		`mapstructure:"disk_size"`
	DiskType			string   		`mapstructure:"disk_type"`
	RamSizeMB       	uint     		`mapstructure:"ram_size_mb"`
	FloppyFiles     	[]string 		`mapstructure:"floppy_files"`
	ISOChecksum     	string   		`mapstructure:"iso_checksum"`
	ISOChecksumType 	string   		`mapstructure:"iso_checksum_type"`
	ISOUrls         	[]string 		`mapstructure:"iso_urls"`
	Version         	string   		`mapstructure:"version"`
	VMName          	string   		`mapstructure:"vm_name"`
	SkipCompaction  	bool     		`mapstructure:"skip_compaction"`
	RawSingleISOUrl 	string 			`mapstructure:"iso_url"`	
	SwitchName      	string			`mapstructure:"switch_name"`
	IPAddressTimeout    time.Duration	`mapstructure:"ip_address_timeout"`
	
	tpl *packer.ConfigTemplate
}

func (b *Builder) Prepare(raws ...interface{}) ([]string, error) {
	
	md, err := common.DecodeConfig(&b.config, raws...)
	if err != nil {
		return nil, err
	}	
	
	b.config.tpl, err = packer.NewConfigTemplate()
	if err != nil {
		return nil, err
	}
	b.config.tpl.UserVars = b.config.PackerUserVars
	
	driver, err := hypervcommon.NewHypervPS4Driver()
	if err != nil {
		return nil, fmt.Errorf("Failed creating Hyper-V driver: %s", err)
	}
	
	// Accumulate any errors and warnings
	errs := common.CheckUnusedConfig(md)
	errs = packer.MultiErrorAppend(errs, b.config.OutputConfig.Prepare(b.config.tpl, &b.config.PackerConfig)...)
	warnings := make([]string, 0)

	//DiskName        
	if b.config.DiskName == "" {
		b.config.DiskName = "disk"
	}
	
	//DiskSize
	err = b.checkDiskSize()
	if err != nil {
		errs = packer.MultiErrorAppend(errs, err)
	}

	//DiskType
	if b.config.DiskType == "" {
		// Default is dynamic
		b.config.DiskType = DiskTypeDynamic
	}
	
	if b.config.DiskType != DiskTypeDynamic || b.config.DiskType != DiskTypeFixed {
		errs = packer.MultiErrorAppend(errs,
			fmt.Errorf("disk_type: %s, invalid disk type, must be %s or %s", b.config.DiskType, DiskTypeDynamic, DiskTypeFixed))
	}
	
	log.Println(fmt.Sprintf("%s: %s", "DiskType", b.config.DiskType))
	
	//RamSizeMB
	
	err = b.checkRamSize()
	if err != nil {
		errs = packer.MultiErrorAppend(errs, err)
	}
	
	//FloppyFiles	
	if b.config.FloppyFiles == nil {
		b.config.FloppyFiles = make([]string, 0)
	}

	for i, file := range b.config.FloppyFiles {
		var err error
		b.config.FloppyFiles[i], err = b.config.tpl.Process(file, nil)
		if err != nil {
			errs = packer.MultiErrorAppend(errs,
				fmt.Errorf("Error processing floppy_files[%d]: %s",
					i, err))
		}
	}
	
	//ISOChecksum ISOChecksumType 
	
	if b.config.ISOChecksumType == "" {
		errs = packer.MultiErrorAppend(
			errs, errors.New("The iso_checksum_type must be specified."))
	} else {
		b.config.ISOChecksumType = strings.ToLower(b.config.ISOChecksumType)
		if b.config.ISOChecksumType != "none" {
			if b.config.ISOChecksum == "" {
				errs = packer.MultiErrorAppend(
					errs, errors.New("Due to large file sizes, an iso_checksum is required"))
			} else {
				b.config.ISOChecksum = strings.ToLower(b.config.ISOChecksum)
			}

			if h := common.HashForType(b.config.ISOChecksumType); h == nil {
				errs = packer.MultiErrorAppend(
					errs,
					fmt.Errorf("Unsupported checksum type: %s", b.config.ISOChecksumType))
			}
		}
	}	
	
	//ISOUrls
	for i, url := range b.config.ISOUrls {
		var err error
		b.config.ISOUrls[i], err = b.config.tpl.Process(url, nil)
		if err != nil {
			errs = packer.MultiErrorAppend(
				errs, fmt.Errorf("Error processing iso_urls[%d]: %s", i, err))
		}
	}     
	
	//Version
	if b.config.Version == "" {
		b.config.Version = "1"
	}         
	
	//VMName    
	if b.config.VMName == "" {
		b.config.VMName = fmt.Sprintf("packer-%s", b.config.PackerBuildName)
	}      

	//SkipCompaction  

	if b.config.RawSingleISOUrl == "" && len(b.config.ISOUrls) == 0 {
		errs = packer.MultiErrorAppend(
			errs, errors.New("One of iso_url or iso_urls must be specified."))
	} else if b.config.RawSingleISOUrl != "" && len(b.config.ISOUrls) > 0 {
		errs = packer.MultiErrorAppend(
			errs, errors.New("Only one of iso_url or iso_urls may be specified."))
	} else if b.config.RawSingleISOUrl != "" {
		b.config.ISOUrls = []string{b.config.RawSingleISOUrl}
	}

	//SwitchName
	
	if b.config.SwitchName == "" {
		// no switch name, try to get one attached to a online network adapter
		onlineSwitchName, err :=  getExternalOnlineVirtualSwitch(driver)
		if onlineSwitchName == "" || err != nil{
			b.config.SwitchName = fmt.Sprintf("pis_%s", uuid.New())
		} else {
			b.config.SwitchName = onlineSwitchName
		}
	}


	// Errors
	templates := map[string]*string{
		"disk_name":              &b.config.DiskName,
		"iso_checksum":           &b.config.ISOChecksum,
		"iso_checksum_type":      &b.config.ISOChecksumType,
		"iso_url":                &b.config.RawSingleISOUrl,
		"vm_name":                &b.config.VMName,
	}

	for n, ptr := range templates {
		var err error
		*ptr, err = b.config.tpl.Process(*ptr, nil)
		if err != nil {
			errs = packer.MultiErrorAppend(
				errs, fmt.Errorf("Error processing %s: %s", n, err))
		}
	}

	//RawSingleISOUrl

	if b.config.RawSingleISOUrl == "" && len(b.config.ISOUrls) == 0 {
		errs = packer.MultiErrorAppend(
			errs, errors.New("One of iso_url or iso_urls must be specified."))
	} else if b.config.RawSingleISOUrl != "" && len(b.config.ISOUrls) > 0 {
		errs = packer.MultiErrorAppend(
			errs, errors.New("Only one of iso_url or iso_urls may be specified."))
	} else if b.config.RawSingleISOUrl != "" {
		b.config.ISOUrls = []string{b.config.RawSingleISOUrl}
	}

	for i, url := range b.config.ISOUrls {
		b.config.ISOUrls[i], err = common.DownloadableURL(url)
		if err != nil {
			errs = packer.MultiErrorAppend(
				errs, fmt.Errorf("Failed to parse iso_url %d: %s", i+1, err))
		}
	}

	// Warnings
	warning := b.checkHostAvailableMemory(driver)
	if warning != "" {
		warnings = appendWarnings(warnings, warning)
	}

	if b.config.ISOChecksumType == "none" {
		warnings = append(warnings,
			"A checksum type of 'none' was specified. Since ISO files are so big,\n"+
				"a checksum is highly recommended.")
	}
	
	if errs != nil && len(errs.Errors) > 0 {
		return warnings, errs
	}

	return warnings, nil
}

func (b *Builder) Run(ui packer.Ui, hook packer.Hook, cache packer.Cache) (packer.Artifact, error) {
	// Create the driver that we'll use to communicate with Hyperv
	driver, err := hypervcommon.NewHypervPS4Driver()
	if err != nil {
		return nil, fmt.Errorf("Failed creating Hyper-V driver: %s", err)
	}

	// Set up the state.
	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("driver", driver)
	state.Put("hook", hook)
	state.Put("ui", ui)

	steps := []multistep.Step{

		new(hypervcommon.StepCreateTempDir),
		
		&hypervcommon.StepOutputDir{
			Force: b.config.PackerForce,
			Path:  b.config.OutputDir,
		},

//		&hypervcommon.StepSetUnattendedProductKey{
//			Files: b.config.FloppyFiles,
//			ProductKey: b.config.ProductKey,
//		},

		&common.StepCreateFloppy{
			Files: b.config.FloppyFiles,
		},
		
		&hypervcommon.StepCreateSwitch{
			SwitchName: b.config.SwitchName,
		},
		
		new(StepCreateVM),
		
		new(hypervcommon.StepEnableIntegrationService),
		
		new(StepMountDvdDrive),
		
		new(StepMountFloppydrive),
		
		//TODO: new(StepMountSecondaryDvdImages),
		
		new(hypervcommon.StepStartVm),
		
		// wait for the vm to be powered off
		&hypervcommon.StepWaitForPowerOff{},

		//TODO: &hypervcommon.StepUnmountSecondaryDvdImages{},

		new(hypervcommon.StepConfigureIp),
		
		new(hypervcommon.StepSetRemoting),
		
		new(common.StepProvision),

		new(StepExportVm),

//		new(hypervcommon.StepConfigureIp),
//		new(hypervcommon.StepSetRemoting),
//		new(hypervcommon.StepCheckRemoting),
//		new(msbldcommon.StepSysprep),
	}

	// Run the steps.
	if b.config.PackerDebug {
		b.runner = &multistep.DebugRunner{
			Steps:   steps,
			PauseFn: common.MultistepDebugFn(ui),
		}
	} else {
		b.runner = &multistep.BasicRunner{Steps: steps}
	}
	b.runner.Run(state)

	// Report any errors.
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	// If we were interrupted or cancelled, then just exit.
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		return nil, errors.New("Build was cancelled.")
	}

	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("Build was halted.")
	}

	return hypervcommon.NewArtifact(b.config.OutputDir)
}

func (b *Builder) Cancel() {
	if b.runner != nil {
		log.Println("Cancelling the hyperv iso step runner...")
		b.runner.Cancel()
	}
}

func appendWarnings(slice []string, data ...string) []string {
	m := len(slice)
	n := m + len(data)
	if n > cap(slice) { // if necessary, reallocate
		// allocate double what's needed, for future growth.
		newSlice := make([]string, (n+1)*2)
		copy(newSlice, slice)
		slice = newSlice
	}
	slice = slice[0:n]
	copy(slice[m:n], data)
	return slice
}

func (b *Builder) checkDiskSize() error {
	if b.config.DiskSize == 0 {
		b.config.DiskSize = DefaultDiskSize
	}

	log.Println(fmt.Sprintf("%s: %v", "DiskSize", b.config.DiskSize))

	if(b.config.DiskSize < MinDiskSize ){
		return fmt.Errorf("disk_size_gb: Windows server requires disk space >= %v GB, but defined: %v", MinDiskSize, b.config.DiskSize/1024)
	} else if b.config.DiskSize > MaxDiskSize {
		return fmt.Errorf("disk_size_gb: Windows server requires disk space <= %v GB, but defined: %v", MaxDiskSize, b.config.DiskSize/1024)
	}

	return nil	
}

func (b *Builder) checkRamSize() error {
	if b.config.RamSizeMB == 0 {
		b.config.RamSizeMB = DefaultRamSize
	}

	log.Println(fmt.Sprintf("%s: %v", "RamSize", b.config.RamSizeMB))

	if(b.config.RamSizeMB < MinRamSize ){
		return fmt.Errorf("ram_size_mb: Windows server requires memory size >= %v MB, but defined: %v", MinRamSize, b.config.RamSizeMB)
	} else if b.config.RamSizeMB > MaxRamSize {
		return fmt.Errorf("ram_size_mb: Windows server requires memory size <= %v MB, but defined: %v", MaxRamSize, b.config.RamSizeMB)
	}

	return nil
}

func (b *Builder) checkHostAvailableMemory(driver hypervcommon.Driver) string {
	freeMBStr, err := getHostAvailableMemory(driver)

	if err == nil {
		freeMB, err := strconv.ParseFloat(freeMBStr, 64)
		
		if err == nil || (freeMB - float64(b.config.RamSizeMB)) < LowRam {
			return fmt.Sprintf("Hyper-V might fail to create a VM if there is not enough free memory in the system.")
		}
	} else {
		return fmt.Sprintf("Hyper-V might fail to create a VM if there is not enough free memory in the system.")
	}

	return ""
}

func getHostAvailableMemory(driver hypervcommon.Driver) (string, error){

	cmdOut, err := driver.HypervManageOutput("(Get-WmiObject Win32_OperatingSystem).FreePhysicalMemory / 1024")
	
	if err != nil {
		return "", err
	}

  	var freeMemory = strings.TrimSpace(cmdOut)
	return freeMemory, err
	
	
}

func getExternalOnlineVirtualSwitch(driver hypervcommon.Driver) (string, error) {

  var script = `
	$adapters = Get-NetAdapter -Physical -ErrorAction SilentlyContinue | Where-Object { $_.Status -eq 'Up' } | Sort-Object -Descending -Property Speed
	foreach ($adapter in $adapters) { 
	  $switch = Get-VMSwitch -SwitchType External | Where-Object { $_.NetAdapterInterfaceDescription -eq $adapter.InterfaceDescription }
	
	  if ($switch -ne $null) {
	    $switch.Name
	    break
	  }
	}
	`	
	cmdOut, err := driver.HypervManageOutput(script)

	if err != nil {
		return "", err
	}

  	var switchName = strings.TrimSpace(cmdOut)
	return switchName, err
}