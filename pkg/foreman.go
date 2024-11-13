package foreman

import "os"



type Check struct {
	cmd string
	tcpPorts []string
	udpPorts []string
}

type ServiceInfo struct {
	cmd string
	runOnce bool
	checks Check
	deps []string
	status string
}

type Service struct {
	name string
	pid int
	info ServiceInfo
}

type Foreman struct {
	procfile string
	services map[string]Service
	servicesGraph map[string][]string
	signalsChannel chan os.Signal
	servicesToRunChannel chan string
}


