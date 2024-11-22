package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	health "github.com/hex0punk/cont-flood-poc/proto" // Change this import path based on your project structure
)

var port = 8443

type healthServer struct {
	health.UnimplementedHealthServiceServer
}

func (s *healthServer) Check(ctx context.Context, req *health.HealthCheckRequest) (*health.HealthCheckResponse, error) {
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil, fmt.Errorf("error getting process info: %w", err)
	}
	cpuPercent, err := proc.Percent(time.Second)
	if err != nil {
		return nil, fmt.Errorf("error retrieving process CPU usage: %w", err)
	}
	vMem, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("error retrieving virtual memory usage: %w", err)
	}
	return &health.HealthCheckResponse{
		CpuUsagePercent:    float32(cpuPercent),
		MemoryUsagePercent: float32(vMem.UsedPercent),
	}, nil
}

func printUsage(interval time.Duration) {

	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		fmt.Println("Error getting process info:", err)
		return
	}

	for {
		percent, err := proc.Percent(interval)
		if err != nil {
			fmt.Printf("Error retrieving process CPU usage: %s\r", err)
			continue
		}

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		fmt.Printf("Process CPU Usage: %.2f%%", percent)
		fmt.Printf(" Alloc = %v MiB", m.Alloc/1024/1024)
		fmt.Printf(" TotalAlloc = %v MiB", m.TotalAlloc/1024/1024)
		fmt.Printf(" Sys = %v MiB", m.Sys/1024/1024)
		fmt.Printf(" NumGC = %v\n", m.NumGC)
		time.Sleep(interval) // Ensure it waits for the specified interval
	}
}

func main() {
	creds, err := credentials.NewServerTLSFromFile("./certs/server.crt", "./certs/server.key")
	if err != nil {
		log.Fatalf("failed to create credentials: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer(grpc.Creds(creds))
	reflection.Register(server)

	healthSvc := &healthServer{}
	health.RegisterHealthServiceServer(server, healthSvc)

	go printUsage(200 * time.Millisecond)

	log.Printf("Starting gRPC server on port %d", port)
	server.Serve(lis)
}
