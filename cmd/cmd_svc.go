package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nanovms/ops/types"
	"github.com/spf13/cobra"
)

// SvcCommand provides support for running multiple services together.
func SvcCommand() *cobra.Command {
	var cmdSvc = &cobra.Command{
		Use:   "svc",
		Short: "Run multiple services",
		Args:  cobra.MinimumNArgs(0),
		Run:   svcCommandHandler,
	}

	/*
		persistentFlags := cmdSvc.PersistentFlags()

		PersistConfigCommandFlags(persistentFlags)
		PersistBuildImageCommandFlags(persistentFlags)
		PersistRunLocalInstanceCommandFlags(persistentFlags)
		PersistNightlyCommandFlags(persistentFlags)
		PersistNanosVersionCommandFlags(persistentFlags)
	*/

	cmdSvc.PersistentFlags().StringP("orchestrate", "o", "", "orchestration file")

	return cmdSvc
}

// Service is an image with its corresponding config.
type Service struct {
	Image  string       `json:"image"`
	Config types.Config `json:"config"`
}

// ServiceConfig is composed of a set of services intended to be ran
// together.
type ServiceConfig struct {
	Services []Service `json:"services"`
}

func svcCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()
	file, err := flags.GetString("orchestrate")
	if err != nil {
		fmt.Println(err)
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("error reading config: %v\n", err)
	}

	sc := &ServiceConfig{}
	dec := json.NewDecoder(bytes.NewReader(data))
	err = dec.Decode(&sc)
	if err != nil {
		fmt.Println(err)
	}

	for i := 0; i < len(sc.Services); i++ {

		svc := sc.Services[i]
		c := svc.Config

		c.CloudConfig.ImageName = svc.Image

		instanceName, _ := flags.GetString("instance-name")

		instanceGroup, _ := flags.GetString("instance-group")

		if instanceName == "" {
			c.RunConfig.InstanceName = fmt.Sprintf("%v-%v",
				strings.Split(filepath.Base(c.CloudConfig.ImageName), ".")[0],
				strconv.FormatInt(time.Now().Unix(), 10),
			)
		} else {
			c.RunConfig.InstanceName = instanceName
		}

		if instanceGroup != "" {
			c.RunConfig.InstanceGroup = instanceGroup
		}

		p, ctx, err := getProviderAndContext(&c, c.CloudConfig.Platform)
		if err != nil {
			exitForCmd(cmd, err.Error())
		}

		err = p.CreateInstance(ctx)
		if err != nil {
			exitWithError(err.Error())
		}

	}
}
