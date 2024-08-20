package foreman

import (
	"fmt"
	"os"
	"os/exec"
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
	}
	if err := foreman.parseProcfile(); err != nil {
		return nil, err
	}
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
		
		if checks, ok := info["checks"].(map[string]interface{}); ok {
			if cmd, ok := checks["cmd"].(string); ok {
				newInfo.checks.cmd = cmd
			}
			if tcpPorts, ok := checks["tcp_ports"].([]interface{}); ok {
				for _, ports := range tcpPorts {
					if port, ok := ports.(string); ok {
						newInfo.checks.tcpPorts = append(newInfo.checks.tcpPorts, port)
					}
				}
			}
			if udpPorts, ok := checks["udp_ports"].([]interface{}); ok {
				for _, ports := range udpPorts {
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
	for _, nodes := range topoGraph {
		for _, node := range nodes {//TODO concurrent
			err := foreman.runService(node)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (foreman *Foreman) runService(serviceName string) error{ 
	service := foreman.services[serviceName]
	serviceCmd := exec.Command("bash", "-c", service.info.cmd)
	serviceCmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
		Pgid: 0,
	}
	err := serviceCmd.Start()
	if err != nil {
		if !service.info.runOnce {
			return foreman.runService(serviceName)
		}
		return err
	}
	err = serviceCmd.Wait()
	if err != nil {
		if !service.info.runOnce {
			return foreman.runService(serviceName)
		}
		return err
	}
	service.id = serviceCmd.SysProcAttr.Pgid
	fmt.Printf("[%s] process started\n", service.name)
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