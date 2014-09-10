package main

import (
	"fmt"
	"time"

	"github.com/savaki/go.wemo"
)

func main() {
	// you can either create a device directly OR use the
	// #Discover/#DiscoverAll methods to find devices
	api, _ := wemo.NewByInterface("en0")
	devices, _ := api.DiscoverAll(3 * time.Second)
	for {
		for _, device := range devices {
			// device        := &wemo.Device{Host:"10.0.1.32:49153"}

			// retrieve device info
			deviceInfo, _ := device.FetchDeviceInfo()
			fmt.Printf("Found => %+v\n", deviceInfo)

			// device controls
			device.On()
			time.Sleep(1 * time.Second)
			device.Off()
			time.Sleep(1 * time.Second)
		}
	}
}
