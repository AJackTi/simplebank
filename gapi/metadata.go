package gapi

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	grpcGatewayUserAgentHeader = "grpcgateway-user-agent"
	xForwardedForHeader        = "x-forwarded-for"
)

type Metadata struct {
	UserAgent string
	ClientIP  string
}

func (server *Server) extractMetadata(ctx context.Context) *Metadata {
	mtdt := &Metadata{}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if userAgents := md.Get(grpcGatewayUserAgentHeader); len(userAgents) > 0 {
			mtdt.UserAgent = userAgents[0]
		}

		if clientIPs := md.Get(xForwardedForHeader); len(clientIPs) > 0 {
			mtdt.ClientIP = firstForwardedClientIP(clientIPs[0])
		}
	}

	if mtdt.ClientIP == "" {
		if p, ok := peer.FromContext(ctx); ok {
			mtdt.ClientIP = p.Addr.String()
		}
	}

	return mtdt
}

func firstForwardedClientIP(header string) string {
	clientIP, _, _ := strings.Cut(header, ",")
	return strings.TrimSpace(clientIP)
}
