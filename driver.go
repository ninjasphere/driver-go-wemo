package main

import (
	"fmt"
	"regexp"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja/api"
	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-ninja/devices"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go-ninja/model"
	"github.com/savaki/go.wemo"
	"golang.org/x/net/context"
)

const (
	driverName       = "com.ninjablocks.wemo"
	switchDesignator = "controllee"
	motionDesignator = "sensor"
)

var log = logger.GetLogger(driverName)
var info = ninja.LoadModuleInfo("./package.json")

type WemoDeviceContext struct {
	Info       *wemo.DeviceInfo
	Device     *wemo.Device
	deviceInfo *model.Device
	driver     ninja.Driver
}

func (w *WemoDeviceContext) GetDeviceInfo() *model.Device {
	return w.deviceInfo
}

func (w *WemoDeviceContext) GetDriver() ninja.Driver {
	return w.driver
}

func (w *WemoDeviceContext) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
}

type WemoDriver struct {
	conn      *ninja.Connection
	sendEvent func(event string, payload interface{}) error
}

func NewWemoDriver() (*WemoDriver, error) {
	conn, err := ninja.Connect(driverName)
	if err != nil {
		log.HandleError(err, "Could not connect to MQTT")
		return nil, err
	}

	driver := &WemoDriver{
		conn: conn,
	}

	err = conn.ExportDriver(driver)
	if err != nil {
		log.Fatalf("Failed to export Wemo driver: %s", err)
	}

	return driver, nil
}

func (d *WemoDriver) Start(x interface{}) error {
	log.Infof("Start method on Wemo driver called")

	return d.startDiscovery()
}

func (d *WemoDriver) startDiscovery() error {

	ipAddr, err := ninja.GetNetAddress()
	if err != nil {
		log.Fatalf("Could not get local address: %s", err)
	}

	log.Infof("Starting discovery of new Wemos with ip interface %s", ipAddr)
	api := wemo.NewByIp(ipAddr)
	//api.Debug = true

	seen := make(map[string]*WemoDeviceContext)

	go func() {
		for {

			devices, _ := api.DiscoverAll(5 * time.Second) //TODO: this needs to be evented

			ctx := context.Background()
			for _, device := range devices {

				deviceInfo, err := device.FetchDeviceInfo(ctx)
				if err != nil {
					log.HandleError(err, "Unable to fetch device info")
					continue
				}

				if existing, ok := seen[deviceInfo.SerialNumber]; ok {
					// We've already seen this device, update it's info
					existing.Info = deviceInfo
					existing.Device = device

				} else {

					deviceStr := deviceInfo.DeviceType

					detectedSwitch, _ := regexp.MatchString(switchDesignator, deviceStr)
					detectedMotion, _ := regexp.MatchString(motionDesignator, deviceStr)

					if detectedSwitch {
						log.Infof("Creating new switch")
						wemoDevice, err := d.NewSwitch(device, deviceInfo)
						if err != nil {
							log.Warningf("Failed to create switch: %s", err)
							continue
						}
						seen[deviceInfo.SerialNumber] = wemoDevice
					} else if detectedMotion {
						log.Infof("Creating new motion sensor")
						wemoDevice, err := d.NewMotion(d, d.conn, device, deviceInfo)
						if err != nil {
							log.Warningf("Failed to create motion sensor: %s", err)
							continue
						}
						seen[deviceInfo.SerialNumber] = wemoDevice
					} else {
						log.Errorf("Unknown device type: %s", deviceStr)
						spew.Dump(deviceInfo)
					}
				}

			}

		}
	}()

	return nil
}

func (d *WemoDriver) NewMotion(driver ninja.Driver, conn *ninja.Connection, device *wemo.Device, info *wemo.DeviceInfo) (*WemoDeviceContext, error) {
	sigs := map[string]string{
		"ninja:thingType":    "motion",
		"ninja:manufacturer": "Belkin",
	}

	ws := &WemoDeviceContext{
		Info:   info,
		Device: device,
		driver: driver,
		deviceInfo: &model.Device{
			NaturalID:     info.MacAddress,
			Name:          &info.FriendlyName,
			NaturalIDType: "wemo-mac",
			Signatures:    &sigs,
		},
	}

	err := conn.ExportDevice(ws)
	if err != nil {
		return nil, err
	}

	channel := channels.NewMotionChannel()

	err = conn.ExportChannel(ws, channel, "motion")
	if err != nil {
		return nil, err
	}

	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for _ = range ticker.C {
			curState := ws.Device.GetBinaryState()
			if curState != 0 {
				channel.SendMotion()
			}
		}
	}()

	return ws, nil
}

func (d *WemoDriver) NewSwitch(device *wemo.Device, info *wemo.DeviceInfo) (*WemoDeviceContext, error) {
	sigs := map[string]string{
		"ninja:thingType":    "socket",
		"ninja:manufacturer": "Belkin",
	}

	deviceInfo := &model.Device{
		NaturalID:     info.MacAddress,
		Name:          &info.FriendlyName,
		NaturalIDType: info.DeviceType,
		Signatures:    &sigs,
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

	return ws, nil
}

func (d *WemoDriver) GetModuleInfo() *model.Module {
	return info
}

func (d *WemoDriver) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	d.sendEvent = sendEvent
}
