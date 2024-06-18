package main

import (
	// Uncomment if following the tutorial
	"math/rand"

	"open-match.dev/open-match/pkg/pb"
)

const (
	ClusterNameKey     = "cluster-name"
	HostClusterName    = "Host"
	RegionBClusterName = "region-b"
)

// Ticket generates a Ticket with data using the package configuration.
func makeTicket() *pb.Ticket {
	return &pb.Ticket{
		SearchFields: &pb.SearchFields{
			StringArgs: generateData(),
		},
	}
}

func generateData() map[string]string {
	var randomClusterName string
	randomValue := rand.Float64()
	if randomValue < 0.6 {
		randomClusterName = HostClusterName
	} else {
		randomClusterName = RegionBClusterName
	}

	return map[string]string{
		ClusterNameKey: randomClusterName,
	}
}
