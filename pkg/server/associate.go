package server

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/HT4w5/socks5/pkg/method"
	"github.com/HT4w5/socks5/pkg/payload"
)

const (
	udpResolveTimeout = 10 * time.Second
)

func (s *Server) handleUDPAssociate(ctx context.Context, conn net.Conn, request *payload.Request, handler method.MethodHandler) {
	// Set up UDP listen
	addrPort, err := netip.ParseAddrPort(conn.LocalAddr().String())
	if err != nil {
		s.logger.Errorf("failed to parse local endpoint address: %v", err)
		s.sendFailureReply(conn, payload.ServerFailure)
		return
	}

	inPc, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   addrPort.Addr().AsSlice(),
		Port: 0, // Use system allocated port
		Zone: addrPort.Addr().Zone(),
	})
	if err != nil {
		s.logger.Errorf("udp listen packet failed: %v", err)
		s.sendFailureReply(conn, payload.ServerFailure)
		return
	}

	s.logger.Debugf("inbound listening on udp://%s", inPc.LocalAddr().String())

	defer func() {
		if err := inPc.Close(); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				s.logger.Warnf("failed to close inbound udp packet connection: %v", err)
			}
		}
	}()

	// Set up outbound UDP socket
	outPc, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   nil,
		Port: 0,
	})
	if err != nil {
		s.logger.Errorf("udp listen packet failed: %v", err)
		s.sendFailureReply(conn, payload.ServerFailure)
		return
	}

	s.logger.Debugf("outbound listening on udp://%s", outPc.LocalAddr().String())

	defer func() {
		if err := outPc.Close(); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				s.logger.Warnf("failed to close outbound udp packet connection: %v", err)
			}
		}
	}()

	// Send success reply
	udpEndpoint, err := netip.ParseAddrPort(inPc.LocalAddr().String())
	if err != nil {
		s.logger.Errorf("failed to parse local udp endpoint address: %v", err)
		s.sendFailureReply(conn, payload.ServerFailure)
		return
	}
	r := payload.NewReply(
		payload.ReplyWithRep(payload.Succeeded),
		payload.ReplyWithIP(udpEndpoint.Addr()),
		payload.ReplyWithPort(udpEndpoint.Port()),
	)

	s.logger.Debugf("sending reply: %s", r.String())

	if err := r.Write(conn); err != nil {
		s.logger.Errorf("failed to write reply: %v", err)
		return
	}

	ctx = contextFromConn(ctx, conn)

	// Closer
	stop := context.AfterFunc(ctx, func() {
		if err := inPc.Close(); err != nil {
			s.logger.Warnf("failed to close inbound udp packet connection")
		}
		if err := outPc.Close(); err != nil {
			s.logger.Warnf("failed to close outbound udp packet connection")
		}
	})
	defer stop()

	// Set up endpoint filter
	var filter func(netip.AddrPort) bool
	var append func(netip.AddrPort)
	var addrSet map[netip.Addr]struct{}
	var addrPortSet map[netip.AddrPort]struct{}
	var mutex sync.Mutex
	switch s.udpNATBehavior {
	case EndpointIndependent:
		filter = func(ap netip.AddrPort) bool { return true }
		append = func(ap netip.AddrPort) {}
	case AddressDependent:
		addrSet = s.udpAddrSetPool.Get()
		defer s.udpAddrSetPool.Put(addrSet)
		filter = func(ap netip.AddrPort) bool {
			mutex.Lock()
			defer mutex.Unlock()
			_, ok := addrSet[ap.Addr().Unmap()]
			return ok
		}
		append = func(ap netip.AddrPort) {
			mutex.Lock()
			defer mutex.Unlock()
			addrSet[ap.Addr().Unmap()] = struct{}{}
		}
	case AddressAndPortDependent:
		addrPortSet = s.udpAddrPortSetPool.Get()
		defer s.udpAddrPortSetPool.Put(addrPortSet)
		filter = func(ap netip.AddrPort) bool {
			mutex.Lock()
			defer mutex.Unlock()
			_, ok := addrPortSet[netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port())]
			return ok
		}
		append = func(ap netip.AddrPort) {
			mutex.Lock()
			defer mutex.Unlock()
			addrPortSet[netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port())] = struct{}{}
		}
	}

	var clientEndpoint netip.AddrPort
	ready := make(chan struct{})
	if !(request.ATyp == payload.FQDNAddr || request.DstAddr.IsUnspecified() || request.DstPort == 0) {
		clientEndpoint = netip.AddrPortFrom(request.DstAddr, request.DstPort)
		close(ready)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		s.udpRelayInbound(ctx, ready, &clientEndpoint, filter, handler, outPc, inPc)
	})
	wg.Go(func() {
		s.udpRelayOutbound(ready, &clientEndpoint, append, handler, outPc, inPc)
	})
	wg.Wait()
}

// Send packets from the socks client to outbound
func (s *Server) udpRelayOutbound(ready chan<- struct{}, clientEndpoint *netip.AddrPort, append func(netip.AddrPort), handler method.MethodHandler, outPc *net.UDPConn, inPc *net.UDPConn) {
	buf := s.udpBytePool.Get()
	defer s.udpBytePool.Put(buf)

	var once sync.Once
	for {
		n, udpEndpoint, err := inPc.ReadFromUDPAddrPort(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			s.logger.Warnf("failed to read from UDP packet connection: %v", err)
			continue
		}

		once.Do(func() {
			if clientEndpoint.IsValid() {
				return
			}
			*clientEndpoint = udpEndpoint
			close(ready)
		})

		if udpEndpoint != *clientEndpoint {
			// Drop packet from invalid sender
			continue
		}

		var udpRequest payload.UDPRequest
		err = udpRequest.Parse(handler.UntransformDatagram(buf[:n]))
		if err != nil {
			s.logger.Warnf("failed to parse udp request: %v", err)
			continue
		}

		s.logger.Debugf("received udp request: %s", udpRequest.String())

		if udpRequest.Frag != payload.NoFrag {
			// TODO: implement fragmentation
			continue
		}

		// Resolve fqdn
		ctx, _ := context.WithTimeout(context.Background(), udpResolveTimeout)
		if udpRequest.ATyp == payload.FQDNAddr {
			udpRequest.DstAddr, err = s.res.Resolve(ctx, string(udpRequest.DstFQDN))
			if err != nil {
				s.logger.Warnf("failed to resolve dst fqdn: %v", err)
				continue
			}
		}

		dstAddrPort := netip.AddrPortFrom(udpRequest.DstAddr, udpRequest.DstPort)
		// Append to endpoint filter set
		append(dstAddrPort)

		// Send to outbound
		if _, err := outPc.WriteToUDPAddrPort(udpRequest.Data, dstAddrPort); err != nil {
			s.logger.Warnf("failed to send udp packet: %v", err)
		}
	}
}

const (
	udpHeaderOffset = 22 // Assume IPv4/IPv6 only
)

// Send incoming packets from outbound to the client
func (s *Server) udpRelayInbound(ctx context.Context, ready <-chan struct{}, clientEndpoint *netip.AddrPort, filter func(netip.AddrPort) bool, handler method.MethodHandler, outPc *net.UDPConn, inPc *net.UDPConn) {
	// Wait until the outbound relay has captured clientEndpoint
	select {
	case <-ready:
	case <-ctx.Done():
		return
	}

	buf := s.udpBytePool.Get()
	defer s.udpBytePool.Put(buf)

	for {
		n, udpEndpoint, err := outPc.ReadFromUDPAddrPort(buf[udpHeaderOffset:])
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			s.logger.Warnf("failed to read from UDP packet connection: %v", err)
			continue
		}

		s.logger.Debugf("received packet from udp://%s", udpEndpoint.String())

		if !filter(udpEndpoint) {
			// Drop packet from unknown endpoint
			continue
		}

		var atyp uint8
		var offset int
		if udpEndpoint.Addr().Is4() {
			atyp = payload.IPv4Addr
			offset = 12
		} else {
			atyp = payload.IPv6Addr
			offset = 0
		}

		udpRequestHeader := payload.UDPRequest{
			Frag:    payload.NoFrag,
			ATyp:    atyp,
			DstAddr: udpEndpoint.Addr(),
			DstPort: udpEndpoint.Port(),
		}

		s.logger.Debugf("sending udp request: %s", udpRequestHeader.String())

		if err := udpRequestHeader.Write(buf[offset:]); err != nil {
			s.logger.Warnf("failed to write udp request header: %v", err)
			continue
		}

		if _, err := inPc.WriteToUDPAddrPort(handler.TransformDatagram(buf[offset:udpHeaderOffset+n]), *clientEndpoint); err != nil {
			s.logger.Warnf("failed to send udp packet: %v", err)
		}
	}
}
