package main

import (
	"fmt"

	foreman "github.com/codescalersinternships/foreman-nabil/pkg"
)


func main() {
	f, err := foreman.InitForeman()
	if err != nil {
		fmt.Println("err")
		fmt.Println(err)
		return
	}
	err = f.RunServices()
	if err != nil {

		fmt.Println("err")
		fmt.Println(err)
	}
	// serviceCmd := exec.Command("bash", "-c", "ping -c 1 google.com")
	// serviceCmd.SysProcAttr = &syscall.SysProcAttr{
	// 	Setpgid: true,
	// 	Pgid: 0,
	// }
	// serviceCmd.Start()
	// x := serviceCmd.Process.Pid
	// fmt.Printf("[%d] process started [%v]\n", x, time.Now())
}