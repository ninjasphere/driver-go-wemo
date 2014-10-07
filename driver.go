package main

import (
	"fmt"
	"regexp"
	"time"
	"github.com/ninjasphere/go-ninja/devices"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/ninjasphere/go.wemo"
	"github.com/davecgh/go-spew/spew"
)

const (
	driverName       = "com.ninjablocks.wemo"
	switchDesignator = "controllee"
	motionDesignator = "sensor"
)

var log = logger.GetLogger(driverName)
var info = ninja.LoadModuleInfo("./package.json")
var seenDevices []string //Store serial numbers of all seen switches

type WemoDeviceContext struct {
	Info   *wemo.DeviceInfo
	Device *wemo.Device
}

type WemoDriver struct {
	config    *WemoDriverConfig
	conn      *ninja.Connection
	sendEvent func(event string, payload interface{}) error
	devices   *[]WemoDeviceContext
}

type WemoDriverConfig struct {
	NumberOfDevices int
}

func defaultConfig() *WemoDriverConfig {
	return &WemoDriverConfig{
		NumberOfDevices: 0,
	}
}

func NewWemoDriver() (*WemoDriver, error) {
	conn, err := ninja.Connect(driverName)
	if err != nil {
		log.HandleError(err, "Could not connect to MQTT")
		return nil,err
	}


	driver := &WemoDriver{
		conn:      conn,
		config:    defaultConfig(),
		sendEvent: nil,
		devices:   nil,
	}

	log.Infof("1");
	err = conn.ExportDriver(driver)
	log.Infof("2");
	if err != nil {
		log.Fatalf("Failed to export Wemo driver: %s", err)
	}

	return driver, nil
}

func (d *WemoDriver) Start(config *WemoDriverConfig) error {
	log.Infof("Start method on Wemo driver called")
	ipAddr, err := ninja.GetNetAddress()
	if err != nil {
		log.HandleError(err, "Could not get net address")
		return err
	}

	log.Infof("Discovering new Wemos with ip interface %s", ipAddr)
	api := wemo.NewByIp(ipAddr)

	devices, _ := api.DiscoverAll(3 * time.Second) //TODO: this needs to be evented
	for _, device := range devices {
		deviceInfo, err := device.FetchDeviceInfo()
		if err != nil {
			log.HandleError(err, "Unable to fetch device info")
			return err
		}
		deviceStr := deviceInfo.DeviceType

		if isUnique(deviceInfo) {
			detectedSwitch, _ := regexp.MatchString(switchDesignator, deviceStr)
			detectedMotion, _ := regexp.MatchString(motionDesignator, deviceStr)

			if detectedSwitch {
				log.Infof("Creating new switch")
				_, err = d.NewSwitch(device, deviceInfo)
			} else if detectedMotion {
				log.Infof("Creating new motion detector")
				//TODO
				// _, err = d.NewMotion(device, deviceInfo)
			} else {
				log.Errorf("Unknown device type: %s", deviceStr)
				spew.Dump(deviceInfo)
			}

		}
	}
	return nil
}

func (d *WemoDriver) NewSwitch(device *wemo.Device, info *wemo.DeviceInfo) (*WemoDeviceContext, error) {
	deviceInfo := &model.Device{
			NaturalID: info.MacAddress,
			Name: &info.FriendlyName,
			NaturalIDType: info.DeviceType,
	}

	wemoSwitch, err := devices.CreateSwitchDevice(d, deviceInfo, d.conn)
	if err != nil {
		log.FatalError(err, "Failed to create switch device")
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
			wemoSwitch.UpdateSwitchState(curState != 0) //curstate needs bool, but get state returns int
		}
	}()

	ws := &WemoDeviceContext{
		Info:   info,
		Device: device,
	}

	return ws, err
}

//TODO : make motion with new APi
// func (d *WemoDriver) NewMotion(device *wemo.Device, info *wemo.DeviceInfo) (*WemoDeviceContext, error) {
//
// }

func isUnique(newDevice *wemo.DeviceInfo) bool {
	ret := true
	for _, s := range seenDevices {
		if newDevice.SerialNumber == s {
			ret = false
		}
	}
	return ret
}

func (d *WemoDriver) GetModuleInfo() *model.Module {
	return info
}

func (d *WemoDriver) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	d.sendEvent = sendEvent
}
