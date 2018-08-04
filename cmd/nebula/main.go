package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/memberlist"
)

var (
	port = flag.Int("port", 7946, "member port")
)

func init() {
	flag.Parse()
}

func main() {
	conf := memberlist.DefaultLocalConfig()
	conf.BindPort = *port

	list, err := memberlist.Create(conf)
	if err != nil {
		log.Fatal("Failed to create memberlist: " + err.Error())
	}

	_, err = list.Join([]string{"nebula-1"})
	if err != nil {
		log.Fatal(err)
	}

	node := list.LocalNode()
	log.Printf("Local member %s:%d\n", node.Addr, node.Port)

	// Ask for members of the cluster
	for _, member := range list.Members() {
		log.Printf("Member: %s %s\n", member.Name, member.Addr)
	}

	p := node.Port + 1

	if err := http.ListenAndServe(fmt.Sprintf(":%d", p), nil); err != nil {
		fmt.Println(err)
	}
}
