package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/ninjasphere/go-ninja"
	"github.com/ninjasphere/go-ninja/logger"
	"github.com/ninjasphere/go.wemo"
)

const driverName = "driver-wemo"

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

	api := wemo.NewByIp(ipAddr)
	devices, _ := api.DiscoverAll(3 * time.Second) //TODO: this needs to be evented
	for _, device := range devices {
		deviceInfo, err := device.FetchDeviceInfo()
		log.HandleError(err, "Unable to fetch device info")
		fmt.Printf("Found => %+v\n", deviceInfo)
		_, err = NewSwitch(bus, device, deviceInfo)

	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// Block until a signal is received.
	s := <-c
	fmt.Println("Got signal:", s)

}
