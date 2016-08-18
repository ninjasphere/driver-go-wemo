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
	"github.com/ninjasphere/go.wemo"
	"golang.org/x/net/context"
	"strings"
)

const (
	switchDesignator  = "controllee"
	insightDesignator = "insight"
	motionDesignator  = "sensor"
)

var info = ninja.LoadModuleInfo("./package.json")
var log = logger.GetLogger(info.ID)

type WemoDeviceContext struct {
	devices.BaseDevice
	Info   *wemo.DeviceInfo
	Device *wemo.Device
}

type WemoDriver struct {
	conn      *ninja.Connection
	sendEvent func(event string, payload interface{}) error
}

func NewWemoDriver() (*WemoDriver, error) {
	conn, err := ninja.Connect(info.ID)
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
					// We've already seen this device, update its info
					existing.Info = deviceInfo
					existing.Device = device

				} else {

					deviceStr := strings.ToLower(deviceInfo.DeviceType)

					detectedSwitch, _ := regexp.MatchString(switchDesignator, deviceStr)
					detectedInsight, _ := regexp.MatchString(insightDesignator, deviceStr)
					detectedMotion, _ := regexp.MatchString(motionDesignator, deviceStr)

					if (detectedSwitch || detectedInsight) && detectedMotion {
						log.Errorf("contradictory device type: %s", deviceStr)
						spew.Dump(deviceInfo)
						continue
					}

					if detectedSwitch || detectedInsight || detectedMotion {
						log.Infof("Creating new device (%v, %v, %v)", detectedSwitch, detectedInsight, detectedMotion)
						wemoDevice, err := d.NewSwitch(d, d.conn, device, deviceInfo, detectedSwitch || detectedInsight, detectedInsight, detectedMotion)
						if err != nil {
							log.Warningf("Failed to create (front-end) device: %s", err)
							continue
						}
						seen[deviceInfo.SerialNumber] = wemoDevice
					}

					if !detectedSwitch && !detectedInsight && !detectedMotion {
						log.Errorf("Unknown device type: %s", deviceStr)
						spew.Dump(deviceInfo)
					}
				}

			}

		}
	}()

	return nil
}

func (wsd *WemoDeviceContext) SetOnOff(state bool) error {
	var err error
	if state {
		wsd.Device.On()
	} else {
		wsd.Device.Off()
	}
	if err != nil {
		return fmt.Errorf("Failed to set on-off state: %s", err)
	}
	return nil
}

func (wsd *WemoDeviceContext) ToggleOnOff() error {
	curState := wsd.Device.GetBinaryState()
	if curState != 0 {
		return wsd.SetOnOff(false)
	} else {
		return wsd.SetOnOff(true)
	}
}

func (d *WemoDriver) NewSwitch(driver ninja.Driver, conn *ninja.Connection, device *wemo.Device, info *wemo.DeviceInfo, hasSwitch bool, hasPower bool, hasMotion bool) (*WemoDeviceContext, error) {
	sigs := map[string]string{
		"ninja:thingType":    "socket",
		"ninja:manufacturer": "Belkin",
	}

	ws := &WemoDeviceContext{
		BaseDevice: devices.BaseDevice{
			Driver: driver,
			Info: &model.Device{
				NaturalID:     info.MacAddress,
				Name:          &info.FriendlyName,
				NaturalIDType: info.DeviceType,
				Signatures:    &sigs,
			},
			Conn: conn,
			Log_: log,
		},
		Info:   info,
		Device: device,
	}

	if err := conn.ExportDevice(ws); err != nil {
		log.Fatalf("failed to export device: %v", err)
	}

	var onOffChannel *channels.OnOffChannel
	var powerChannel *channels.PowerChannel
	var motionChannel *channels.MotionChannel

	onOffChannel = channels.NewOnOffChannel(ws)
	err := conn.ExportChannel(ws, onOffChannel, "on-off")
	if err != nil {
		log.Fatalf("failed to export on-off channel: %v", err)
	}

	if hasMotion {
		motionChannel = channels.NewMotionChannel()
		err = conn.ExportChannel(ws, motionChannel, "motion")
		if err != nil {
			return nil, err
		}
	}

	if hasPower {
		powerChannel = channels.NewPowerChannel(d)
		if err != nil {
			log.Fatalf("failed to export power channel: %v", err)
		}
	}

	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for _ = range ticker.C {
			curState := device.GetBinaryState()
			onOffChannel.SendState(curState != 0) //curstate needs bool, but get state returns int
			if powerChannel != nil {
				insightState := device.GetInsightParams()
				powerChannel.SendState(float64(insightState.Power) / 1000.0) //curstate needs bool, but get state returns int
			}

			if motionChannel != nil {
				curState := ws.Device.GetBinaryState()
				if curState != 0 {
					motionChannel.SendMotion()
				}
			}
		}
	}()

	return ws, nil
}

func (d *WemoDriver) GetModuleInfo() *model.Module {
	return info
}

func (d *WemoDriver) SetEventHandler(sendEvent func(event string, payload interface{}) error) {
	d.sendEvent = sendEvent
}
