// Copyright (c) Microsoft Open Technologies, Inc.
// All Rights Reserved.
// Licensed under the Apache License, Version 2.0.
// See License.txt in the project root for license information.
package common

import (
	"fmt"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"bytes"
	"time"
)

const (
	SleepSeconds = 10
)

type StepWaitForPowerOff struct {
}

func (s *StepWaitForPowerOff) Run(state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	driver := state.Get("driver").(Driver)
	vmName := state.Get("vmName").(string)
	ui.Say("Waiting for vm to be powered down...")

	// unless the person has a super fast disk, it should take at least 5 minutes
	// for the install and post-install operations to take. Wait 5 minutes to 
	// avoid hammering on getting VM status via PowerShell
	time.Sleep(time.Second * 300);
	
	var blockBuffer bytes.Buffer
	blockBuffer.WriteString("(Get-VM -Name ")
	blockBuffer.WriteString(vmName)
	blockBuffer.WriteString(").State -eq [Microsoft.HyperV.PowerShell.VMState]::Off")

	for {
		cmdOut, err := driver.HypervManageOutput(blockBuffer.String())
		if err != nil {
			err := fmt.Errorf("Error checking VM's state: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		if cmdOut == "True" {
			break
		} else {
			time.Sleep(time.Second * SleepSeconds)
		}
	}

	return multistep.ActionContinue
}

func (s *StepWaitForPowerOff) Cleanup(state multistep.StateBag) {
}

