package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"

	service "github.com/s02190058/grpc-compare/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	path     = "./server"
	filename = "gopher.png"
)

const chunkSize = 1 << 12 // 4 kB

var port = flag.String("port", "8089", "gRPC server port")

func InternalError() error {
	return status.Errorf(codes.Internal, "internal server error")
}

type ServiceServer struct {
	service.UnimplementedServiceServer
}

func (s ServiceServer) UnaryDownload(_ context.Context, _ *emptypb.Empty) (*service.UnaryDownloadResponse, error) {
	content, err := os.ReadFile(filepath.Join(path, filename))
	if err != nil {
		log.Printf("os.ReadFile: %s", err)
		return nil, InternalError()
	}

	return &service.UnaryDownloadResponse{
		Filename: filename,
		Content:  content,
	}, nil
}
func (s ServiceServer) StreamDownload(_ *emptypb.Empty, stream service.Service_StreamDownloadServer) error {
	file, err := os.Open(filepath.Join(path, filename))
	if err != nil {
		log.Printf("os.Open: %s", err)
		return InternalError()
	}
	buf := make([]byte, chunkSize)
	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
			var n int
			n, err = file.Read(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				} else {
					return InternalError()
				}
			}

			if err = stream.Send(&service.StreamDownloadResponse{
				Filename: filename,
				Chunk: &service.Chunk{
					Data: buf[:n],
				},
			}); err != nil {
				return InternalError()
			}
		}
	}
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", ":"+(*port))
	if err != nil {
		log.Fatalf("can't create listener: %s", err)
	}

	s := grpc.NewServer()
	srv := &ServiceServer{}
	service.RegisterServiceServer(s, srv)
	log.Printf("starting server on %s", *port)
	if err = s.Serve(lis); err != nil {
		log.Fatalf("can't start gPRC server: %s", err)
	}
}
