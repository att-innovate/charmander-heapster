package main

import (
	"flag"
	"os"
	"strings"
	"time"

	"github.com/att-innovate/charmander-heapster/sinks"
	"github.com/att-innovate/charmander-heapster/sources"
	"github.com/golang/glog"
)

var argPollDuration = flag.Duration("poll_duration", 15*time.Second, "Polling duration")

func main() {
	flag.Parse()
	glog.Infof(strings.Join(os.Args, " "))
	glog.Infof("Heapster version %v", heapsterVersion)
	err := doWork()
	if err != nil {
		glog.Error(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func doWork() error {
	source, err := sources.NewSource(*argPollDuration)
	if err != nil {
		return err
	}
	sink, err := sinks.NewSink()
	if err != nil {
		return err
	}
	ticker := time.NewTicker(*argPollDuration)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			data, err := source.GetInfo()
			if err != nil {
				glog.Error(err)
				continue
			}
			if err := sink.StoreData(data); err != nil {
				return err
			}
		}
	}
	return nil
}
