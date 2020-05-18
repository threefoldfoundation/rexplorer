package gateway

import (
	"time"

	"github.com/threefoldtech/rivine/build"
)

const (
	// maxLocalOutbound is currently set to 3, meaning the gateway will not
	// consider a local node to be an outbound peer if the gateway already has
	// 3 outbound peers. Three is currently needed to handle situations where
	// the gateway is at high risk of connecting to itself (such as a low
	// number of total peers, especially such as in a testing environment).
	// Once the gateway has a proper way to figure out that it's trying to
	// connect to itself, this number can be reduced.
	maxLocalOutboundPeers = 3

	// EncodedSessionHeaderLength is the static length of a session header encoded
	// with the encode package.
	// sizeof(blockID) + sizeof(gatewayID) + sizeof(bool) = 32 + 8 + 1 = 41
	EncodedSessionHeaderLength = 41

	// saveFrequency defines how often the gateway saves its persistence.
	saveFrequency = time.Minute * 2
)

var (
	// MinAcceptableVersion is the version below which the gateway will refuse to
	// connect to peers and reject connection attempts
	MinAcceptableVersion = build.NewVersion(1, 0, 0, 0)

	// HandshakNetAddressUpgrade is the version where we upgraded the handshake,
	// to replace the wantConn with a NetAddr.
	HandshakNetAddressUpgrade = build.NewVersion(1, 0, 2, 0)

	// fastNodePurgeDelay defines the amount of time that is waited between each
	// iteration of the purge loop when the gateway has enough nodes to be
	// needing to purge quickly.
	fastNodePurgeDelay = build.Select(build.Var{
		Standard: 1 * time.Minute,
		Dev:      5 * time.Second,
		Testing:  1 * time.Second,
	}).(time.Duration)

	// healthyNodeListLen defines the number of nodes that the gateway must
	// have in the node list before it will stop asking peers for more nodes.
	healthyNodeListLen = build.Select(build.Var{
		Standard: int(200),
		Dev:      int(30),
		Testing:  int(15),
	}).(int)

	// maxSharedNodes defines the number of nodes that will be shared between
	// peers when they are expanding their node lists.
	maxSharedNodes = build.Select(build.Var{
		Standard: uint64(10),
		Dev:      uint64(5),
		Testing:  uint64(3),
	}).(uint64)

	// nodePurgeDelay defines the amount of time that is waited between each
	// iteration of the node purge loop.
	nodePurgeDelay = build.Select(build.Var{
		Standard: 10 * time.Minute,
		Dev:      20 * time.Second,
		Testing:  6 * time.Second,
	}).(time.Duration)

	// nodeListDelay defines the amount of time that is waited between each
	// iteration of the node list loop.
	nodeListDelay = build.Select(build.Var{
		Standard: 5 * time.Second,
		Dev:      3 * time.Second,
		Testing:  1 * time.Second,
	}).(time.Duration)

	// pruneNodeListLen defines the number of nodes that the gateway must have
	// to be pruning nodes from the node list.
	pruneNodeListLen = build.Select(build.Var{
		Standard: int(50),
		Dev:      int(15),
		Testing:  int(10),
	}).(int)

	// quickPruneListLen defines the number of nodes that the gateway must have
	// to be pruning nodes quickly from the node list.
	quickPruneListLen = build.Select(build.Var{
		Standard: int(250),
		Dev:      int(40),
		Testing:  int(20),
	}).(int)
)

var (
	// The gateway will sleep this long between incoming connections. For
	// attack reasons, the acceptInterval should be longer than the
	// nodeListDelay. Right at startup, a node is vulnerable to being flooded
	// by Sybil attackers. The node's best defense is to wait until it has
	// filled out its nodelist somewhat from the bootstrap nodes. An attacker
	// needs to completely dominate the nodelist and the peerlist to be
	// successful, so just a few honest nodes from requests to the bootstraps
	// should be enough to fend from most attacks.
	acceptInterval = build.Select(build.Var{
		Standard: 6 * time.Second,
		Dev:      3 * time.Second,
		Testing:  100 * time.Millisecond,
	}).(time.Duration)

	// acquiringPeersDelay defines the amount of time that is waited between
	// iterations of the peer acquisition loop if the gateway is actively
	// forming new connections with peers.
	acquiringPeersDelay = build.Select(build.Var{
		Standard: 5 * time.Second,
		Dev:      3 * time.Second,
		Testing:  500 * time.Millisecond,
	}).(time.Duration)

	// fullyConnectedThreshold defines the number of peers that the gateway can
	// have before it stops accepting inbound connections.
	fullyConnectedThreshold = build.Select(build.Var{
		Standard: 128,
		Dev:      20,
		Testing:  10,
	}).(int)

	// maxConcurrentOutboundPeerRequests defines the maximum number of peer
	// connections that the gateway will try to form concurrently.
	maxConcurrentOutboundPeerRequests = build.Select(build.Var{
		Standard: 3,
		Dev:      2,
		Testing:  2,
	}).(int)

	// noNodesDelay defines the amount of time that is waited between
	// iterations of the peer acquisition loop if the gateway does not have any
	// nodes in the nodelist.
	noNodesDelay = build.Select(build.Var{
		Standard: 20 * time.Second,
		Dev:      10 * time.Second,
		Testing:  3 * time.Second,
	}).(time.Duration)

	// unwawntedLocalPeerDelay defines the amount of time that is waited
	// between iterations of the permanentPeerManager if the gateway has at
	// least a few outbound peers, but is not well connected, and the recently
	// selected peer was a local peer. The wait is mostly to prevent the
	// gateway from hogging the CPU in the event that all peers are local
	// peers.
	unwantedLocalPeerDelay = build.Select(build.Var{
		Standard: 2 * time.Second,
		Dev:      1 * time.Second,
		Testing:  100 * time.Millisecond,
	}).(time.Duration)

	// wellConnectedDelay defines the amount of time that is waited between
	// iterations of the peer acquisition loop if the gateway is well
	// connected.
	wellConnectedDelay = build.Select(build.Var{
		Standard: 5 * time.Minute,
		Dev:      1 * time.Minute,
		Testing:  3 * time.Second,
	}).(time.Duration)

	// wellConnectedThreshold is the number of outbound connections at which
	// the gateway will not attempt to make new outbound connections.
	wellConnectedThreshold = build.Select(build.Var{
		Standard: 8,
		Dev:      5,
		Testing:  4,
	}).(int)
)

var (
	// connStdDeadline defines the standard deadline that should be used for
	// all temporary connections to the gateway.
	connStdDeadline = build.Select(build.Var{
		Standard: 5 * time.Minute,
		Dev:      2 * time.Minute,
		Testing:  30 * time.Second,
	}).(time.Duration)

	// the gateway will abort a connection attempt after this long
	dialTimeout = build.Select(build.Var{
		Standard: 3 * time.Minute,
		Dev:      20 * time.Second,
		Testing:  6 * time.Second,
	}).(time.Duration)

	// rpcStdDeadline defines the standard deadline that should be used for all
	// incoming RPC calls.
	rpcStdDeadline = build.Select(build.Var{
		Standard: 5 * time.Minute,
		Dev:      3 * time.Minute,
		Testing:  10 * time.Second,
	}).(time.Duration)
)

var (
	// minPeersForIPDiscovery is the minimum number of peer connections we wait
	// for before we try to discover our public ip from them. It is also the
	// minimum number of successful replies we expect from our peers before we
	// accept a result.
	minPeersForIPDiscovery = build.Select(build.Var{
		Standard: 5,
		Dev:      3,
		Testing:  2,
	}).(int)

	// timeoutIPDiscovery is the time after which managedIPFromPeers will fail
	// if the ip couldn't be discovered successfully.
	timeoutIPDiscovery = build.Select(build.Var{
		Standard: 5 * time.Minute,
		Dev:      5 * time.Minute,
		Testing:  time.Minute,
	}).(time.Duration)

	// rediscoverIPIntervalSuccess is the time that has to pass after a
	// successful IP discovery before we rediscover the IP.
	rediscoverIPIntervalSuccess = build.Select(build.Var{
		Standard: 3 * time.Hour,
		Dev:      10 * time.Minute,
		Testing:  30 * time.Second,
	}).(time.Duration)

	// rediscoverIPIntervalFailure is the time that has to pass after a failed
	// IP discovery before we try again.
	rediscoverIPIntervalFailure = build.Select(build.Var{
		Standard: 15 * time.Minute,
		Dev:      1 * time.Minute,
		Testing:  10 * time.Second,
	}).(time.Duration)

	// peerDiscoveryRetryInterval is the time we wait when there were not
	// enough peers to determine our public ip address before trying again.
	peerDiscoveryRetryInterval = build.Select(build.Var{
		Standard: 10 * time.Second,
		Dev:      1 * time.Second,
		Testing:  100 * time.Millisecond,
	}).(time.Duration)
)
