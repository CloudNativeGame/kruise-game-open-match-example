package function

import (
	"github.com/CloudNativeGame/kruise-game-open-match-example/soloduel/pkg/function/mmf"
)

const (
	queryServiceAddr = "open-match-query.open-match.svc.cluster.local:50503" // Address of the QueryService endpoint.
	serverPort       = 50502                                                 // The port for hosting the Match Function.
)

func main() {
	mmf.Start(queryServiceAddr, serverPort)
}
