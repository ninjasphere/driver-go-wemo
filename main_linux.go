package main

import (
	"time"

	"github.com/ninjasphere/go-ninja/logger"
)

const driverName = "driver-wemo"

var log = logger.GetLogger(driverName)

func main() {

	log.Infof("Starting up NO-OP wemo driver.\n" + driverName)

	// see real driver contents in main_wemo.
	time.Sleep(86400*time.Second);
}
