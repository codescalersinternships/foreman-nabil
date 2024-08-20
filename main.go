package main

import (
	"fmt"

	foreman "github.com/codescalersinternships/foreman-nabil/pkg"
)


func main() {
	f, err := foreman.InitForeman()
	if err != nil {
		fmt.Println(err)
		return
	}
	err = f.RunServices()
	if err != nil {
		fmt.Println(err)
	}
}