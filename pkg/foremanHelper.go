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
		return nil, nil
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
			cmd: info["cmd"].(string),
			runOnce: info["run_once"].(bool),
			checks: Check{
				cmd: info["checks"].(map[string]any)["cmd"].(string),
				tcpPorts: []string{},
				udpPorts: []string{},
			},
			deps: []string{},
		}
		newInfo.deps = append(newInfo.deps, info["deps"].([]string)...)
		newInfo.checks.tcpPorts = append(newInfo.checks.tcpPorts, info["checks"].(map[string]any)["tcp_ports"].([]string)...)
		newInfo.checks.udpPorts = append(newInfo.checks.udpPorts, info["checks"].(map[string]any)["udp_ports"].([]string)...)

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
	service.id = serviceCmd.SysProcAttr.Pgid
	fmt.Printf("[%s] process started\n", service.name)
	foreman.services[serviceName] = service
	return nil
}

func dfs(node *string, graph map[string][]string, que *[]string, vis map[string]bool) (bool){
	if vis[*node] {
		return true
	}
	vis[*node] = false
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
	return topoGraph, false

}