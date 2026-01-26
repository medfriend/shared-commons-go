package grpc

import (
	"context"
	"math"
	"math/rand"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

// Resilient gRPC client dialer and interceptors for retries with backoff.

type ClientOptions struct {
	// MaxRetries defines how many times to retry a unary call (not counting the first attempt).
	MaxRetries int
	// InitialBackoff is the initial backoff before the first retry.
	InitialBackoff time.Duration
	// MaxBackoff caps the backoff delay.
	MaxBackoff time.Duration
	// BackoffMultiplier is the exponential factor for backoff growth.
	BackoffMultiplier float64
	// Jitter adds randomization to backoff timings to reduce thundering herd.
	Jitter bool
	// PerRetryTimeout sets a per-attempt timeout. 0 means inherit from ctx.
	PerRetryTimeout time.Duration
	// DialTimeout caps the initial connection dial time. 0 means default (5s).
	DialTimeout time.Duration
	// KeepaliveTime is the period after which, if no activity, a ping is sent.
	KeepaliveTime time.Duration
	// KeepaliveTimeout is the timeout for keepalive pings.
	KeepaliveTimeout time.Duration
	// PermitWithoutStream allows keepalive pings when there are no active RPCs.
	PermitWithoutStream bool
	// EnableCircuitBreaker toggles a simple circuit breaker around calls.
	EnableCircuitBreaker bool
	// CBFailureThreshold is the number of consecutive failures to open the circuit.
	CBFailureThreshold int
	// CBCoolDown is the period the circuit remains open before half-opening.
	CBCoolDown time.Duration
	// AdditionalUnaryInterceptors allows injecting custom unary interceptors to chain before retry.
	AdditionalUnaryInterceptors []grpc.UnaryClientInterceptor
	// AdditionalStreamInterceptors allows injecting custom stream interceptors to chain before retry.
	AdditionalStreamInterceptors []grpc.StreamClientInterceptor
	// LoadBalancingPolicy e.g. "round_robin". Empty to keep gRPC default.
	LoadBalancingPolicy string
	// TransportCredentials allows configuring TLS. If nil, uses insecure.
	TransportCredentials credentials.TransportCredentials
}

func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		MaxRetries:        3,
		InitialBackoff:    150 * time.Millisecond,
		MaxBackoff:        2 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            true,
		PerRetryTimeout:   2 * time.Second,
		DialTimeout:       5 * time.Second,
		// Keepalive settings tuned to avoid server ENHANCE_YOUR_CALM (too_many_pings)
		// Ping no more than every 5 minutes and do not ping without active streams.
		KeepaliveTime:        5 * time.Minute,
		KeepaliveTimeout:     10 * time.Second,
		PermitWithoutStream:  false,
		EnableCircuitBreaker: false,
		CBFailureThreshold:   5,
		CBCoolDown:           30 * time.Second,
		LoadBalancingPolicy:  "round_robin",
	}
}

// DialResilient dials a gRPC target with retry + backoff interceptors and keepalive.
// It uses insecure transport by default; customize TLS at call sites if needed.
func DialResilient(ctx context.Context, target string, logger *zap.Logger, optFns ...func(*ClientOptions)) (*grpc.ClientConn, error) {
	opts := DefaultClientOptions()
	for _, fn := range optFns {
		fn(&opts)
	}

	if logger == nil {
		// create a no-op logger if none provided to avoid nil checks
		l, _ := zap.NewProduction()
		logger = l
	}

	// Client-side keepalive and connect backoff
	dialOpts := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  opts.InitialBackoff,
				Multiplier: opts.BackoffMultiplier,
				MaxDelay:   opts.MaxBackoff,
				Jitter:     0.2, // small jitter for TCP connect attempts
			},
			MinConnectTimeout: opts.DialTimeout,
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                opts.KeepaliveTime,
			Timeout:             opts.KeepaliveTimeout,
			PermitWithoutStream: opts.PermitWithoutStream,
		}),
		grpc.WithBlock(),
	}

	if opts.TransportCredentials != nil {
		dialOpts = append([]grpc.DialOption{grpc.WithTransportCredentials(opts.TransportCredentials)}, dialOpts...)
	} else {
		dialOpts = append([]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, dialOpts...)
	}

	// Set load balancing policy if provided (useful with multiple endpoints via resolver)
	if opts.LoadBalancingPolicy != "" {
		dialOpts = append(dialOpts, grpc.WithDefaultServiceConfig(
			`{"loadBalancingConfig": [{"`+opts.LoadBalancingPolicy+`":{}}]}`,
		))
	}

	// Build interceptors chain: custom -> retry -> (optionally) circuit breaker wrapped inside retry
	retryUnary := newRetryUnaryInterceptor(logger, opts)
	retryStream := newRetryStreamInterceptor(logger, opts)

	unaryInterceptors := append([]grpc.UnaryClientInterceptor{}, opts.AdditionalUnaryInterceptors...)
	streamInterceptors := append([]grpc.StreamClientInterceptor{}, opts.AdditionalStreamInterceptors...)

	unaryInterceptors = append(unaryInterceptors, retryUnary)
	streamInterceptors = append(streamInterceptors, retryStream)

	if len(unaryInterceptors) == 1 {
		dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(unaryInterceptors[0]))
	} else if len(unaryInterceptors) > 1 {
		dialOpts = append(dialOpts, grpc.WithChainUnaryInterceptor(unaryInterceptors...))
	}

	if len(streamInterceptors) == 1 {
		dialOpts = append(dialOpts, grpc.WithStreamInterceptor(streamInterceptors[0]))
	} else if len(streamInterceptors) > 1 {
		dialOpts = append(dialOpts, grpc.WithChainStreamInterceptor(streamInterceptors...))
	}

	dctx := ctx
	var cancel context.CancelFunc
	if opts.DialTimeout > 0 {
		dctx, cancel = context.WithTimeout(ctx, opts.DialTimeout)
		defer cancel()
	}

	conn, err := grpc.DialContext(dctx, target, dialOpts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// retry policy helpers

func isRetriableError(err error) bool {
	if err == nil {
		return false
	}
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted:
		return true
	default:
		return false
	}
}

func nextBackoff(opts ClientOptions, attempt int) time.Duration {
	if attempt <= 0 {
		return opts.InitialBackoff
	}
	// exponential backoff
	d := float64(opts.InitialBackoff) * math.Pow(opts.BackoffMultiplier, float64(attempt))
	if d > float64(opts.MaxBackoff) {
		d = float64(opts.MaxBackoff)
	}
	delay := time.Duration(d)
	if opts.Jitter {
		// 20% jitter
		factor := 0.8 + rand.Float64()*0.4
		delay = time.Duration(float64(delay) * factor)
	}
	return delay
}

// Simple in-memory circuit breaker state
type cbState struct {
	consecutiveFailures int
	openedAt            time.Time
	open                bool
	halfOpen            bool
}

func (s *cbState) onSuccess() {
	s.consecutiveFailures = 0
	s.open = false
	s.halfOpen = false
}

func (s *cbState) onFailure(threshold int, cooldown time.Duration) {
	s.consecutiveFailures++
	if !s.open && s.consecutiveFailures >= threshold {
		s.open = true
		s.openedAt = time.Now()
	}
	if s.open && time.Since(s.openedAt) > cooldown {
		// allow transition to half-open on next attempt
		s.halfOpen = true
	}
}

func (s *cbState) allow(cooldown time.Duration) bool {
	if !s.open {
		return true
	}
	// open: deny until cooldown elapsed, then allow a single half-open attempt
	if time.Since(s.openedAt) >= cooldown {
		if s.halfOpen {
			// already allowed one half-open attempt; keep open until success
			return false
		}
		s.halfOpen = true
		return true
	}
	return false
}

func newRetryUnaryInterceptor(logger *zap.Logger, opts ClientOptions) grpc.UnaryClientInterceptor {
	var cb cbState
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, callOpts ...grpc.CallOption) error {
		// Circuit breaker check
		if opts.EnableCircuitBreaker && !cb.allow(opts.CBCoolDown) {
			return status.Error(codes.Unavailable, "circuit open (cooldown)")
		}

		attempts := opts.MaxRetries + 1 // first attempt + retries
		var lastErr error
		for i := 0; i < attempts; i++ {
			// context per attempt
			callCtx := ctx
			var cancel context.CancelFunc
			if opts.PerRetryTimeout > 0 {
				callCtx, cancel = context.WithTimeout(ctx, opts.PerRetryTimeout)
			}
			if cancel != nil {
				defer cancel()
			}

			err := invoker(callCtx, method, req, reply, cc, callOpts...)
			if err == nil {
				if opts.EnableCircuitBreaker {
					cb.onSuccess()
				}
				return nil
			}

			lastErr = err

			// context cancellation should not be retried
			if callCtx.Err() == context.Canceled || callCtx.Err() == context.DeadlineExceeded {
				break
			}

			if !isRetriableError(err) || i == attempts-1 {
				if opts.EnableCircuitBreaker {
					cb.onFailure(opts.CBFailureThreshold, opts.CBCoolDown)
				}
				break
			}

			// failure accounted for retries as well
			if opts.EnableCircuitBreaker {
				cb.onFailure(opts.CBFailureThreshold, opts.CBCoolDown)
			}

			// backoff before next attempt
			time.Sleep(nextBackoff(opts, i))
		}

		return lastErr
	}
}

func newRetryStreamInterceptor(logger *zap.Logger, opts ClientOptions) grpc.StreamClientInterceptor {
	var cb cbState
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, callOpts ...grpc.CallOption) (grpc.ClientStream, error) {
		if opts.EnableCircuitBreaker && !cb.allow(opts.CBCoolDown) {
			return nil, status.Error(codes.Unavailable, "circuit open (cooldown)")
		}

		attempts := opts.MaxRetries + 1
		var lastErr error
		for i := 0; i < attempts; i++ {
			callCtx := ctx
			var cancel context.CancelFunc
			if opts.PerRetryTimeout > 0 {
				callCtx, cancel = context.WithTimeout(ctx, opts.PerRetryTimeout)
			}
			if cancel != nil {
				defer cancel()
			}

			cs, err := streamer(callCtx, desc, cc, method, callOpts...)
			if err == nil {
				if opts.EnableCircuitBreaker {
					cb.onSuccess()
				}
				return cs, nil
			}

			lastErr = err

			if callCtx.Err() == context.Canceled || callCtx.Err() == context.DeadlineExceeded {
				break
			}

			if !isRetriableError(err) || i == attempts-1 {
				if opts.EnableCircuitBreaker {
					cb.onFailure(opts.CBFailureThreshold, opts.CBCoolDown)
				}
				break
			}

			if opts.EnableCircuitBreaker {
				cb.onFailure(opts.CBFailureThreshold, opts.CBCoolDown)
			}

			time.Sleep(nextBackoff(opts, i))
		}

		return nil, lastErr
	}
}
