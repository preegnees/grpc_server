package common

import (
	"fmt"
	
	m "streaming/pkg/models"
)

func PeertoString(peer m.Peer) string {
	return fmt.Sprintf(
		"Name=%s, IdChannel=%s, AllowedNames=%s, GrpcStream=%v",
		peer.Name, peer.IdChannel, peer.AllowedNames, peer.GrpcStream,
	)
}