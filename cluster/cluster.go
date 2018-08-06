package cluster

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	"github.com/hashicorp/memberlist"
)

const (
	defaultRPCPort = 7777
)

// Manager is the main Cluster entity, it handles node discovery, stream setup, and broadcasting
type Manager struct {
	// private fields
	config     *ManagerConfig
	broadcasts chan *Event
	membership *memberlist.Memberlist
	name       string

	grpcServer  *grpc.Server
	rpcListener net.Listener

	streams     map[string]Cluster_StreamEventsClient
	connections map[string]*grpc.ClientConn

	wg sync.WaitGroup
	mu sync.Mutex
}

// ManagerConfig holds all the configuration for a cluster
type ManagerConfig struct {
	BroadcastChannelSize int
	DiscoveryListenPort  int
	StreamListenPort     int
}

// NewClusterManager returns a configured cluster manager
func NewClusterManager(config *ManagerConfig) (*Manager, error) {
	var err error

	if config.StreamListenPort == 0 {
		config.StreamListenPort = defaultRPCPort
	}

	manager := &Manager{
		config:      config,
		broadcasts:  make(chan *Event, config.BroadcastChannelSize),
		connections: map[string]*grpc.ClientConn{},
		streams:     map[string]Cluster_StreamEventsClient{},
	}

	conf := memberlist.DefaultLocalConfig()
	conf.BindPort = config.DiscoveryListenPort
	conf.Events = manager
	conf.LogOutput = ioutil.Discard

	conf.Name, err = os.Hostname()
	if err != nil {
		return nil, err
	}

	manager.name = conf.Name

	manager.membership, err = memberlist.Create(conf)
	if err != nil {
		return nil, err
	}

	log.Println("[DEBUG] ", manager.membership.LocalNode())

	return manager, nil
}

// Start the cluster, must be called before Seed
func (m *Manager) Start() error {
	// Setup gRPC
	var err error
	m.rpcListener, err = net.Listen("tcp", fmt.Sprintf(":%d", m.config.StreamListenPort))
	if err != nil {
		return err
	}

	m.grpcServer = grpc.NewServer()

	RegisterClusterServer(m.grpcServer, &Server{})

	m.wg.Add(1)
	go m.startRPCServer()

	m.wg.Add(1)
	go m.broadcastLoop()

	return nil
}

func (m *Manager) startRPCServer() {
	err := m.grpcServer.Serve(m.rpcListener)
	if err != nil {
		log.Fatalln("[ERROR] grpc server died", err)
	}
}

func (m *Manager) broadcastLoop() {
	for e := range m.broadcasts {
		for node, stream := range m.streams {
			if node != m.membership.LocalNode().Name {
				err := stream.Send(e)
				if err != nil {
					log.Fatalln("[ERROR] failed to send event", err)
				}
			}
		}
	}
}

// Seed the cluster with an initial set of nodes, start must be called first
func (m *Manager) Seed(hosts []string) error {
	_, err := m.membership.Join(hosts)
	return err
}

// Broadcast returns the input channel used to broadcast events
func (m *Manager) Broadcast() chan<- *Event {
	return m.broadcasts
}

// Close shuts down the cluster
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, stream := range m.streams {
		stream.CloseAndRecv()
	}

	for _, conn := range m.connections {
		conn.Close()
	}

	m.grpcServer.GracefulStop()
}

// NotifyJoin is invoked when a node is detected to have joined.
func (m *Manager) NotifyJoin(node *memberlist.Node) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if node.Name == m.name {
		return
	}

	log.Println("[JOIN] got new member", node.Name)
	conn, err := grpc.Dial(fmt.Sprintf("%s:7777", node.Addr), grpc.WithInsecure())
	if err != nil {
		log.Println("[ERROR] failed to connect to grpc server:", node.Name, node.Addr, err)
		return
	}

	m.connections[node.Name] = conn

	client := NewClusterClient(conn)
	stream, err := client.StreamEvents(context.Background())
	if err != nil {
		log.Println("[ERROR] failed to create grpc stream:", node.Name, node.Addr, err)
		return
	}

	m.streams[node.Name] = stream
}

// NotifyLeave is invoked when a node is detected to have left.
func (m *Manager) NotifyLeave(node *memberlist.Node) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if node.Name == m.name {
		return
	}

	log.Println("[LEAVE] lost member", node.Name)

	m.streams[node.Name].CloseSend()
	m.connections[node.Name].Close()

	delete(m.streams, node.Name)
	delete(m.connections, node.Name)
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data.
func (m *Manager) NotifyUpdate(node *memberlist.Node) {
	log.Println("[UPDATE] member updated", node.Name)
}
