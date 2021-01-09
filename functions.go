package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/ddo/go-fast"
	"github.com/go-ping/ping"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"
)

// Init global types
type parameters struct {
	Ping		[]string `json:"ping_target"`
	HTTP      	[]string `json:"http_target"`
	Interval  	int      `json:"speed_interval"`
}

type ipInfoStruct struct {
	City		string  `json:"city"`
    Country		string  `json:"country"`
	ISP			string  `json:"org"`
}


// Functions Start
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

func getHttpStat(url string, c chan map[string]map[string]int64) {

	schema := "https://"

	req, _ := http.NewRequest("GET", schema+url, nil)

	var start, connect, dns, tlsHandshake time.Time

    results := make(map[string]map[string]int64)
	results[url] = make(map[string]int64)


	trace := &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) { dns = time.Now() },
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			results[url]["dnsTime"] = int64(time.Since(dns) / time.Millisecond)
		},

		TLSHandshakeStart: func() { tlsHandshake = time.Now() },
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			results[url]["tlsTime"] = int64(time.Since(tlsHandshake) / time.Millisecond)
		},

		ConnectStart: func(network, addr string) { connect = time.Now() },
		ConnectDone: func(network, addr string, err error) {
			results[url]["connTime"] =int64(time.Since(connect) / time.Millisecond)
		},

		GotFirstResponseByte: func() {
			results[url]["ttfbTime"] = int64(time.Since(start) / time.Millisecond)
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()
	if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
		log.Fatal(err)
	}

	c <- results
}

func getPingStat(target string, wg *sync.WaitGroup) map[string]map[string]float64 {
	defer wg.Done()

	result := make(map[string]map[string]float64)
	result[target] = make(map[string]float64)

	pinger, err := ping.NewPinger(target)
	if err != nil {
		result[target]["packetLoss"] = 0
		result[target]["minRTT"]     = 0
		result[target]["avgRTT"]     = 0
		result[target]["maxRTT"]     = 0

		return result
	}

	pinger.Count = 10
	err = pinger.Run()
	if err != nil {
		result[target]["packetLoss"] = 0
		result[target]["minRTT"]     = 0
		result[target]["avgRTT"]     = 0
		result[target]["maxRTT"]     = 0

		return result
	}

	stats := pinger.Statistics()

	result[target]["packetLoss"] = stats.PacketLoss
	result[target]["minRTT"]     = float64(stats.MinRtt / time.Millisecond)
	result[target]["avgRTT"]     = float64(stats.AvgRtt / time.Millisecond)
	result[target]["maxRTT"]     = float64(stats.MaxRtt / time.Millisecond)

	return result

}

func getDownloadSpeed() float64 {
	fastCom := fast.New()

	// init
	err := fastCom.Init()
	if err != nil {
		return 0
	}

	// get urls
	urls, err := fastCom.GetUrls()
	if err != nil {
		return 0
	}

	// measure
	KbpsChan := make(chan float64)
	var last float64

	go func() {
		for Kbps := range KbpsChan {
			last = Kbps
		}
	}()

	err = fastCom.Measure(urls, KbpsChan)

	if err != nil {
		return 0
	}

	return last // return download speed in Kbps
}