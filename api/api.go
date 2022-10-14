package api

import (
	"context"

	"github.com/NpoolPlatform/message/npool/servicetmpl"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Server struct {
	servicetmpl.UnimplementedManagerServer
}

func Register(server grpc.ServiceRegistrar) {
	servicetmpl.RegisterManagerServer(server, &Server{})
}

func RegisterGateway(mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	if err := servicetmpl.RegisterManagerHandlerFromEndpoint(context.Background(), mux, endpoint, opts); err != nil {
		return err
	}
	return nil
}
