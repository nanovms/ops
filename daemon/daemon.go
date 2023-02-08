package daemon

import (
	"fmt"

	"context"
	"log"
	"net"
	"net/http"
	"os"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider"
	"github.com/nanovms/ops/types"

	"github.com/nanovms/ops/protos/imageservice"
	"github.com/nanovms/ops/protos/instanceservice"
	"github.com/nanovms/ops/protos/volumeservice"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type server struct{}

func (*server) GetInstances(_ context.Context, in *instanceservice.InstanceListRequest) (*instanceservice.InstancesResponse, error) {
	// stubbed for now - could conceivablly store creds in server and
	// target any provider which would be nice
	c := &types.Config{}
	pc := &types.ProviderConfig{}

	p, err := provider.CloudProvider("onprem", pc)
	if err != nil {
		fmt.Println(err)
	}

	// for now we read from old 'ops run' output stored in
	// ~/.ops/instances/{pid} but can def. re-factor to be in-mem for
	// future since this is coming from daemon
	ctx := api.NewContext(c)
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return nil, err
	}

	pb := &instanceservice.InstancesResponse{
		Count: int32(len(instances)),
	}

	for i := 0; i < len(instances); i++ {
		instance := &instanceservice.Instance{
			Name:      instances[i].Name,
			Image:     instances[i].Image,
			Pid:       instances[i].ID,
			Status:    instances[i].Status,
			PrivateIp: instances[i].PrivateIps[0],
			Created:   instances[i].Created,
		}

		pb.Instances = append(pb.Instances, instance)
	}

	return pb, nil
}

func (*server) GetImages(_ context.Context, in *imageservice.ImageListRequest) (*imageservice.ImagesResponse, error) {

	// stubbed for now - could conceivablly store creds in server and
	// target any provider which would be nice
	c := &types.Config{}
	pc := &types.ProviderConfig{}

	p, err := provider.CloudProvider("onprem", pc)
	if err != nil {
		fmt.Println(err)
	}

	ctx := api.NewContext(c)
	images, err := p.GetImages(ctx)
	if err != nil {
		return nil, err
	}

	pb := &imageservice.ImagesResponse{
		Count: int32(len(images)),
	}

	for i := 0; i < len(images); i++ {
		img := &imageservice.Image{
			Name:    images[i].Name,
			Path:    images[i].Path,
			Size:    images[i].Size,
			Created: images[i].Created.String(),
		}

		pb.Images = append(pb.Images, img)
	}

	return pb, nil
}

func (*server) GetVolumes(_ context.Context, in *volumeservice.VolumeListRequest) (*volumeservice.VolumesResponse, error) {

	// stubbed for now - could conceivablly store creds in server and
	// target any provider which would be nice
	c := &types.Config{}
	pc := &types.ProviderConfig{}

	p, err := provider.CloudProvider("onprem", pc)
	if err != nil {
		fmt.Println(err)
	}

	ctx := api.NewContext(c)
	// no clue why this is passed around like this
	ctx.Config().VolumesDir = api.LocalVolumeDir

	volumes, err := p.GetAllVolumes(ctx)
	if err != nil {
		return nil, err
	}

	rvols := *volumes

	pb := &volumeservice.VolumesResponse{
		Count: int32(len(rvols)),
	}

	for i := 0; i < len(rvols); i++ {
		if err != nil {
			return nil, err
		}

		vol := &volumeservice.Volume{
			Name:    rvols[i].Name,
			Path:    rvols[i].Path,
			Size:    rvols[i].Size, // unfort this has extra meta such as 'mb'
			Created: rvols[i].CreatedAt,
		}

		pb.Volumes = append(pb.Volumes, vol)
	}

	return pb, nil
}

func Daemonize() {
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	imageservice.RegisterImagesServer(s, &server{})
	instanceservice.RegisterInstancesServer(s, &server{})
	volumeservice.RegisterVolumesServer(s, &server{})

	log.Println("Serving gRPC on 0.0.0.0:8080")
	go func() {
		err := s.Serve(lis)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	conn, err := grpc.DialContext(
		context.Background(),
		"0.0.0.0:8080",
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	gwmux := runtime.NewServeMux()
	err = imageservice.RegisterImagesHandler(context.Background(), gwmux, conn)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = instanceservice.RegisterInstancesHandler(context.Background(), gwmux, conn)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = volumeservice.RegisterVolumesHandler(context.Background(), gwmux, conn)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	gwServer := &http.Server{
		Addr:    ":8090",
		Handler: gwmux,
	}

	log.Println("Serving json on http://0.0.0.0:8090")
	fmt.Println("try issuing a request:\tcurl -XGET -k http://localhost:8090/v1/images | jq")
	err = gwServer.ListenAndServe()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
