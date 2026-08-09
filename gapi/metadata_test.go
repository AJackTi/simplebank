package gapi

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func TestExtractMetadataPrefersForwardedClientIP(t *testing.T) {
	server := &Server{}

	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(
			grpcGatewayUserAgentHeader, "grpc-gateway",
			xForwardedForHeader, "203.0.113.10, 10.0.0.1",
		),
	)
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345},
	})

	mtdt := server.extractMetadata(ctx)

	require.Equal(t, "grpc-gateway", mtdt.UserAgent)
	require.Equal(t, "203.0.113.10", mtdt.ClientIP)
}

func TestExtractMetadataFallsBackToPeerClientIP(t *testing.T) {
	server := &Server{}

	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345},
	})

	mtdt := server.extractMetadata(ctx)

	require.Equal(t, "127.0.0.1:12345", mtdt.ClientIP)
}
