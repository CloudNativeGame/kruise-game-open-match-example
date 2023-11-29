package mmf

import (
	"fmt"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"log"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
)

var (
	matchName = "normal-5v5-matchfunction"
)

const (
	dividedTeamMapKey = "Assignment/divided-team-map"
	pointKey          = "point"
	playersNumKey     = "player-num"
	maxPoint          = 5000
)

// matchFunctionService implements pb.MatchFunctionServer, the server generated
// by compiling the protobuf, by fulfilling the pb.MatchFunctionServer interface.
type matchFunctionService struct {
	grpc               *grpc.Server
	queryServiceClient pb.QueryServiceClient
	port               int
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

func getPools(profileName string) []*pb.Pool {
	var pools []*pb.Pool
	var createdAfter *timestamppb.Timestamp
	var createdBefore *timestamppb.Timestamp
	var tagPresentFilters []*pb.TagPresentFilter
	var doubleRangeFilter []*pb.DoubleRangeFilter
	var stringEqualsFilters []*pb.StringEqualsFilter

	for level := 0; level < 4; level++ {
		distance := maxPoint
		switch level {
		case 0:
			// level0 等待15秒内，分差200，地图筛选
			createdAfter = timestamppb.New(time.Now().Add(-15 * time.Second))
			createdBefore = nil
			distance = 200
			tagPresentFilters = []*pb.TagPresentFilter{
				{
					Tag: profileName,
				},
			}

		case 1:
			// level1 等待150内，分差200，随机地图
			createdAfter = timestamppb.New(time.Now().Add(-150 * time.Second))
			createdBefore = timestamppb.New(time.Now().Add(-15 * time.Second))
			distance = 200
			tagPresentFilters = nil

		case 2:
			// level2 等待200内，分差500，随机地图
			createdAfter = timestamppb.New(time.Now().Add(-200 * time.Second))
			createdBefore = timestamppb.New(time.Now().Add(-150 * time.Second))
			distance = 500
			tagPresentFilters = nil

		case 3:
			// level3 等待超过200s，不设分差，随机地图
			createdBefore = timestamppb.New(time.Now().Add(-200 * time.Second))
			createdAfter = nil
			tagPresentFilters = nil

		}

		for playersNum := 1; playersNum <= 5; playersNum++ {
			playersNumString := strconv.Itoa(playersNum)
			stringEqualsFilters = []*pb.StringEqualsFilter{
				{
					StringArg: playersNumKey,
					Value:     playersNumString,
				},
			}
			for point := 0; point < maxPoint; point = point + distance {
				doubleRangeFilter = []*pb.DoubleRangeFilter{
					{
						DoubleArg: pointKey,
						Min:       float64(point),
						Max:       float64(point + distance - 1),
					},
				}
				minPointValue := strconv.Itoa(point)
				maxPointValue := strconv.Itoa(point + distance - 1)
				pools = append(pools, &pb.Pool{
					// level<id>-分值最小值-分值最大值-人数
					Name:                "level" + strconv.Itoa(level) + "-" + minPointValue + "-" + maxPointValue + "-" + playersNumString,
					TagPresentFilters:   tagPresentFilters,
					DoubleRangeFilters:  doubleRangeFilter,
					StringEqualsFilters: stringEqualsFilters,
					CreatedBefore:       createdBefore,
					CreatedAfter:        createdAfter,
				})
			}
		}
	}

	return pools
}

func makeMatches(poolTickets map[string][]*pb.Ticket, profileName string) ([]*pb.Match, error) {
	var matches []*pb.Match
	matchNum := 0
	for level := 0; level < 4; level++ {
		distance := maxPoint
		switch level {
		case 0:
			distance = 200
		case 1:
			distance = 200
		case 2:
			distance = 500
		}
		for point := 0; point < maxPoint; point = point + distance {
			minPointValue := strconv.Itoa(point)
			maxPointValue := strconv.Itoa(point + distance - 1)
			prefixName := "level" + strconv.Itoa(level) + "-" + minPointValue + "-" + maxPointValue + "-"
			var tickets [][]*pb.Ticket
			for playersNum := 1; playersNum <= 5; playersNum++ {
				playersNumString := strconv.Itoa(playersNum)
				tickets = append(tickets, poolTickets[prefixName+playersNumString])
			}
			matchesPerLevelPerPointsRange, err := makeMatchesPerLevelPerPointsRange(tickets, profileName, &matchNum)
			if err != nil {
				return nil, nil
			}
			matches = append(matches, matchesPerLevelPerPointsRange...)
		}
	}
	return matches, nil
}

// 在五组人数分别为1、2、3、4、5的tickets中构成对局，输入参数tickets对应的index为人数-1
func makeMatchesPerLevelPerPointsRange(tickets [][]*pb.Ticket, profileName string, matchNum *int) ([]*pb.Match, error) {
	var matches []*pb.Match
	// 五组index，分别记录组队玩家数量1、2、3、4、5对应tickets所在位置，避免重复分配
	indexes := []int{0, 0, 0, 0, 0}
	noMoreMatch := false

	for !noMoreMatch {
		var matchTickets []*pb.Ticket
		var dividedTeamMap string
		// teamIndex为0，组成A队；teamIndex为1，构成B队

		for teamIndex := 0; teamIndex < 2; teamIndex++ {
			situations := [][]int{
				// 0. 选择5人队
				{0, 0, 0, 0, 1},
				// 1. 选择4人队 + 1人队
				{1, 0, 0, 1, 0},
				// 2. 选择3人队 + 2人队
				{0, 1, 1, 0, 0},
				// 3. 选择3人队 + 1人队 + 1人队
				{2, 0, 1, 0, 0},
				// 4. 选择2人队 + 2人队 + 1人队
				{1, 2, 0, 0, 0},
				// 5. 选择2人队 + 1人队 + 1人队 + 1人队
				{3, 1, 0, 0, 0},
				// 6. 选择1人队 + 1人队 + 1人队 + 1人队 + 1人队
				{5, 0, 0, 0, 0},
			}

			var takenTickets []*pb.Ticket
			takenTickets = nil
			for _, playerTicketsNum := range situations {
				takenTickets = getTicketByPlayerNum(tickets, indexes, playerTicketsNum)
				if takenTickets != nil {
					matchTickets = append(matchTickets, takenTickets...)
					dividedTeamMap = teamDivision(teamIndex, dividedTeamMap, takenTickets)
					break
				}
			}
			if takenTickets == nil {
				noMoreMatch = true
				break
			}
		}

		if !noMoreMatch {
			t := time.Now().Format("2006-01-02T15:04:05.00")
			extensions := make(map[string]*anypb.Any)
			dividedTeamMapAny, err := anypb.New(&wrapperspb.StringValue{Value: dividedTeamMap})
			if err != nil {
				return nil, err
			}
			extensions[dividedTeamMapKey] = dividedTeamMapAny
			matches = append(matches, &pb.Match{
				MatchId:       fmt.Sprintf("profile-%s-time-%s-num-%d", profileName, t, matchNum),
				MatchProfile:  profileName,
				MatchFunction: matchName,
				Tickets:       matchTickets,
				Extensions:    extensions,
			})
			*matchNum++
		}
	}

	return matches, nil
}

func getTicketByPlayerNum(tickets [][]*pb.Ticket, indexes []int, playerTicketsNum []int) []*pb.Ticket {
	var takenTickets []*pb.Ticket
	takenTickets = nil

	for index := 0; index < len(playerTicketsNum); index++ {
		ticketNum := playerTicketsNum[index]
		if ticketNum == 0 {
			continue
		}

		// 超出了对应可获取的数量
		if index+ticketNum > len(tickets[index]) {
			return nil
		}

		// 取出对应的tickets并更新index位置
		takenTickets = append(takenTickets, tickets[index][indexes[index]:indexes[index+ticketNum]]...)
		indexes[index] = indexes[index] + ticketNum
	}

	return takenTickets
}

func teamDivision(teamIndex int, dividedTeamMap string, takenTickets []*pb.Ticket) string {
	var takenTicketsIds []string
	for _, t := range takenTickets {
		takenTicketsIds = append(takenTicketsIds, t.Id)
	}
	if teamIndex == 0 {
		return "A:" + strings.Join(takenTicketsIds, ",")
	}
	return dividedTeamMap + ";B:" + strings.Join(takenTicketsIds, ",")
}
