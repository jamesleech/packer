// Copyright (c) Microsoft Open Technologies, Inc.
// All Rights Reserved.
// Licensed under the Apache License, Version 2.0.
// See License.txt in the project root for license information.
package iso

import (
	"fmt"
	"os"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"log"
	"io"
	"io/ioutil"
	"bytes"
	"path/filepath"
	hypervcommon "github.com/mitchellh/packer/builder/hyperv/common"
)

type StepMountFloppydrive struct {
	floppyPath string
}

func (s *StepMountFloppydrive) Run(state multistep.StateBag) multistep.StepAction {
	// Determine if we even have a floppy disk to attach
	var floppyPath string
	if floppyPathRaw, ok := state.GetOk("floppy_path"); ok {
		floppyPath = floppyPathRaw.(string)
	} else {
		log.Println("No floppy disk, not attaching.")
		return multistep.ActionContinue
	}

	// Hyper-V is really dumb and can't figure out the format of the file
	// without an extension, so we need to add the "vfd" extension to the
	// floppy.
	floppyPath, err := s.copyFloppy(floppyPath)
	if err != nil {
		state.Put("error", fmt.Errorf("Error preparing floppy: %s", err))
		return multistep.ActionHalt
	}	

	ui := state.Get("ui").(packer.Ui)
	ui.Say("Mounting floppy drive...")	
	vmName := state.Get("vmName").(string)
	driver := state.Get("driver").(hypervcommon.Driver)		

	err = mountFloppyDrive(driver, vmName, floppyPath)
	if err != nil {
		state.Put("error", fmt.Errorf("Error mounting floppy drive: %s", err))
		return multistep.ActionHalt
	}

	// Track the path so that we can unregister it from Hyper-V later
	s.floppyPath = floppyPath

	return multistep.ActionContinue}

func (s *StepMountFloppydrive) Cleanup(state multistep.StateBag) {
	if s.floppyPath == "" {
		return
	}

	errorMsg := "Error unmounting floppy drive: %s"

	vmName := state.Get("vmName").(string)
	ui := state.Get("ui").(packer.Ui)

	ui.Say("Unmounting floppy drive (cleanup)...")
	driver := state.Get("driver").(hypervcommon.Driver)	

	err := unmountFloppyDrive(driver, vmName)
	if err != nil {
		ui.Error(fmt.Sprintf(errorMsg, err))
	}

	err = os.Remove(s.floppyPath)

	if err != nil {
		ui.Error(fmt.Sprintf(errorMsg, err))
	}
}

func (s *StepMountFloppydrive) copyFloppy(path string) (string, error) {
	tempdir, err := ioutil.TempDir("", "packer")
	if err != nil {
		return "", err
	}

	floppyPath := filepath.Join(tempdir, "floppy.vfd")
	f, err := os.Create(floppyPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sourceF, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer sourceF.Close()

	log.Printf("Copying floppy to temp location: %s", floppyPath)
	if _, err := io.Copy(f, sourceF); err != nil {
		return "", err
	}

	return floppyPath, nil
}

func mountFloppyDrive(driver hypervcommon.Driver, vmName string, path string) error {

	var blockBuffer bytes.Buffer
	blockBuffer.WriteString("Set-VMFloppyDiskDrive -VMName ")
	blockBuffer.WriteString(vmName)
	blockBuffer.WriteString(" -Path ")
	blockBuffer.WriteString(path)

	err := driver.HypervManage( blockBuffer.String() )
	
	return err	
}

func unmountFloppyDrive(driver hypervcommon.Driver, vmName string) error {
	
	return mountFloppyDrive(driver, vmName, "$null") //unmount floppy by mounting $null
}

