package p2p

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/logging"
	"github.com/tendermint/tendermint/libs/protoio"
	tmp2p "github.com/tendermint/tendermint/proto/tendermint/p2p"
	"net"
	"os"
	"runtime/debug"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/p2p/conn"
	"github.com/tendermint/tendermint/pkg/trace"
)

const (
	defaultDialTimeout      = time.Second
	defaultFilterTimeout    = 5 * time.Second
	defaultHandshakeTimeout = 3 * time.Second
)

// IPResolver is a behaviour subset of net.Resolver.
type IPResolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

// accept is the container to carry the upgraded connection and NodeInfo from an
// asynchronously running routine to the Accept method.
type accept struct {
	netAddr  *NetAddress
	conn     quic.Connection
	nodeInfo NodeInfo
	err      error
}

// peerConfig is used to bundle data we need to fully setup a Peer with an
// MConn, provided by the caller of Accept and Dial (currently the Switch). This
// a temporary measure until reactor setup is less dynamic and we introduce the
// concept of PeerBehaviour to communicate about significant Peer lifecycle
// events.
// TODO(xla): Refactor out with more static Reactor setup and PeerBehaviour.
type peerConfig struct {
	chDescs     []*conn.ChannelDescriptor
	onPeerError func(Peer, interface{})
	outbound    bool
	// isPersistent allows you to set a function, which, given socket address
	// (for outbound peers) OR self-reported address (for inbound peers), tells
	// if the peer is persistent or not.
	isPersistent  func(*NetAddress) bool
	reactorsByCh  map[byte]Reactor
	msgTypeByChID map[byte]proto.Message
	metrics       *Metrics
	mlc           *metricsLabelCache
}

// Transport emits and connects to Peers. The implementation of Peer is left to
// the transport. Each transport is also responsible to filter establishing
// peers specific to its domain.
type Transport interface {
	// Listening address.
	NetAddress() NetAddress

	// Accept returns a newly connected Peer.
	Accept(peerConfig) (Peer, error)

	// Dial connects to the Peer for the address.
	Dial(NetAddress, peerConfig) (Peer, error)

	// Cleanup any resources associated with Peer.
	Cleanup(Peer)
}

// transportLifecycle bundles the methods for callers to control start and stop
// behaviour.
type transportLifecycle interface {
	Close() error
	Listen(NetAddress) error
}

// ConnFilterFunc to be implemented by filter hooks after a new connection has
// been established. The set of exisiting connections is passed along together
// with all resolved IPs for the new connection.
type ConnFilterFunc func(ConnSet, quic.Connection, []net.IP) error

// ConnDuplicateIPFilter resolves and keeps all ips for an incoming connection
// and refuses new ones if they come from a known ip.
func ConnDuplicateIPFilter() ConnFilterFunc {
	return func(cs ConnSet, c quic.Connection, ips []net.IP) error {
		for _, ip := range ips {
			if cs.HasIP(ip) {
				return ErrRejected{
					conn:        c,
					err:         fmt.Errorf("ip<%v> already connected", ip),
					isDuplicate: true,
				}
			}
		}

		return nil
	}
}

// MultiplexTransportOption sets an optional parameter on the
// MultiplexTransport.
type MultiplexTransportOption func(*MultiplexTransport)

// MultiplexTransportConnFilters sets the filters for rejection new connections.
func MultiplexTransportConnFilters(
	filters ...ConnFilterFunc,
) MultiplexTransportOption {
	return func(mt *MultiplexTransport) { mt.connFilters = filters }
}

// MultiplexTransportFilterTimeout sets the timeout waited for filter calls to
// return.
func MultiplexTransportFilterTimeout(
	timeout time.Duration,
) MultiplexTransportOption {
	return func(mt *MultiplexTransport) { mt.filterTimeout = timeout }
}

// MultiplexTransportResolver sets the Resolver used for ip lokkups, defaults to
// net.DefaultResolver.
func MultiplexTransportResolver(resolver IPResolver) MultiplexTransportOption {
	return func(mt *MultiplexTransport) { mt.resolver = resolver }
}

// MultiplexTransportMaxIncomingConnections sets the maximum number of
// simultaneous connections (incoming). Default: 0 (unlimited)
func MultiplexTransportMaxIncomingConnections(n int) MultiplexTransportOption {
	return func(mt *MultiplexTransport) { mt.maxIncomingConnections = n }
}

// MultiplexTransport accepts and dials tcp connections and upgrades them to
// multiplexed peers.
type MultiplexTransport struct {
	netAddr                NetAddress
	listener               *quic.Listener
	maxIncomingConnections int // see MaxIncomingConnections

	acceptc chan accept
	closec  chan struct{}

	// Lookup table for duplicate ip and id checks.
	conns       ConnSet
	connFilters []ConnFilterFunc

	dialTimeout      time.Duration
	filterTimeout    time.Duration
	handshakeTimeout time.Duration
	nodeInfo         NodeInfo
	nodeKey          NodeKey
	resolver         IPResolver

	// the tracer is passed to peers for collecting trace data
	tracer trace.Tracer
}

// Test multiplexTransport for interface completeness.
var _ Transport = (*MultiplexTransport)(nil)
var _ transportLifecycle = (*MultiplexTransport)(nil)

// NewMultiplexTransport returns a tcp connected multiplexed peer.
func NewMultiplexTransport(
	nodeInfo NodeInfo,
	nodeKey NodeKey,
	tracer trace.Tracer,
) *MultiplexTransport {
	return &MultiplexTransport{
		acceptc:          make(chan accept),
		closec:           make(chan struct{}),
		dialTimeout:      defaultDialTimeout,
		filterTimeout:    defaultFilterTimeout,
		handshakeTimeout: defaultHandshakeTimeout,
		nodeInfo:         nodeInfo,
		nodeKey:          nodeKey,
		conns:            NewConnSet(),
		resolver:         net.DefaultResolver,
		tracer:           tracer,
	}
}

// NetAddress implements Transport.
func (mt *MultiplexTransport) NetAddress() NetAddress {
	return mt.netAddr
}

// Accept implements Transport.
func (mt *MultiplexTransport) Accept(cfg peerConfig) (Peer, error) {
	select {
	// This case should never have any side-effectful/blocking operations to
	// ensure that quality peers are ready to be used.
	case a := <-mt.acceptc:
		if a.err != nil {
			return nil, a.err
		}

		cfg.outbound = false

		return mt.wrapPeer(a.conn, a.nodeInfo, cfg, a.netAddr), nil
	case <-mt.closec:
		return nil, ErrTransportClosed{}
	}
}

// Dial implements Transport.
func (mt *MultiplexTransport) Dial(
	addr NetAddress,
	cfg peerConfig,
) (Peer, error) {
	tlsConfig, err := NewTLSConfig(mt.nodeKey.PrivKey)
	if err != nil {
		return nil, err
	}
	tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) (err error) {
		defer func() {
			if rerr := recover(); rerr != nil {
				fmt.Fprintf(os.Stderr, "panic when processing peer certificate in TLS exchangeNodeInfo: %s\n%s\n", rerr, debug.Stack())
				err = fmt.Errorf("panic when processing peer certificate in TLS exchangeNodeInfo: %s", rerr)
			}
		}()

		if len(rawCerts) != 1 {
			return errors.New("expected one certificates in the chain")
		}
		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return err
		}

		pubKey, err := VerifyCertificate(cert)
		if err != nil {
			return err
		}
		remoteID := PubKeyToID(pubKey)
		if remoteID != addr.ID {
			return fmt.Errorf("mismatch peer ID")
		}
		return nil
	}

	c, err := addr.DialTimeout(mt.dialTimeout, tlsConfig)
	if err != nil {
		return nil, err
	}

	// TODO(xla): Evaluate if we should apply filters if we explicitly dial.
	if err := mt.filterConn(c); err != nil {
		return nil, err
	}

	_, nodeInfo, err := mt.getNodeInfo(c)
	if err != nil {
		return nil, err
	}

	cfg.outbound = true

	p := mt.wrapPeer(c, nodeInfo, cfg, &addr)

	return p, nil
}

// Close implements transportLifecycle.
func (mt *MultiplexTransport) Close() error {
	close(mt.closec)

	if mt.listener != nil {
		return mt.listener.Close()
	}

	return nil
}

// Listen implements transportLifecycle.
func (mt *MultiplexTransport) Listen(addr NetAddress) error {
	tlsConfig, err := NewTLSConfig(mt.nodeKey.PrivKey)
	if err != nil {
		return err
	}
	tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) (err error) {
		defer func() {
			if rerr := recover(); rerr != nil {
				fmt.Fprintf(os.Stderr, "panic when processing peer certificate in TLS exchangeNodeInfo: %s\n%s\n", rerr, debug.Stack())
				err = fmt.Errorf("panic when processing peer certificate in TLS exchangeNodeInfo: %s", rerr)
			}
		}()

		if len(rawCerts) != 1 {
			return errors.New("expected one certificates in the chain")
		}
		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return err
		}

		_, err = VerifyCertificate(cert)
		if err != nil {
			return err
		}
		return nil
	}
	quickConfig := quic.Config{
		// TODO(rach-id): do we want to enable 0RTT? are the replay risks fine?
		Allow0RTT:             false,
		MaxIdleTimeout:        time.Minute,
		MaxIncomingStreams:    10000,
		MaxIncomingUniStreams: 10000,
		KeepAlivePeriod:       10 * time.Second,
		EnableDatagrams:       true,
		Tracer: func(ctx context.Context, perspective logging.Perspective, id quic.ConnectionID) *logging.ConnectionTracer {
			return logging.NewMultiplexedConnectionTracer(GetNewTracer())
		},
	}
	listener, err := quic.ListenAddr(addr.DialString(), tlsConfig, &quickConfig)
	if err != nil {
		return err
	}
	mt.listener = listener
	mt.netAddr = addr

	// TODO(rach-id): use a better context
	go mt.acceptPeers(context.Background())

	return nil
}

func GetNewTracer() *logging.ConnectionTracer {
	return &logging.ConnectionTracer{
		StartedConnection: func(local, remote net.Addr, srcConnID, destConnID quic.ConnectionID) {
			fmt.Println(fmt.Sprintf("StartedConnection: local=%v, remote=%v, srcConnID=%v, destConnID=%v", local, remote, srcConnID, destConnID))
		},
		NegotiatedVersion: func(chosen quic.Version, clientVersions, serverVersions []quic.Version) {
			//fmt.Println(fmt.Sprintf("NegotiatedVersion: chosen=%v, clientVersions=%v, serverVersions=%v", chosen, clientVersions, serverVersions))
		},
		ClosedConnection: func(err error) {
			fmt.Println(fmt.Sprintf("ClosedConnection: error=%v", err))
		},
		SentTransportParameters: func(params *logging.TransportParameters) {
			//fmt.Println(fmt.Sprintf("SentTransportParameters: params=%v", params))
		},
		ReceivedTransportParameters: func(params *logging.TransportParameters) {
			//fmt.Println(fmt.Sprintf("ReceivedTransportParameters: params=%v", params))
		},
		RestoredTransportParameters: func(params *logging.TransportParameters) {
			//fmt.Println(fmt.Sprintf("RestoredTransportParameters: params=%v", params))
		},
		SentLongHeaderPacket: func(header *logging.ExtendedHeader, byteCount logging.ByteCount, ecn logging.ECN, ack *logging.AckFrame, frames []logging.Frame) {
			//fmt.Println(fmt.Sprintf("SentLongHeaderPacket: header=%v, byteCount=%v, ecn=%v, ack=%v, frames=%v", header, byteCount, ecn, ack, frames))
		},
		SentShortHeaderPacket: func(header *logging.ShortHeader, byteCount logging.ByteCount, ecn logging.ECN, ack *logging.AckFrame, frames []logging.Frame) {
			//fmt.Println(fmt.Sprintf("SentShortHeaderPacket: header=%v, byteCount=%v, ecn=%v, ack=%v, frames=%v", header, byteCount, ecn, ack, frames))
		},
		ReceivedVersionNegotiationPacket: func(dest, src logging.ArbitraryLenConnectionID, versions []logging.Version) {
			//fmt.Println(fmt.Sprintf("ReceivedVersionNegotiationPacket: dest=%v, src=%v, versions=%v", dest, src, versions))
		},
		ReceivedRetry: func(header *logging.Header) {
			fmt.Println(fmt.Sprintf("ReceivedRetry: header=%v", header))
		},
		ReceivedLongHeaderPacket: func(header *logging.ExtendedHeader, byteCount logging.ByteCount, ecn logging.ECN, frames []logging.Frame) {
			//fmt.Println(fmt.Sprintf("ReceivedLongHeaderPacket: header=%v, byteCount=%v, ecn=%v, frames=%v", header, byteCount, ecn, frames))
		},
		ReceivedShortHeaderPacket: func(header *logging.ShortHeader, byteCount logging.ByteCount, ecn logging.ECN, frames []logging.Frame) {
			//fmt.Println(fmt.Sprintf("ReceivedShortHeaderPacket: header=%v, byteCount=%v, ecn=%v, frames=%v", header, byteCount, ecn, frames))
		},
		BufferedPacket: func(packetType logging.PacketType, byteCount logging.ByteCount) {
			//fmt.Println(fmt.Sprintf("BufferedPacket: packetType=%v, byteCount=%v", packetType, byteCount))
		},
		DroppedPacket: func(packetType logging.PacketType, packetNum logging.PacketNumber, byteCount logging.ByteCount, reason logging.PacketDropReason) {
			fmt.Println(fmt.Sprintf("DroppedPacket: packetType=%v, packetNum=%v, byteCount=%v, reason=%v", packetType, packetNum, byteCount, reason))
		},
		UpdatedMetrics: func(rttStats *logging.RTTStats, cwnd, bytesInFlight logging.ByteCount, packetsInFlight int) {
			//fmt.Println(fmt.Sprintf("UpdatedMetrics: rttStats=%v, cwnd=%v, bytesInFlight=%v, packetsInFlight=%v", rttStats, cwnd, bytesInFlight, packetsInFlight))
		},
		AcknowledgedPacket: func(encLevel logging.EncryptionLevel, packetNum logging.PacketNumber) {
			//fmt.Println(fmt.Sprintf("AcknowledgedPacket: encLevel=%v, packetNum=%v", encLevel, packetNum))
		},
		LostPacket: func(encLevel logging.EncryptionLevel, packetNum logging.PacketNumber, reason logging.PacketLossReason) {
			fmt.Println(fmt.Sprintf("LostPacket: encLevel=%v, packetNum=%v, reason=%v", encLevel, packetNum, reason))
		},
		UpdatedMTU: func(mtu logging.ByteCount, done bool) {
			//fmt.Println(fmt.Sprintf("UpdatedMTU: mtu=%v, done=%v", mtu, done))
		},
		UpdatedCongestionState: func(state logging.CongestionState) {
			//fmt.Println(fmt.Sprintf("UpdatedCongestionState: state=%v", state))
		},
		UpdatedPTOCount: func(value uint32) {
			//fmt.Println(fmt.Sprintf("UpdatedPTOCount: value=%v", value))
		},
		UpdatedKeyFromTLS: func(encLevel logging.EncryptionLevel, perspective logging.Perspective) {
			//fmt.Println(fmt.Sprintf("UpdatedKeyFromTLS: encLevel=%v, perspective=%v", encLevel, perspective))
		},
		UpdatedKey: func(phase logging.KeyPhase, remote bool) {
			//fmt.Println(fmt.Sprintf("UpdatedKey: phase=%v, remote=%v", phase, remote))
		},
		DroppedEncryptionLevel: func(encLevel logging.EncryptionLevel) {
			//fmt.Println(fmt.Sprintf("DroppedEncryptionLevel: encLevel=%v", encLevel))
		},
		DroppedKey: func(phase logging.KeyPhase) {
			//fmt.Println(fmt.Sprintf("DroppedKey: phase=%v", phase))
		},
		SetLossTimer: func(timerType logging.TimerType, encLevel logging.EncryptionLevel, t time.Time) {
			//fmt.Println(fmt.Sprintf("SetLossTimer: timerType=%v, encLevel=%v, time=%v", timerType, encLevel, t))
		},
		LossTimerExpired: func(timerType logging.TimerType, encLevel logging.EncryptionLevel) {
			//fmt.Println(fmt.Sprintf("LossTimerExpired: timerType=%v, encLevel=%v", timerType, encLevel))
		},
		LossTimerCanceled: func() {
			//fmt.Println(fmt.Sprintf("LossTimerCanceled"))
		},
		ECNStateUpdated: func(state logging.ECNState, trigger logging.ECNStateTrigger) {
			//fmt.Println(fmt.Sprintf("ECNStateUpdated: state=%v, trigger=%v", state, trigger))
		},
		ChoseALPN: func(protocol string) {
			//fmt.Println(fmt.Sprintf("ChoseALPN: protocol=%v", protocol))
		},
		Close: func() {
			fmt.Println(fmt.Sprintf("Close"))
		},
		Debug: func(name, msg string) {
			fmt.Println(fmt.Sprintf("Debug: name=%v, msg=%v", name, msg))
		},
	}
}

// AddChannel registers a channel to nodeInfo.
// NOTE: NodeInfo must be of type DefaultNodeInfo else channels won't be updated
// This is a bit messy at the moment but is cleaned up in the following version
// when NodeInfo changes from an interface to a concrete type
func (mt *MultiplexTransport) AddChannel(chID byte) {
	if ni, ok := mt.nodeInfo.(DefaultNodeInfo); ok {
		if !ni.HasChannel(chID) {
			ni.Channels = append(ni.Channels, chID)
		}
		mt.nodeInfo = ni
	}
}

func (mt *MultiplexTransport) acceptPeers(ctx context.Context) {
	for {
		c, err := mt.listener.Accept(ctx)
		if err != nil {
			// If Close() has been called, silently exit.
			select {
			case _, ok := <-mt.closec:
				if !ok {
					return
				}
			default:
				// Transport is not closed
			}

			mt.acceptc <- accept{err: err}
			return
		}

		// Connection upgrade and filtering should be asynchronous to avoid
		// Head-of-line blocking[0].
		// Reference:  https://github.com/tendermint/tendermint/issues/2047
		//
		// [0] https://en.wikipedia.org/wiki/Head-of-line_blocking
		go func(c quic.Connection) {
			defer func() {
				if r := recover(); r != nil {
					err := ErrRejected{
						conn:          c,
						err:           fmt.Errorf("recovered from panic: %v", r),
						isAuthFailure: true,
					}
					select {
					case mt.acceptc <- accept{err: err}:
					case <-mt.closec:
						// Give up if the transport was closed.
						// TODO(rach-id): valid error code
						_ = c.CloseWithError(quic.ApplicationErrorCode(http3.ErrCodeConnectError), err.Error())
						return
					}
				}
			}()

			var (
				nodeInfo NodeInfo
				netAddr  *NetAddress
			)

			err := mt.filterConn(c)
			if err == nil {
				_, nodeInfo, err = mt.getNodeInfo(c)
				if err == nil {
					addr := c.RemoteAddr()
					//id := PubKeyToID(mt.nodeKey.PubKey())
					netAddr = NewUDPNetAddress(nodeInfo.ID(), addr)
				}
			}

			select {
			case mt.acceptc <- accept{netAddr, c, nodeInfo, err}:
				// Make the upgraded peer available.
			case <-mt.closec:
				// Give up if the transport was closed.
				_ = c.CloseWithError(quic.ApplicationErrorCode(http3.ErrCodeConnectError), "closes transport")
				return
			}
		}(c)
	}
}

// Cleanup removes the given address from the connections set and
// closes the connection.
func (mt *MultiplexTransport) Cleanup(p Peer) {
	mt.conns.RemoveAddr(p.RemoteAddr())
	_ = p.CloseConn()
}

func (mt *MultiplexTransport) cleanup(c quic.Connection) error {
	mt.conns.Remove(c)

	// TODO(rach-id): valid error
	return c.CloseWithError(quic.ApplicationErrorCode(quic.NoError), "closing for cleanup")
}

func (mt *MultiplexTransport) filterConn(c quic.Connection) (err error) {
	defer func() {
		if err != nil {
			// TODO(rach-id): valid error
			_ = c.CloseWithError(quic.ApplicationErrorCode(http3.ErrCodeConnectError), err.Error())
		}
	}()

	// Reject if connection is already present.
	if mt.conns.Has(c) {
		return ErrRejected{conn: c, isDuplicate: true}
	}

	// Resolve ips for incoming conn.
	ips, err := resolveIPs(mt.resolver, c)
	if err != nil {
		return err
	}

	errc := make(chan error, len(mt.connFilters))

	for _, f := range mt.connFilters {
		go func(f ConnFilterFunc, c quic.Connection, ips []net.IP, errc chan<- error) {
			errc <- f(mt.conns, c, ips)
		}(f, c, ips, errc)
	}

	for i := 0; i < cap(errc); i++ {
		select {
		case err := <-errc:
			if err != nil {
				return ErrRejected{conn: c, err: err, isFiltered: true}
			}
		case <-time.After(mt.filterTimeout):
			return ErrFilterTimeout{}
		}

	}

	mt.conns.Set(c, ips)

	return nil
}

func (mt *MultiplexTransport) getNodeInfo(c quic.Connection) (conn quic.Connection, remoteNodeInfo NodeInfo, err error) {
	defer func() {
		if err != nil {
			_ = mt.cleanup(c)
		}
	}()

	remoteNodeInfo, err = exchangeNodeInfo(c, mt.handshakeTimeout, mt.nodeInfo)
	if err != nil {
		return nil, nil, ErrRejected{
			conn:          c,
			err:           fmt.Errorf("exchangeNodeInfo failed: %v", err),
			isAuthFailure: true,
		}
	}

	if err := remoteNodeInfo.Validate(); err != nil {
		return nil, nil, ErrRejected{
			conn:              c,
			err:               err,
			isNodeInfoInvalid: true,
		}
	}

	// Reject self.
	if mt.nodeInfo.ID() == remoteNodeInfo.ID() {
		return nil, nil, ErrRejected{
			addr:   *NewUDPNetAddress(remoteNodeInfo.ID(), c.RemoteAddr()),
			conn:   c,
			id:     remoteNodeInfo.ID(),
			isSelf: true,
		}
	}

	if err := mt.nodeInfo.CompatibleWith(remoteNodeInfo); err != nil {
		return nil, nil, ErrRejected{
			conn:           c,
			err:            err,
			id:             remoteNodeInfo.ID(),
			isIncompatible: true,
		}
	}

	return c, remoteNodeInfo, nil
}

func exchangeNodeInfo(
	c quic.Connection,
	timeout time.Duration,
	nodeInfo NodeInfo,
) (NodeInfo, error) {
	var (
		errc = make(chan error, 2)

		pbpeerNodeInfo tmp2p.DefaultNodeInfo
		peerNodeInfo   DefaultNodeInfo
		ourNodeInfo    = nodeInfo.(DefaultNodeInfo)
	)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func(errc chan<- error, c quic.Connection) {
		stream, err := c.OpenStreamSync(ctx)
		if err != nil {
			errc <- err
			return
		}
		_, err = protoio.NewDelimitedWriter(stream).WriteMsg(ourNodeInfo.ToProto())
		errc <- err
	}(errc, c)
	go func(errc chan<- error, c quic.Connection) {
		stream, err := c.AcceptStream(ctx)
		if err != nil {
			errc <- err
			return
		}
		protoReader := protoio.NewDelimitedReader(stream, MaxNodeInfoSize())
		_, err = protoReader.ReadMsg(&pbpeerNodeInfo)
		errc <- err
	}(errc, c)

	for i := 0; i < cap(errc); i++ {
		err := <-errc
		if err != nil {
			return nil, err
		}
	}

	peerNodeInfo, err := DefaultNodeInfoFromToProto(&pbpeerNodeInfo)
	if err != nil {
		return nil, err
	}

	return peerNodeInfo, nil
}

func (mt *MultiplexTransport) wrapPeer(
	c quic.Connection,
	ni NodeInfo,
	cfg peerConfig,
	socketAddr *NetAddress,
) Peer {

	persistent := false
	if cfg.isPersistent != nil {
		if cfg.outbound {
			persistent = cfg.isPersistent(socketAddr)
		} else {
			selfReportedAddr, err := ni.NetAddress()
			if err == nil {
				persistent = cfg.isPersistent(selfReportedAddr)
			}
		}
	}

	peerConn := newPeerConn(
		cfg.outbound,
		persistent,
		c,
		socketAddr,
	)

	p := newPeer(
		peerConn,
		ni,
		cfg.reactorsByCh,
		cfg.msgTypeByChID,
		cfg.mlc,
		cfg.onPeerError,
		PeerMetrics(cfg.metrics),
		WithPeerTracer(mt.tracer),
	)

	return p
}

func resolveIPs(resolver IPResolver, c quic.Connection) ([]net.IP, error) {
	host, _, err := net.SplitHostPort(c.RemoteAddr().String())
	if err != nil {
		return nil, err
	}

	addrs, err := resolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return nil, err
	}

	ips := []net.IP{}

	for _, addr := range addrs {
		ips = append(ips, addr.IP)
	}

	return ips, nil
}
