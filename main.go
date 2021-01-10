// ISPMON Client
// Gets list of targets from http://ispmon.cloud/config and executing the following:
// HTTP Trace - measure the DNS time, TLS Handshake time, Connection time and Time To First Byte.
// PING       - running 60 PING against the list of targets. reports back the avg RTT and Packet Loss ratio.
// SpeedTest  - Although the results are not always accurate, running SpeedTest every <interval> and reports back
//              Download and Upload speed.
//
//  When starting, you should see your UID printed to STDOUT.
//  To see your results please navigate to https://ispmon.cloud Grafana dashboard. (look for your UID)

package main

import (
    "bytes"
    "crypto/sha1"
    "encoding/json"
    "fmt"
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
    pingCount int = 10
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

    // Getting current time for Speedtest interval
    start := time.Now()

    // calculate user unique id based on MAC addresses. variable: uid
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


    // get client and ISP info from ipinfo.io
    GETINFO: // yes yes I know, goto is ugly...
        IpInfo, err := getIspInfo()
        if err !=nil {
            fmt.Printf("Error getting client IP info: %s", err )
            fmt.Println("Trying again in 2 sec")
            time.Sleep(2*time.Second)
            goto GETINFO
        }

    for {
        // getting targets and interval parameters
        GETPARAM:
            pingLinks, httpLinks, interval, err := getParameters()
            if err != nil {
                fmt.Printf("Error getting parameters: %s", err )
                fmt.Println("Trying again in 2 sec")
                goto GETPARAM
            }


        // http trace
        var httpResults []map[string]map[string]int64
        c := make(chan map[string]map[string]int64) // Using channel and not WaitingGroup just for fun :-)

        for _, link := range httpLinks {
            go getHttpStat(link, c)
        }

        for i := 0; i < len(httpLinks); i++ {
            httpResults = append(httpResults, <-c)
        }

        // PING test
        var wg sync.WaitGroup
        var pingResults []map[string]map[string]float64

        for _, t := range pingLinks {
            wg.Add(1)
            go func(t string) {
                result := getPingStat(t, &wg)
                pingResults = append(pingResults, result)
            }(t)
        }

        wg.Wait()

        // speed test
        downloadSpeed := "null"
        t := time.Now()
        elapsed := t.Sub(start).Minutes()
        if int(elapsed) >= interval {
            downloadSpeed = fmt.Sprintf("%f", getDownloadSpeed()) //float64 to String
            start = time.Now()
        }

        // push results
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

        // Print Response
        //fmt.Println("response Status:", resp.Status)
        //fmt.Println("response Headers:", resp.Header)
        //body, _ := ioutil.ReadAll(resp.Body)
        //fmt.Println("response Body:", string(body))
    }


}
