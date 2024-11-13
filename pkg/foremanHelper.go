package foreman

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"

	"gopkg.in/yaml.v2"
)


func InitForeman(args ...string) (*Foreman, error) {
	procFile := "procfile.yaml"
	if len(args) > 0{
		procFile = args[0]
	}
	foreman := Foreman {
		procfile: procFile,
		services: map[string]Service{},
		servicesGraph: map[string][]string{},
		signalsChannel: make(chan os.Signal, 1e6),
		servicesToRunChannel: make(chan string, 1e6),
	}
	if err := foreman.parseProcfile(); err != nil {
		return nil, err
	}
	foreman.signal()
	return &foreman, nil
}

func (foreman *Foreman)parseProcfile () error {
	yamlMap := make(map[string]map[string]interface{})

    data, err := os.ReadFile(foreman.procfile)
	if err != nil {
		return err
	}
    err = yaml.Unmarshal([]byte(data), yamlMap)

	if err != nil {
		return err
	}
	for service, info := range yamlMap {
		newInfo := ServiceInfo{
			checks: Check{
				tcpPorts: []string{},
				udpPorts: []string{},
			},
			deps: []string{},
		}
		if cmd, ok := info["cmd"].(string); ok {
			newInfo.cmd = cmd
		}
		
		if runOnce, ok := info["run_once"].(bool); ok {
			newInfo.runOnce = runOnce
		}
		
		if checks, ok := info["checks"].(map[interface{}]interface{}); ok {
			if cmd, ok := checks["cmd"].(string); ok {
				newInfo.checks.cmd = cmd
			}
			if tcpPorts, ok := checks["tcp_ports"].([]interface{}); ok {
				for _, ports := range tcpPorts {
					if port, ok := ports.(int); ok {
						newInfo.checks.tcpPorts = append(newInfo.checks.tcpPorts, strconv.Itoa(port))
						continue
					}
					if port, ok := ports.(string); ok {
						newInfo.checks.tcpPorts = append(newInfo.checks.tcpPorts, port)
					}
				}
			}
			if udpPorts, ok := checks["udp_ports"].([]interface{}); ok {
				for _, ports := range udpPorts {
					if port, ok := ports.(int); ok {
						newInfo.checks.udpPorts = append(newInfo.checks.udpPorts, strconv.Itoa(port))
						continue
					}
					if port, ok := ports.(string); ok {
						newInfo.checks.udpPorts = append(newInfo.checks.udpPorts, port)
					}
				}
			}
		}
		
		if deps, ok := info["deps"].([]interface{}); ok {
			for _, depInterface := range deps {
				if dep, ok := depInterface.(string); ok {
					newInfo.deps = append(newInfo.deps, dep)
				}
			}
		}
		

		foreman.services[service] = Service{name: service, info: newInfo}
	}

	for _, service := range foreman.services {
		foreman.servicesGraph[service.name] = append(foreman.servicesGraph[service.name], service.info.deps...)
	}

	return nil
}

func (foreman *Foreman)RunServices() (error){
	topoGraph, isCyc := topologicalSort(foreman.servicesGraph, foreman.services)
	if isCyc {
		return fmt.Errorf("dependacies form cycle route from parent %v", topoGraph[len(topoGraph) -1])
	}
	var wg sync.WaitGroup
	for _, nodes := range topoGraph {
		var conErr error
		wg.Add(1)
		go func(nodes []string){
			defer wg.Done()
			for _, node := range nodes {
				err := foreman.runService(node)
				if err != nil {
					conErr = err
				}
			}
		}(nodes)
		if conErr != nil {
			return conErr
		}
	}
	wg.Wait()

	foreman.createServiceRunners(foreman.servicesToRunChannel, 5)
	return nil
}

func (foreman *Foreman) createServiceRunners(services <-chan string, numWorkers int) {
	for w := 0; w < numWorkers; w++ {
		go foreman.serviceRunner(services)
	}
}

// serviceRunner is the worker, of which we’ll run several concurrent instances.
func (foreman *Foreman) serviceRunner(services <-chan string) {
	for serviceName := range services {
		foreman.runService(serviceName)
	}
}

func (foreman *Foreman) serviceDepsAreAllActive(service Service) (bool, string) {
	for _, dep := range service.info.deps {
		if foreman.services[dep].info.status == "inactive" {
			foreman.restartService(dep)
			return false, dep
		} 
	}
	return true, ""
}

func (foreman *Foreman) runService(serviceName string) error{ 
	service := foreman.services[serviceName]
	if ok, _ := foreman.serviceDepsAreAllActive(service); !ok {
		foreman.restartService(serviceName)
		return nil
	}
	serviceCmd := exec.Command("bash", "-c", service.info.cmd)
	serviceCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid: 0,
	}
	startErr := serviceCmd.Start()
	err := serviceCmd.Wait()
	if startErr != nil {
		if !service.info.runOnce {
			return foreman.runService(serviceName)
		}
		return startErr
	}
	if err != nil {
		if !service.info.runOnce {
			return foreman.runService(serviceName)
		}
		return err
	}
	service.pid = serviceCmd.Process.Pid
	service.info.status = "active"
	fmt.Printf("[%d] process [%s] started\n", service.pid, service.name)
	foreman.services[serviceName] = service
	return nil
}

func dfs(node *string, graph map[string][]string, que *[]string, vis map[string]bool) (bool){
	if vis[*node] {
		return true
	}
	vis[*node] = true
	isCyc := false
	for _, child := range graph[*node] {
		isCyc = isCyc || dfs(&child, graph, que, vis)
	}
	*que = append(*que, *node)
	return isCyc
}

func topologicalSort(graph map[string][]string, services map[string]Service) ([][]string,bool) {
	vis := make(map[string]bool)
	in := make(map[string]int)

	for _, deps := range graph {
		for _, dep := range deps {
			in[dep]++;
		}
	}
	startingNodes := []string{}
	for service := range services {
		if in[service] == 0 {
			startingNodes = append(startingNodes, service)
		}
	}
	topoGraph := [][]string{}
	isCyc := false
	for _, node := range startingNodes {
		topoGraph = append(topoGraph, []string{})
		isCyc = isCyc || dfs(&node,graph,&topoGraph[len(topoGraph) - 1],vis)
		if isCyc {
			return topoGraph, true
		}
	}
	for service := range graph {
		if !vis[service] {
			topoGraph = append(topoGraph, []string{})
			isCyc = isCyc || dfs(&service,graph,&topoGraph[len(topoGraph) - 1],vis)
			if isCyc {
				return topoGraph, true
			}
		}
	}
	return topoGraph, false

}