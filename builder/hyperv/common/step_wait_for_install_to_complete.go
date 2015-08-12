package common

//type StepWaitForInstallToComplete struct {
//	ExpectedRebootCount uint
//	ActionName string
//}

//func (s *StepWaitForInstallToComplete) Run(state multistep.StateBag) multistep.StepAction {
//	ui := state.Get("ui").(packer.Ui)
//	vmName := state.Get("vmName").(string)

//	if(len(s.ActionName)>0){
//		ui.Say(fmt.Sprintf("%v ! Waiting for VM to reboot %v times...",s.ActionName, s.ExpectedRebootCount))
//	}

//	var rebootCount uint
//	var lastUptime uint64


//	script.WriteLine("(Get-VM -Name $vmName).Uptime.TotalSeconds")

//	uptimeScript := script.String()

//	for rebootCount < s.ExpectedRebootCount {
//		powershell := new(powershell.PowerShellCmd)
//		cmdOut, err := powershell.Output(uptimeScript, vmName);
//		if err != nil {
//			err := fmt.Errorf("Error checking uptime: %s", err)
//			state.Put("error", err)
//			ui.Error(err.Error())
//			return multistep.ActionHalt
//		}

//		uptime, _ := strconv.ParseUint(strings.TrimSpace(string(cmdOut)), 10, 64)
//		if uint64(uptime) < lastUptime {
//			rebootCount++
//			ui.Say(fmt.Sprintf("%v  -> Detected reboot %v after %v seconds...", s.ActionName, rebootCount, lastUptime))
//		}

//		lastUptime = uptime

//		if (rebootCount < s.ExpectedRebootCount) {
//			time.Sleep(time.Second * SleepSeconds);
//		}
//	}


//	return multistep.ActionContinue
//}

//func (s *StepWaitForInstallToComplete) Cleanup(state multistep.StateBag) {

//}


