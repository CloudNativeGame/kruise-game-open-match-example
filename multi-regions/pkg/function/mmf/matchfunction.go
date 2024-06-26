package mmf

import (
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
)

var (
	matchName = "multi-clusters-1v1-matchfunction"
)

// matchFunctionService implements pb.MatchFunctionServer, the server generated
// by compiling the protobuf, by fulfilling the pb.MatchFunctionServer interface.
type matchFunctionService struct {
	grpc               *grpc.Server
	queryServiceClient pb.QueryServiceClient
	port               int
}

func getPools(profileName string) []*pb.Pool {
	var pools []*pb.Pool
	strs := strings.Split(profileName, "_")
	clusterName := strs[1]
	for i := 0; i < 2; i++ {
		pools = append(pools, &pb.Pool{
			Name: fmt.Sprintf("%s-pool", clusterName),
			StringEqualsFilters: []*pb.StringEqualsFilter{
				&pb.StringEqualsFilter{
					StringArg: "cluster-name",
					Value:     clusterName,
				},
			},
		})
	}
	return pools
}

func makeMatches(poolTickets map[string][]*pb.Ticket, profileName string) ([]*pb.Match, error) {
	tickets := map[string]*pb.Ticket{}
	for _, pool := range poolTickets {
		for _, ticket := range pool {
			tickets[ticket.GetId()] = ticket
		}
	}

	var matches []*pb.Match

	t := time.Now().Format("2006-01-02T15:04:05.00")

	thisMatch := make([]*pb.Ticket, 0, 2)
	matchNum := 0

	for _, ticket := range tickets {
		thisMatch = append(thisMatch, ticket)

		if len(thisMatch) >= 2 {
			matches = append(matches, &pb.Match{
				MatchId:       fmt.Sprintf("profile-%s-time-%s-num-%d", profileName, t, matchNum),
				MatchProfile:  profileName,
				MatchFunction: matchName,
				Tickets:       thisMatch,
			})

			thisMatch = make([]*pb.Ticket, 0, 2)
			matchNum++
		}
	}

	return matches, nil
}

// Run is this match function's implementation of the gRPC call defined in api/matchfunction.proto.
func (s *matchFunctionService) Run(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
	// Fetch tickets for the pools specified in the Match Profile.

	log.Printf("Generating proposals for function %v", req.GetProfile().GetName())
	profileName := req.GetProfile().GetName()
	poolTickets, err := matchfunction.QueryPools(stream.Context(), s.queryServiceClient, getPools(profileName))
	if err != nil {
		log.Printf("Failed to query tickets for the given pools, got %s", err.Error())
		return err
	}

	// Generate proposals.
	proposals, err := makeMatches(poolTickets, profileName)
	if err != nil {
		log.Printf("Failed to generate matches, got %s", err.Error())
		return err
	}

	log.Printf("Streaming %v proposals to Open Match", len(proposals))
	// Stream the generated proposals back to Open Match.
	for _, proposal := range proposals {
		if err := stream.Send(&pb.RunResponse{Proposal: proposal}); err != nil {
			log.Printf("Failed to stream proposals to Open Match, got %s", err.Error())
			return err
		}
	}

	return nil
}
