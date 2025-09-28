package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/medfriend/shared-commons-go/util/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// splitFullMethod splits a full method name like "pkg.Service/Method".
func splitFullMethod(full string) (string, string, error) {
	for i := 0; i < len(full); i++ {
		if full[i] == '/' {
			return full[:i], full[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid full_method, expected 'package.Service/Method'")
}

// CallJSON calls an arbitrary gRPC method using server reflection, taking a JSON payload
// for the request and returning the response serialized as JSON. The server must enable reflection.
func CallJSON(ctx context.Context, conn *grpc.ClientConn, fullMethod string, jsonPayload []byte) (string, error) {
	log := logger.GetLogger()
	//log.Info("grpc CallJSON start", zap.String("full_method", fullMethod))
	if conn == nil {
		log.Error("nil gRPC connection")
		return "", fmt.Errorf("nil connection")
	}

	serviceName, methodName, err := splitFullMethod(fullMethod)
	if err != nil {
		log.Error("invalid full_method", zap.String("full_method", fullMethod), zap.Error(err))
		return "", err
	}
	//log.Info("resolving service", zap.String("service", serviceName), zap.String("method", methodName))

	// Reflection client
	rcli := grpcreflect.NewClient(ctx, reflectpb.NewServerReflectionClient(conn))
	defer rcli.Reset()

	svcDesc, err := rcli.ResolveService(serviceName)
	if err != nil {
		log.Error("resolve service failed", zap.String("service", serviceName), zap.Error(err))
		return "", fmt.Errorf("resolve service %s: %w", serviceName, err)
	}
	//log.Info("service resolved")

	mDesc := svcDesc.FindMethodByName(methodName)
	if mDesc == nil {
		log.Error("method not found in service", zap.String("service", serviceName), zap.String("method", methodName))
		return "", fmt.Errorf("method %s not found in %s", methodName, serviceName)
	}
	//log.Info("method found", zap.String("method", methodName))

	// Build input message and unmarshal JSON payload
	in := dynamic.NewMessage(mDesc.GetInputType())

	// Normalize jsonPayload: if it's null or empty, use {}
	if len(jsonPayload) == 0 || string(jsonPayload) == "null" {
		jsonPayload = []byte("{}")
	}
	// Validate JSON before unmarshaling into proto
	var tmp interface{}
	if err := json.Unmarshal(jsonPayload, &tmp); err != nil {
		log.Error("invalid JSON payload", zap.ByteString("payload", jsonPayload), zap.Error(err))
		return "", fmt.Errorf("invalid JSON payload: %w", err)
	}
	um := &jsonpb.Unmarshaler{AllowUnknownFields: false}
	if err := um.Unmarshal(bytesNewReader(jsonPayload), in); err != nil {
		log.Error("unmarshal payload failed", zap.ByteString("payload", jsonPayload), zap.Error(err))
		return "", fmt.Errorf("unmarshal payload: %w", err)
	}
	//log.Info("payload unmarshaled")

	// Invoke (grpcdynamic returns the response message)
	stub := grpcdynamic.NewStub(conn)
	resp, err := stub.InvokeRpc(ctx, mDesc, in)
	if err != nil {
		log.Error("rpc invoke failed", zap.String("full_method", fullMethod), zap.Error(err))
		return "", err
	}
	//log.Info("rpc invoke ok", zap.String("full_method", fullMethod))

	// Marshal response to JSON
	marshaler := &jsonpb.Marshaler{OrigName: true, EmitDefaults: false}
	jsonStr, err := marshaler.MarshalToString(resp)
	if err != nil {
		log.Error("marshal response failed", zap.Error(err))
		return "", fmt.Errorf("marshal response: %w", err)
	}
	//log.Info("marshal response ok")
	return jsonStr, nil
}

// tiny helper to adapt []byte to io.Reader without importing bytes in multiple places
func bytesNewReader(b []byte) *bytesReader { return &bytesReader{b: b} }

type bytesReader struct {
	b []byte
	i int
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}
