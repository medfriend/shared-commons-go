package grpc

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// GRPCClient es un singleton que administra una conexión gRPC resiliente.
type GRPCClient struct {
	conn   *grpc.ClientConn
	target string
	mu     sync.Mutex
	logger *zap.Logger
	// Opciones para el dialer resiliente
	optFns []func(*ClientOptions)
}

var (
	grpcInstance *GRPCClient
	grpcOnce     sync.Once
)

// GetGRPCInstances devuelve una instancia única de GRPCClient.
// Si target está vacío, intentará usar util.GetSecurityGRPC(), luego la env "GRPC_TARGET",
// y por último un valor por defecto "localhost:9041".
func GetGRPCInstances(target string, logger *zap.Logger, optFns ...func(*ClientOptions)) *GRPCClient {
	grpcOnce.Do(func() {
		if target == "security" {
			target = GetSecurityGRPC()
		}
		grpcInstance = &GRPCClient{target: target, logger: logger, optFns: optFns}
		// dial inicial bloqueante con timeout para fallar rápido en arranque
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		grpcInstance.connect(ctx)
	})
	return grpcInstance
}

// connect establece (o restablece) la conexión gRPC usando el dialer resiliente.
func (g *GRPCClient) connect(ctx context.Context) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Cerrar anterior si existe
	if g.conn != nil {
		_ = g.conn.Close()
		g.conn = nil
	}

	// Dial resiliente (usa insecure por defecto si no se configuran credenciales)
	if g.logger != nil {
		g.logger.Info("grpc dialing", zap.String("target", g.target))
	}
	conn, err := DialResilient(ctx, g.target, g.logger, g.optFns...)
	if err != nil {
		// No hacemos panic aquí para permitir reintentos posteriores vía ensureConnection
		if g.logger != nil {
			g.logger.Warn("grpc dial failed", zap.String("target", g.target), zap.Error(err))
		}
		return
	}
	g.conn = conn
	if g.logger != nil {
		g.logger.Info("grpc dial ok", zap.String("target", g.target))
	}
}

// ensureConnection verifica y restablece la conexión si está ausente o en estado Shutdown.
func (g *GRPCClient) ensureConnection() {
	g.mu.Lock()
	conn := g.conn
	g.mu.Unlock()

	if conn == nil || conn.GetState() == connectivity.Shutdown {
		// re-dial con timeout corto para no bloquear
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		g.connect(ctx)
		cancel()
	}
}

// GetConn retorna la conexión gRPC, asegurando que esté disponible.
func (g *GRPCClient) GetConn() *grpc.ClientConn {
	g.ensureConnection()
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.conn
}

// SetTarget permite cambiar el destino y forzar un re-dial.
func (g *GRPCClient) SetTarget(target string) {
	g.mu.Lock()
	g.target = target
	g.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	g.connect(ctx)
	cancel()
}

// Close cierra la conexión actual.
func (g *GRPCClient) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conn != nil {
		_ = g.conn.Close()
		g.conn = nil
	}
}
