package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/spf13/cobra"
)

func imageCommandHandler(cmd *cobra.Command, args []string) {
	if _, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); !ok {
		fmt.Printf(api.ErrorColor, "error: GOOGLE_APPLICATION_CREDENTIALS not set.\n")
		fmt.Printf(api.ErrorColor, "Follow https://cloud.google.com/storage/docs/reference/libraries to set it up.\n")
		os.Exit(1)
	}

	provider, _ := cmd.Flags().GetString("target-cloud")
	config, _ := cmd.Flags().GetString("config")
	config = strings.TrimSpace(config)
	pkg, _ := cmd.Flags().GetString("package")
	pkg = strings.TrimSpace(pkg)
	cmdargs, _ := cmd.Flags().GetStringArray("args")

	c := unWarpConfig(config)
	// override config from command line
	if len(provider) > 0 {
		c.CloudConfig.Platform = provider
	}

	if len(c.CloudConfig.Platform) == 0 {
		fmt.Printf(api.ErrorColor, "error: Please select on of the cloud platform in config. [onprem, gcp]")
		os.Exit(1)
	}

	if len(c.CloudConfig.ProjectID) == 0 {
		fmt.Printf(api.ErrorColor, "error: Please specifiy a cloud projectid in config.\n")
		os.Exit(1)
	}

	if len(c.CloudConfig.BucketName) == 0 {
		fmt.Printf(api.ErrorColor, "error: Please specifiy a cloud bucket in config.\n")
		os.Exit(1)
	}

	if len(pkg) > 0 {
		c.Args = append(c.Args, cmdargs...)
	} else {
		if len(cmdargs) != 0 {
			c.Program = cmdargs[0]
		} else if len(c.Args) != 0 {
			c.Program = c.Args[0]
		} else {
			fmt.Println(api.ErrorColor, "error: Please mention program to run.")
			os.Exit(1)
		}
	}

	prepareImages(c)
	p := getCloudProvider(provider)
	ctx := api.NewContext(c, &p)

	var archpath string
	var err error
	if len(pkg) > 0 {
		expackage := downloadAndExtractPackage(pkg)
		// load the package manifest
		manifest := path.Join(expackage, "package.manifest")
		if _, err := os.Stat(manifest); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		pkgConfig := unWarpConfig(manifest)
		c = mergeConfigs(pkgConfig, c)
		setDefaultImageName(cmd, c)
		// Config merged with package config, need to update context
		ctx = api.NewContext(c, &p)
		archpath, err = p.BuildImageWithPackage(ctx, expackage)
	} else {
		setDefaultImageName(cmd, c)
		archpath, err = p.BuildImage(ctx)
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if c.CloudConfig.Platform == "gcp" {
		gcloud := p.(*api.GCloud)
		err = gcloud.Storage.CopyToBucket(c, archpath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = gcloud.CreateImage(ctx)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("gcp image '%s' created...\n", c.CloudConfig.ImageName)
		}
	}
}

func ImageCommands() *cobra.Command {
	var (
		targetCloud, config, pkg, imageName string
		args                                []string
	)
	var cmdImageCreate = &cobra.Command{
		Use:   "create",
		Short: "create nanos image from ELF",
		Run:   imageCommandHandler,
	}

	cmdImageCreate.PersistentFlags().StringVarP(&targetCloud, "target-cloud", "t", "gcp", "cloud platform [gcp, onprem]")
	cmdImageCreate.PersistentFlags().StringVarP(&config, "config", "c", "", "ops config file")
	cmdImageCreate.PersistentFlags().StringVarP(&pkg, "package", "p", "", "ops package name")
	cmdImageCreate.PersistentFlags().StringArrayVarP(&args, "args", "a", nil, "command line arguments")
	cmdImageCreate.PersistentFlags().StringVarP(&imageName, "imagename", "i", "", "image name")

	var cmdImage = &cobra.Command{
		Use:       "image",
		Short:     "manage nanos images",
		ValidArgs: []string{"create"},
		Args:      cobra.OnlyValidArgs,
	}
	cmdImage.AddCommand(cmdImageCreate)
	return cmdImage
}
