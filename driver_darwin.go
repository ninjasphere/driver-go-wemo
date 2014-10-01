package main

import (
	"fmt"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/ninjasphere/go.wemo"

	"github.com/ninjasphere/go-ninja"
	"github.com/ninjasphere/go-ninja/devices"
)

var seenDevices []string //Store serial numbers of all seen switches

type WemoDeviceContext struct {
	Info   *wemo.DeviceInfo
	Device *wemo.Device
}

func NewMotion(bus *ninja.DriverBus, device *wemo.Device, info *wemo.DeviceInfo) (*WemoDeviceContext, error) {

	sigs, _ := simplejson.NewJson([]byte(`{
			"ninja:manufacturer": "Belkin",
			"ninja:productName": "Wemo",
			"manufacturer:productModelId": "",
			"ninja:productType": "Motion",
			"ninja:thingType": "motion"
	}`))

	sigs.Set("manufacturer:productModelId", info.DeviceType)
	deviceBus, err := bus.AnnounceDevice(info.SerialNumber, "wemo", info.FriendlyName, sigs)
	_ = deviceBus // FIXME
	if err != nil {
		log.FatalError(err, "Failed to create light device bus ")
	}

	//FIXME:
	//wemoMotion, err := devices.CreateMotionDevice(info.SerialNumber, deviceBus)
	err = fmt.Errorf("API change - please edit FIXME in driver.go")

	if err != nil {
		log.FatalError(err, "Failed to create motion device")
	}

	// FIXME:
	//if wemoMotion.EnableMotionChannel(); err != nil {
	//	log.FatalError(err, "Could not enable wemo motion motion channel")
	//}

	ticker := time.NewTicker(time.Second * 2) //TODO: this needs to be nicer for motion since this data is much more time sensitive.
	go func() {
		for _ = range ticker.C {
			// curState := device.GetBinaryState()
			// boolCurState := curState != 0
			// log.Infof("Got state %t", boolCurState)
			// wemoMotion.UpdateMotionState(boolCurState) //curstate needs bool, but get state returns int
		}
	}()

	ws := &WemoDeviceContext{
		Info:   info,
		Device: device,
	}

	return ws, err
}

func NewSwitch(bus *ninja.DriverBus, device *wemo.Device, info *wemo.DeviceInfo) (*WemoDeviceContext, error) {

	sigs, _ := simplejson.NewJson([]byte(`{
      "ninja:manufacturer": "Belkin",
      "ninja:productName": "Wemo",
      "manufacturer:productModelId": "",
      "ninja:productType": "Switch",
      "ninja:thingType": "unknown"
  }`))
	sigs.Set("manufacturer:productModelId", info.DeviceType)

	deviceBus, err := bus.AnnounceDevice(info.SerialNumber, "wemo", info.FriendlyName, sigs)

	if err != nil {
		log.FatalError(err, "Failed to create wemo switch device bus ")
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

	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for _ = range ticker.C {
			curState := device.GetBinaryState()
			wemoSwitch.UpdateSwitchOnOffState(curState != 0) //curstate needs bool, but get state returns int
		}
	}()

	ws := &WemoDeviceContext{
		Info:   info,
		Device: device,
	}

	return ws, err
}

func isUnique(newDevice *wemo.DeviceInfo) bool {
	ret := true
	for _, s := range seenDevices {
		if newDevice.SerialNumber == s {
			ret = false
		}
	}
	return ret
}
