package cmd

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

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/nanovms/ops/protos/imageservice"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DaemonizeCommand turns ops into a daemon
func DaemonizeCommand() *cobra.Command {
	var cmdDaemonize = &cobra.Command{
		Use:   "daemonize",
		Short: "Daemonize OPS",
		Run:   daemonizeCommandHandler,
	}

	return cmdDaemonize
}

// prob belongs in a root grpc-server folder
// not in the cmds folder
type server struct{}

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

func daemonizeCommandHandler(cmd *cobra.Command, args []string) {
	fmt.Println("Note: If on a mac this expects ops to have suid bit set for networking.")
	fmt.Println("if you used the installer you are set otherwise run the following command\n" +
		"\tsudo chown -R root /usr/local/bin/qemu-system-x86_64\n" +
		"\tsudo chmod u+s /usr/local/bin/qemu-system-x86_64")

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	imageservice.RegisterImagesServer(s, &server{})
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
