package main

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go.wemo"
)

const driverName = "driver-wemo"
const SWITCH = "controllee"
const MOTION = "sensor"

var log = logger.GetLogger(driverName)

func main() {

	log.Infof("Starting " + driverName)

	conn, err := ninja.Connect("com.ninjablocks.wemo")
	if err != nil {
		log.HandleError(err, "Could not connect to MQTT")
	}

	pwd, _ := os.Getwd()

	bus, err := conn.AnnounceDriver("com.ninjablocks.wemo", driverName, pwd)
	if err != nil {
		log.HandleError(err, "Could not get driver bus")
	}

	statusJob, err := ninja.CreateStatusJob(conn, driverName)

	if err != nil {
		log.HandleError(err, "Could not setup status job")
	}

	statusJob.Start()

	ipAddr, err := ninja.GetNetAddress()
	if err != nil {
		log.HandleError(err, "Could not get net address")
	}

	/////////////////////NUKE ME////////////////////////////
	ipAddr = "10.0.1.150"
	////////////////////////////////////////////////////////

	log.Infof("Discovering new Wemos with interface %s", ipAddr)
	api := wemo.NewByIp(ipAddr)

	devices, _ := api.DiscoverAll(3 * time.Second) //TODO: this needs to be evented
	for _, device := range devices {
		deviceInfo, err := device.FetchDeviceInfo()
		if err != nil {
			log.HandleError(err, "Unable to fetch device info")
		}
		deviceStr := deviceInfo.DeviceType

		if isUnique(deviceInfo) {
			detectedSwitch, _ := regexp.MatchString(SWITCH, deviceStr)
			detectedMotion, _ := regexp.MatchString(MOTION, deviceStr)

			if detectedSwitch {
				log.Infof("Creating new switch")
				_, err = NewSwitch(bus, device, deviceInfo)
			} else if detectedMotion {
				log.Infof("Creating new motion detector")
				_, err = NewMotion(bus, device, deviceInfo)
			} else {
				log.Errorf("Unknown device type: %s", deviceStr)
			}
			spew.Dump(deviceInfo)
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// Block until a signal is received.
	s := <-c
	fmt.Println("Got signal:", s)

}
