package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/CursedHardware/ch912x"
	"gopkg.in/antage/eventsource.v1"
)

var plane *ch912x.ControlPlane

func init() {
	var err error
	var nic string
	flag.StringVar(&nic, "nic", "", "")
	flag.Parse()
	plane, err = ch912x.ListenCH912XByName(nic)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	events := eventsource.New(nil, nil)
	defer events.Close()
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	mux.Handle("/events", events)
	mux.Handle("/api/", http.StripPrefix("/api", makeAPIService()))
	go sendDiscoveryEvents(events)
	log.Println("listen :8080")
	_ = http.ListenAndServe(":8080", mux)
}

func sendDiscoveryEvents(events eventsource.EventSource) {
	for module := range plane.Discovery() {
		data, _ := json.Marshal(module)
		events.SendEventMessage(string(data), "discovery", "")
	}
}
