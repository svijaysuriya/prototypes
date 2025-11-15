package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

func getEnrichedDataFromGovt(ch chan interface{}, isChanClosed atomic.Bool) {
	time.Sleep(1 * time.Millisecond)
	if !isChanClosed.Load() {
		ch <- "NHIS data"
	}

}

func getEnrichedDataFromIntegration(ch chan interface{}, isChanClosed atomic.Bool) {
	time.Sleep(1 * time.Millisecond)
	if !isChanClosed.Load() {
		ch <- "Integration data"
	}

}
func getEnrichedDataFromGovtOrIntegration(vin string) interface{} {
	enrichedDataCh := make(chan interface{})
	isChannelClosed := atomic.Bool{}

	defer func() {
		isChannelClosed.Store(true)
		close(enrichedDataCh)
	}()
	go getEnrichedDataFromGovt(enrichedDataCh, isChannelClosed)
	go getEnrichedDataFromIntegration(enrichedDataCh, isChannelClosed)

	timeout := time.After(45 * time.Second)
	select {
	case data := <-enrichedDataCh:
		fmt.Println("got enrichedData , ", data)
		return data
	case <-timeout:
		return "timeout"
	}

}
