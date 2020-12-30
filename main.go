package main

import (
	"fmt"
)

func main() {

	//Get PING, HTTP and speed test parameters
	pingLinks, httpLinks, interval := getParameters()
	schema := "https://"

	fmt.Println(pingLinks, httpLinks, interval)

	c := make(chan map[string]string)

	for _, link := range httpLinks {
		go httpStat(schema+link, c)
	}

	for i := 0; i < len(pingLinks); i++ {
		fmt.Println(<-c)
	}

}
