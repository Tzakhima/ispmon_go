package main

import (
    "github.com/go-ping/ping"
    "log"
    "time"
)

// PingResult contains the results ping tests on teh target
type pingResult struct {
	PacketLoss  float64 `json:"packetLoss"`
	MinRTT		float64 `json:"minRTT"`
	AvgRTT		float64 `json:"avgRTT"`
	MaxRTT      float64 `json:"maxRTT"`
}

// GetPingStat will ping target count times and return PinResult
func getPingStat(target string, count uint) *pingResult {

    result := new(pingResult)

    pinger, err := ping.NewPinger(target)
    pinger.SetPrivileged(true)
        
    if err != nil {
        log.Printf("%+v", err)
        result.PacketLoss = 0
        result.MinRTT     = 0
        result.AvgRTT     = 0
        result.MaxRTT     = 0

        return result
    }

    pinger.Count = int(count) // why is this int?  you can't send a negitive number of pings!
    pinger.Timeout = time.Duration((count+5) * uint(time.Second))
    err = pinger.Run()
    if err != nil {
        log.Printf("%+v", err)
        result.PacketLoss = 0
        result.MinRTT     = 0
        result.AvgRTT     = 0
        result.MaxRTT     = 0

        return result
    }

    stats := pinger.Statistics()

    result.PacketLoss = stats.PacketLoss
    result.MinRTT     = float64(stats.MinRtt / time.Millisecond)
    result.AvgRTT     = float64(stats.AvgRtt / time.Millisecond)
    result.MaxRTT     = float64(stats.MaxRtt / time.Millisecond)

    return result
}
