// ISPMON Client
// Gets list of targets from http://ispmon.cloud/config and executing the following:
// HTTP Trace - measure the DNS time, TLS Handshake time, Connection time and Time To First Byte.
// PING       - running 120 PING (by default) against the list of targets. reports back the avg RTT and Packet Loss ratio.
// SpeedTest  - Using Netflix Fast.com Speed-test every <interval>
//
//  When starting, you should see your UID printed to STDOUT.
//  To see your results please navigate to https://ispmon.cloud Grafana dashboard. (look for your UID)

package main

import (
    "bytes"
    "crypto/sha1"
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "net"
    "net/http"
    "os"
    "sync"
    "time"
)

const (
    parametersURL string = "http://ispmon.cloud/config"
    ipInfoURL     string = "http://ipinfo.io/"
    pushInfoURL   string = "http://ispmon.cloud/gometrics"
)

// How many PING to send
var (
    pingCount uint
    verbose bool
)


type pushResults struct {
    HTTP        []map[string]map[string]int64
    Ping        map[string]*pingResult
    Speed       string
    UID         string
    Isp         string
    Country     string
    City        string
}


func main() {

    flag.UintVar(&pingCount, "ping-count", 60, "ping count")
    flag.BoolVar(&verbose, "verbose", false, "enable verbose logging")
    flag.Parse()

    // GETTING CURRENT TIME FOR SPEED-TEST INTERVAL
    start := time.Now()

    // CALCULATE USER UNIQUE ID - UID
    mac, err := getMacAddr()

    if err != nil {
        log.Printf("Error getting Mac address: %s", err )
        os.Exit(1) // If we cant generate UID there is no pint to continue
    }

    var macString string
    for _, addr := range mac{
        macString = macString + addr
    }

    macSha1  := sha1.Sum([]byte(macString))
    uid := fmt.Sprintf("%x\n", string(macSha1[:]))[0:10]
    log.Printf("Your Unique ID is: %v. Go to https://ispmon.cloud to see your results", uid)


    GETINFO: // yes yes I know, goto is ugly...
        IPInfo, err := getIspInfo()
        if err !=nil {
            log.Printf("Error getting client IP info: %s", err )
            log.Println("Trying again in 2 sec")
            time.Sleep(2*time.Second)
            goto GETINFO
        }

    if verbose {
        log.Printf("IpInfo=%v", IPInfo)
    }

    var pingMutex = &sync.Mutex{}

    for {
        // GET TARGETS AND INTERVAL INFO
        GETPARAM:
            params, err := getParameters()
            if err != nil {
                log.Printf("Error getting parameters: %s", err )
                log.Println("Trying again in 2 sec")
                time.Sleep(2*time.Second)
                goto GETPARAM
            }

        if verbose {
            log.Printf("params=%v", params)
        }


        // RUN HTTP TRACE

        // Editing transport parameters to avoid connection reuse
        http.DefaultTransport = &http.Transport{
            ForceAttemptHTTP2:     false,
            IdleConnTimeout:       1 * time.Second,
            DialContext: (&net.Dialer{
                Timeout:   10 * time.Second,
                KeepAlive: 1 * time.Second,
            }).DialContext,
        }

        var httpResults []map[string]map[string]int64
        c := make(chan map[string]map[string]int64) // Using channel and not WaitingGroup just for fun :-)

        for _, link := range params.HTTP {
            go getHTTPStat(link, c)
        }

        for i := 0; i < len(params.HTTP); i++ {
            httpResults = append(httpResults, <-c)
        }

        if verbose {
            log.Printf("http results: %v", httpResults)
        }

        // PING TEST
        var wg sync.WaitGroup
        pingResults := make(map[string]*pingResult)

        for _, t := range params.Ping {
            wg.Add(1)
            go func(t string, mutex *sync.Mutex) {
                if verbose {
                    log.Printf("ping '%s' starting", t)
                }
                result := getPingStat(t, pingCount)
                // maps are not thread safe
                mutex.Lock()
                pingResults[t] = result
                mutex.Unlock()
                wg.Done()
            }(t, pingMutex)
        }

        wg.Wait()

        if verbose {
            for t, r := range pingResults {
                log.Printf("ping: %+v, results: %+v\n", t, r)
            }
        }

        // RUN SPEED TEST
        downloadSpeed := "null"
        t := time.Now()
        elapsed := t.Sub(start).Minutes()
        if int(elapsed) >= params.Interval {
            downloadSpeed = fmt.Sprintf("%f", getDownloadSpeed()) //float64 to String
            start = time.Now()
        }

        // PUSH RESULTS
        push := pushResults{}
        push.Speed = downloadSpeed
        push.HTTP = httpResults
        push.Ping = pingResults
        push.Isp = IPInfo.ISP
        push.Country = IPInfo.Country
        push.City = IPInfo.City
        push.UID = uid

        b, err := json.Marshal(push)
        req, err := http.NewRequest("POST", pushInfoURL, bytes.NewBuffer(b))
        req.Header.Set("Content-Type", "application/json")

        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            log.Printf("could not send HTTP POST: %s", err)
        }
        defer resp.Body.Close()

        // Print Response if verbose
        if verbose {
            log.Println("response Status:", resp.Status)
            // log.Println("response Headers:", resp.Header)
            body, _ := ioutil.ReadAll(resp.Body)
            log.Println("response Body:", string(body))
            log.Println("end of push\n\n")

        }
    }


}
