package main

import (
	proxy "github.com/saas/hostgolang-proxy"
	"log"
)

func main() {
	addr := ":9093"
	if err := proxy.Run(addr); err != nil {
		log.Fatal(err)
	}
}