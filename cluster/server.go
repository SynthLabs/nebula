package cluster

import (
	"fmt"
)

// Server implements the cluster server gRPC interface
type Server struct {
}

// StreamEvents is the receiver for getting events from the client
func (s *Server) StreamEvents(stream Cluster_StreamEventsServer) error {
	for {
		event, err := stream.Recv()
		if err != nil {
			return err
		}
		fmt.Println("EVENT: ", event)
	}
}
