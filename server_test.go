package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"testing"
	"time"

	service "github.com/s02190058/grpc-compare/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

func newServer(b *testing.B, register func(s *grpc.Server)) *grpc.ClientConn {
	lis := bufconn.Listen(1024 * 1024)
	b.Cleanup(func() {
		_ = lis.Close()
	})

	s := grpc.NewServer()
	b.Cleanup(func() {
		s.Stop()
	})

	register(s)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server.Serve: %v", err)
		}
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	b.Cleanup(func() {
		cancel()
	})

	conn, err := grpc.DialContext(
		ctx,
		"",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		b.Fatalf("grpc.DialContext: %v", err)
	}

	return conn
}

func BenchmarkServiceServer_UnaryDownload(b *testing.B) {
	srv := ServiceServer{}
	conn := newServer(b, func(s *grpc.Server) {
		service.RegisterServiceServer(s, srv)
	})

	client := service.NewServiceClient(conn)

	for i := 0; i < b.N; i++ {
		res, err := client.UnaryDownload(context.Background(), &emptypb.Empty{})
		if err != nil {
			b.Fatalf("ServiceClient.UnaryDownload: %v", err)
		}

		file, err := os.Create(filename)
		if err != nil {
			b.Fatalf("os.Create: %v", err)
		}

		if _, err = file.Write(res.Content); err != nil {
			b.Fatalf("File.Write: %v", err)
		}

		if err = os.Remove(filename); err != nil {
			b.Fatalf("os.Remove: %v", err)
		}
	}
}

func BenchmarkServiceServer_StreamDownload(b *testing.B) {
	srv := ServiceServer{}
	conn := newServer(b, func(s *grpc.Server) {
		service.RegisterServiceServer(s, srv)
	})

	client := service.NewServiceClient(conn)

	for i := 0; i < b.N; i++ {
		stream, err := client.StreamDownload(context.Background(), &emptypb.Empty{})
		if err != nil {
			b.Fatalf("ServiceClient.StreamDownload: %v", err)
		}

		file, err := os.Create(filename)
		if err != nil {
			b.Fatalf("os.Create: %v", err)
		}

		for {
			res, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				b.Fatalf("Service_StreamDownloadClient.Recv: %v", err)
			}

			if _, err = file.Write(res.Chunk.Data); err != nil {
				b.Fatalf("File.Write: %v", err)
			}
		}

		if err = os.Remove(filename); err != nil {
			b.Fatalf("os.Remove: %v", err)
		}
	}
}
