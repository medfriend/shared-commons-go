package grpc

import "sync"

var (
	securityGRPC string
	mu           sync.Mutex
)

func SetSecurityGRPC(value string) {
	mu.Lock()
	defer mu.Unlock()
	securityGRPC = value
}

func GetSecurityGRPC() string {
	mu.Lock()
	defer mu.Unlock()
	return securityGRPC
}
