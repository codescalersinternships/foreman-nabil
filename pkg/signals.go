package foreman

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/process"
)

// signal initialize signal handling in foreman
func (foreman *Foreman) signal() {
	sigs := []os.Signal{syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGCHLD, syscall.SIGHUP}
	signal.Notify(foreman.signalsChannel, sigs...)
	go foreman.receiveSignals(foreman.signalsChannel)
}

// receiveSignals receive signals form sigChannel and calls a
// proper signal handler.
func (foreman *Foreman) receiveSignals(sigChannel <-chan os.Signal) {
	for sig := range sigChannel {
		switch sig {
		case syscall.SIGCHLD:
			foreman.sigchldHandler()
		default:
			foreman.killServicesAndExit()
		}
	}
}

// sigintHandler handles SIGINT signals
func (foreman *Foreman) killServicesAndExit() {
	for _, service := range foreman.services {
		fmt.Println(service.pid)
		_ = syscall.Kill(service.pid, syscall.SIGINT)
	}
	os.Exit(0)
}

// sigchldHandler handles SIGCHLD signals
func (foreman *Foreman) sigchldHandler() {
	for _, service := range foreman.services {
		childProcess, _ := process.NewProcess(int32(service.pid))
		childStatus, _ := childProcess.Status()
		if childStatus == "Z" {
			service.info.status = "inactive"
			p, _ := os.FindProcess(service.pid)
			_,_ = p.Wait()
			fmt.Printf("[%d] %s process terminated [%v]\n", service.pid, service.name, time.Now())
			if !service.info.runOnce {
				fmt.Printf("[%d] %s process restarted [%v]\n", service.pid, service.name, time.Now())
				foreman.restartService(service.name)
			}
		}
	}
}

// restartService restarts service by sending service to servicesToRunChannel
// to be run by a worker thread.
func (foreman *Foreman) restartService(service string) {
	foreman.servicesToRunChannel <- service
}
