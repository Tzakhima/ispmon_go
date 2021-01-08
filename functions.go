package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"
)

type parameters struct {
	Ping     []string `json:"ping_target"`
	HTTP     []string `json:"http_target"`
	Interval int      `json:"speed_interval"`
}

type ipInfoStruct struct {
	City    string  `json:"city"`
    Country string  `json:"country"`
	ISP     string  `json:"org"`
}


func getMacAddr() ([]string, error) {
	ifas, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var as []string
	for _, ifa := range ifas {
		a := ifa.HardwareAddr.String()
		if a != "" {
			as = append(as, a)
		}
	}
	return as, nil
}

func getParameters() ([]string, []string, int, error) {

	apiClient := http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest(http.MethodGet, ParametersUrl, nil)
	if err != nil {
		return nil, nil, -1, fmt.Errorf("could not build HTTP request: %g", err)
	}

	res, getErr := apiClient.Do(req)

	if getErr != nil {
		return nil, nil, -1, fmt.Errorf("could not get HTTP request: %g", getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return nil, nil, -1, fmt.Errorf("reading response body error: %g", readErr)
	}

	response := parameters{}
	jsonErr := json.Unmarshal(body, &response)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	fmt.Println(response.Ping)

	return response.Ping, response.HTTP, response.Interval, nil

}

func getIspInfo() (map[string]string, error) {
	infoClient := http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest(http.MethodGet, IpInfoUrl, nil)
	if err != nil {
		log.Fatal("Setting Req Error: ",err)
	}

	res, getErr := infoClient.Do(req)
	if getErr != nil {
		log.Fatal("Do req Error: ",getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal("Read Error: ",readErr)
	}

	respInfo := ipInfoStruct{}
	jsonErr  := json.Unmarshal(body, &respInfo)
	if jsonErr != nil {
		log.Fatal("JSON Unmarshal Error: ",jsonErr)
	}

	returnInfo := make(map[string]string)
	returnInfo["country"] = respInfo.Country
	returnInfo["city"]    = respInfo.City
	returnInfo["isp"]     = respInfo.ISP

	return returnInfo, nil

}

func httpStat(url string, c chan map[string]string) {

	req, _ := http.NewRequest("GET", url, nil)

	var start, connect, dns, tlsHandshake time.Time

	results := make(map[string]string)

	results["url"] = url

	trace := &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) { dns = time.Now() },
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			results["dnsTime"] = time.Since(dns).String()
		},

		TLSHandshakeStart: func() { tlsHandshake = time.Now() },
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			results["tlsTime"] = time.Since(tlsHandshake).String()
		},

		ConnectStart: func(network, addr string) { connect = time.Now() },
		ConnectDone: func(network, addr string, err error) {
			results["connTime"] = time.Since(connect).String()
		},

		GotFirstResponseByte: func() {
			results["ttfbTime"] = time.Since(start).String()
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			results["connReuse"] = fmt.Sprint(connInfo.Reused)
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()
	if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
		log.Fatal(err)
	}

	c <- results
}
