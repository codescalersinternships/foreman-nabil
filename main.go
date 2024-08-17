package main

import foreman "github.com/codescalersinternships/foreman-nabil/pkg"


func main() {
	f, _ := foreman.InitForeman()
	_ = f.RunServices()
	
}