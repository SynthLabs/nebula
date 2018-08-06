package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/synthlabs/nebula/cluster"
)

var (
	port = flag.Int("port", 7946, "member port")
)

func init() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	c, err := cluster.NewClusterManager(&cluster.ManagerConfig{
		DiscoveryListenPort: *port,
	})
	if err != nil {
		log.Fatalln("Failed to create cluster", err)
	}

	c.Start()

	c.Seed([]string{"nebula-1"})

	hostname, _ := os.Hostname()

	t := time.NewTicker(time.Second * time.Duration(rand.Intn(15)+1))
	for range t.C {
		log.Println("BROADCASTING EVENT FROM", hostname)
		event := &cluster.Event{
			Name: "Event from " + hostname,
		}
		c.Broadcast() <- event
	}
}
