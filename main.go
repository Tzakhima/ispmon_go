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
    "log"
    "net"
    "net/http"
    "os"
    "sync"
    "time"
)

const (
    ParametersUrl string = "http://ispmon.cloud/config"
    IpInfoUrl     string = "http://ipinfo.io/"
    PushInfoUrl   string = "http://ispmon.cloud/gometrics"
)

// How many PING to send
var (
    pingCount uint
    verbose bool
)

type PushResults struct {
    Http        []map[string]map[string]int64
    Ping        []map[string]map[string]float64
    Speed       string
    Uid         string
    Isp         string
    Country     string
    City        string
}


func main() {

    flag.UintVar(&pingCount, "ping-count", 120, "ping count")
    flag.BoolVar(&verbose, "verbose", false, "enable verbose logging")
    flag.Parse()

    // GETTING CURRENT TIME FOR SPEED-TEST INTERVAL
    start := time.Now()

    // CALCULATE USER UNIQUE ID - UID
    mac, err := getMacAddr()

    if err != nil {
        fmt.Printf("Error getting Mac address: %s", err )
        os.Exit(1) // If we cant generate UID there is no pint to continue
    }

    var macString string
    for _, addr := range mac{
        macString = macString + addr
    }

    macSha1  := sha1.Sum([]byte(macString))
    uid := fmt.Sprintf("%x\n", string(macSha1[:]))[0:10]
    fmt.Printf("Your Unique ID is: %v. Go to https://ispmon.cloud to see your results", uid)


    // GET CLIENT IP AND ISP INFO
    GETINFO: // yes yes I know, goto is ugly...
        IpInfo, err := getIspInfo()
        if err !=nil {
            fmt.Printf("Error getting client IP info: %s", err )
            fmt.Println("Trying again in 2 sec")
            time.Sleep(2*time.Second)
            goto GETINFO
        }

    if verbose {
        log.Printf("IpInfo=%v", IpInfo)
    }

    for {
        // GET TARGETS AND INTERVAL INFO
        GETPARAM:
            pingLinks, httpLinks, interval, err := getParameters()
            if err != nil {
                log.Printf("Error getting parameters: %s", err )
                log.Println("Trying again in 2 sec")
                goto GETPARAM
            }

        if verbose {
            log.Printf("pingLinks=%v", pingLinks)
            log.Printf("httpLinks=%v", httpLinks)
            log.Printf("interval=%d",  interval)
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

        for _, link := range httpLinks {
            go getHttpStat(link, c)
        }

        for i := 0; i < len(httpLinks); i++ {
            httpResults = append(httpResults, <-c)
        }

        if verbose {
            log.Printf("http results: %v", httpResults)
        }

        // PING TEST
        var wg sync.WaitGroup
        var pingResults []map[string]map[string]float64

        for _, t := range pingLinks {
            wg.Add(1)
            go func(t string) {
                if verbose {
                    log.Printf("ping '%s' starting", t)
                }
                result := getPingStat(t, &wg)
                if verbose {
                    log.Printf("ping '%s' results=%v", t, result)
                }
                pingResults = append(pingResults, result)
                wg.Done()
            }(t)
        }

        wg.Wait()

        if verbose {
            log.Printf("ping results: %v", pingResults)
        }

        // RUN SPEED TEST
        downloadSpeed := "null"
        t := time.Now()
        elapsed := t.Sub(start).Minutes()
        if int(elapsed) >= interval {
            downloadSpeed = fmt.Sprintf("%f", getDownloadSpeed()) //float64 to String
            start = time.Now()
        }

        // PUSH RESULTS
        push := PushResults{}
        push.Speed = downloadSpeed
        push.Http = httpResults
        push.Ping = pingResults
        push.Isp = IpInfo["isp"]
        push.Country = IpInfo["country"]
        push.City = IpInfo["city"]
        push.Uid = uid

        b, err := json.Marshal(push)
        req, err := http.NewRequest("POST", PushInfoUrl, bytes.NewBuffer(b))
        req.Header.Set("Content-Type", "application/json")

        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            fmt.Errorf("could not send HTTP POST: %g", err)
        }
        defer resp.Body.Close()

        // Print Response if verbose
        if verbose {
            log.Println("response Status:", resp.Status)
            // log.Println("response Headers:", resp.Header)
            // body, _ := ioutil.ReadAll(resp.Body)
            // fmt.Println("response Body:", string(body))

        }
    }


}
