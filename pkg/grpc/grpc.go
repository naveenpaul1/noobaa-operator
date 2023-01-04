package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	//pb "externalscaler-sample/externalscaler"
)

type ExternalScaler struct{}

type USGSResponse struct {
	Features []USGSFeature `json:"features"`
}

type USGSFeature struct {
	Properties USGSProperties `json:"properties"`
}

type USGSProperties struct {
	Mag float64 `json:"mag"`
}

func (e *ExternalScaler) IsActive(ctx context.Context, scaledObject *ScaledObjectRef) (*IsActiveResponse, error) {

	return &IsActiveResponse{
		Result: true,
	}, nil
}

func (e *ExternalScaler) GetMetricSpec(context.Context, *ScaledObjectRef) (*GetMetricSpecResponse, error) {
	return &GetMetricSpecResponse{
		MetricSpecs: []*MetricSpec{{
			MetricName: "noobaaThreshold",
			TargetSize: 10,
		}},
	}, nil
}

func (e *ExternalScaler) GetMetrics(_ context.Context, metricRequest *GetMetricsRequest) (*GetMetricsResponse, error) {

	//value, _ := 30 //os.LookupEnv("NOOBAA_THRESH")
	//fmt.Println("NOOBAA_THRESH ***********", value)
	intValue := 30 //strconv.Atoi(value)
	return &GetMetricsResponse{
		MetricValues: []*MetricValue{{
			MetricName:  "noobaaThreshold",
			MetricValue: int64(intValue),
		}},
	}, nil
}

func (e *ExternalScaler) StreamIsActive(scaledObject *ScaledObjectRef, epsServer ExternalScaler_StreamIsActiveServer) error {

	for {
		select {
		case <-epsServer.Context().Done():
			// call cancelled
			fmt.Println("INSIDE DONE")
			return nil
		case <-time.Tick(time.Hour * 1):
			if true {
				epsServer.Send(&IsActiveResponse{
					Result: true,
				})
			}
		}
	}
}

func Startgrpc() {
	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":9094")
	RegisterExternalScalerServer(grpcServer, &ExternalScaler{})

	fmt.Println("listenting on :9094")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
