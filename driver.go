package main

import (
	"fmt"

	"github.com/bitly/go-simplejson"
	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go.wemo"

	"github.com/ninjasphere/go-ninja"
	"github.com/ninjasphere/go-ninja/devices"
)

var seenSwitches []string //Store serial numbers of all seen switches

type WemoSwitchContext struct {
	Info   *wemo.DeviceInfo
	Switch *wemo.Device
}

func NewSwitch(bus *ninja.DriverBus, device *wemo.Device, info *wemo.DeviceInfo) (*WemoSwitchContext, error) {

	log.Infof("Making Wemo switch with device info: ")
	spew.Dump(info)

	sigs, _ := simplejson.NewJson([]byte(`{
      "ninja:manufacturer": "Belkin",
      "ninja:productName": "Wemo",
      "manufacturer:productModelId": "",
      "ninja:productType": "Switch",
      "ninja:thingType": "switch"
  }`))
	sigs.Set("manufacturer:productModelId", info.DeviceType)

	deviceBus, err := bus.AnnounceDevice(info.SerialNumber, "wemo", info.FriendlyName, sigs)

	if err != nil {
		log.FatalError(err, "Failed to create light device bus ")
	}

	// func CreateSwitchDevice(name string, bus *ninja.DeviceBus) (*SwitchDevice, error) {

	wemoSwitch, err := devices.CreateSwitchDevice(info.SerialNumber, deviceBus)
	if err != nil {
		log.FatalError(err, "Failed to create switch device")
	}

	if err := wemoSwitch.EnableOnOffChannel(); err != nil {
		log.FatalError(err, "Could not enable wemo switch on-off channel")
	}

	wemoSwitch.ApplyOnOff = func(state bool) error {
		var err error
		if state {
			device.On()
		} else {
			device.Off()
		}
		if err != nil {
			return fmt.Errorf("Failed to set on-off state: %s", err)
		}
		return nil
	}

	ws := &WemoSwitchContext{
		Info:   info,
		Switch: device,
	}

	return ws, nil
}

func isUnique(newSwitch *wemo.DeviceInfo) bool {
	ret := true
	for _, s := range seenSwitches {
		if newSwitch.SerialNumber == s {
			ret = false
		}
	}
	return ret
}
