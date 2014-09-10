#Ninja Sphere Go Belkin Wemo Driver

##Building
Run `make` in the directory of the driver

or to develop on mac and run on the sphere
`GOOS=linux GOARCH=arm go build -o driver-go-wemo main.go driver.go version.go && scp driver-go-wemo ninja@ninjasphere.local:~/

##Running
Run `./bin/driver-go-wemo` from the `bin` directory after building
