package main

import (
    "crypto/tls"
    "encoding/json"
    "fmt"
    "github.com/ddo/go-fast"
    "io/ioutil"
    "log"
    "net"
    "net/http"
    "net/http/httptrace"
    "time"
)

// Init global types
type parameters struct {
    Ping          []string `json:"ping_target"`
    HTTP          []string `json:"http_target"`
    Interval      int      `json:"speed_interval"`
}

type ipInfoStruct struct {
    City          string  `json:"city"`
    Country       string  `json:"country"`
    ISP           string  `json:"org"`
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


func getParameters() (*parameters, error) {

    apiClient := http.Client{
        Timeout: time.Second * 2,
    }

    req, err := http.NewRequest(http.MethodGet, parametersURL, nil)
    if err != nil {
        return nil, fmt.Errorf("could not build HTTP request: %g", err)
    }

    res, getErr := apiClient.Do(req)

    if getErr != nil {
        return nil, fmt.Errorf("could not get HTTP request: %g", getErr)
    }

    if res.Body != nil {
        defer res.Body.Close()
    }

    body, readErr := ioutil.ReadAll(res.Body)
    if readErr != nil {
        return nil, fmt.Errorf("could not read Body: %g", readErr)
    }

    response := new(parameters)
    jsonErr := json.Unmarshal(body, &response)
    if jsonErr != nil {
        return nil, fmt.Errorf("could not unmarshal json: %g", jsonErr)
    }

    return response, nil
}


func getIspInfo() (*ipInfoStruct, error) {
    infoClient := http.Client{
        Timeout: time.Second * 2,
    }

    req, err := http.NewRequest(http.MethodGet, ipInfoURL, nil)
    if err != nil {
        return nil, fmt.Errorf("could not build HTTP request: %g", err)
    }

    res, getErr := infoClient.Do(req)
    if getErr != nil {
        return nil, fmt.Errorf("could not get HTTP request: %g", getErr)
    }

    if res.Body != nil {
        defer res.Body.Close()
    }

    body, readErr := ioutil.ReadAll(res.Body)
    if readErr != nil {
        return nil, fmt.Errorf("could not read body: %g", readErr)
    }

    respInfo := new(ipInfoStruct)
    jsonErr  := json.Unmarshal(body, &respInfo)
    if jsonErr != nil {
        return respInfo, fmt.Errorf("could not unmarshal JSON: %g", jsonErr)
    }

    return respInfo, nil

}


func getHTTPStat(url string, c chan map[string]map[string]int64) {

    schema := "https://"

    req, _ := http.NewRequest("GET", schema+url, nil)

    var start, connect, dns, tlsHandshake time.Time
    var reused bool

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

        GotConn: func(info httptrace.GotConnInfo) {
            reused = info.Reused
        },
    }


    req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
    start = time.Now()
    if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
        fmt.Errorf("could not run HTTP stat: %g", err)
    }

   if verbose {
       log.Printf("Is The connection to %v reused ? %v\n", url, reused)
   }

    c <- results
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
