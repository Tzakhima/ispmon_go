package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptrace"
	"time"
)

type parameters struct {
	Ping     []string `json:"ping_target"`
	HTTP     []string `json:"http_target"`
	Interval int      `json:"speed_interval"`
}

func getParameters() ([]string, []string, int) {

	apiURL := "http://ispmon.cloud/config"

	apiClient := http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		log.Fatal(err)
	}

	res, getErr := apiClient.Do(req)

	if getErr != nil {
		log.Fatal(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	response := parameters{}
	jsonErr := json.Unmarshal(body, &response)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	fmt.Println(response.Ping)

	return response.Ping, response.HTTP, response.Interval

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
