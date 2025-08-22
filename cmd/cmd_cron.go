package cmd

import (
	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/provider/aws"
	"github.com/spf13/cobra"
)

// CronCommands provides cron related commands
func CronCommands() *cobra.Command {
	var cmdCron = &cobra.Command{
		Use:       "cron",
		Short:     "manage cronjobs",
		ValidArgs: []string{"create", "list", "delete", "enable", "disable"},
		Args:      cobra.OnlyValidArgs,
	}

	PersistConfigCommandFlags(cmdCron.PersistentFlags())
	PersistProviderCommandFlags(cmdCron.PersistentFlags())

	cmdCron.AddCommand(cronCreateCommand())
	cmdCron.AddCommand(cronListCommand())
	cmdCron.AddCommand(cronDeleteCommand())
	cmdCron.AddCommand(cronEnableCommand())
	cmdCron.AddCommand(cronDisableCommand())

	return cmdCron
}

func cronCreateCommand() *cobra.Command {
	// should take cron as well

	var cmdCronCreate = &cobra.Command{
		Use:   "create <cron_name>",
		Short: "create cron from image",
		Run:   cronCreateCommandHandler,
	}

	persistentFlags := cmdCronCreate.PersistentFlags()

	PersistBuildImageCommandFlags(persistentFlags)
	PersistNightlyCommandFlags(persistentFlags)
	PersistNanosVersionCommandFlags(persistentFlags)
	PersistPkgCommandFlags(persistentFlags)

	return cmdCronCreate
}

func cronCreateCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()
	c := api.NewConfig()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	nightlyFlags := NewNightlyCommandFlags(flags)
	nanosVersionFlags := NewNanosVersionCommandFlags(flags)
	buildImageFlags := NewBuildImageCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)
	pkgFlags := NewPkgCommandFlags(flags)

	c.CloudConfig.ImageName = args[0]
	// schedule = args[1]
	schedule := "rate(1 minutes)"

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, nightlyFlags, nanosVersionFlags, buildImageFlags, providerFlags, pkgFlags)

	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	p := aws.NewProvider()
	ctx := api.NewContext(c)
	err = p.Initialize(&c.CloudConfig)

	err = p.CreateCron(ctx, schedule)
	if err != nil {
		exitWithError(err.Error())
	}

}

func cronListCommand() *cobra.Command {
	var cmdCronList = &cobra.Command{
		Use:   "list",
		Short: "list crons from provider",
		Run:   cronListCommandHandler,
	}
	return cmdCronList
}

func cronListCommandHandler(cmd *cobra.Command, args []string) {
	flags := cmd.Flags()

	configFlags := NewConfigCommandFlags(flags)
	globalFlags := NewGlobalCommandFlags(flags)
	providerFlags := NewProviderCommandFlags(flags)

	c := api.NewConfig()

	mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags)
	err := mergeContainer.Merge(c)
	if err != nil {
		exitWithError(err.Error())
	}

	/*
	   p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)

	   	if err != nil {
	   		exitWithError(err.Error())
	   	}
	*/
	p := aws.NewProvider()
	ctx := api.NewContext(c)

	err = p.ListCrons(ctx)
	if err != nil {
		exitWithError(err.Error())
	}
}

func cronDeleteCommand() *cobra.Command {
	var cmdCronDelete = &cobra.Command{
		Use:   "delete <image_name>",
		Short: "delete images from provider",
		Run:   imageDeleteCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}

	cmdCronDelete.PersistentFlags().StringP("lru", "", "", "clean least recently used images with a time notation. Use \"1w\" notation to delete images older than one week. Other notation examples are 300d, 3w, 1m and 2y.")
	cmdCronDelete.PersistentFlags().BoolP("assume-yes", "", false, "clean images without waiting for confirmation")
	cmdCronDelete.PersistentFlags().BoolP("force", "", false, "force even if image is being used by instance")

	return cmdCronDelete
}

func cronDeleteCommandHandler(cmd *cobra.Command, args []string) {
	/*
		flags := cmd.Flags()

		configFlags := NewConfigCommandFlags(flags)
		globalFlags := NewGlobalCommandFlags(flags)
		providerFlags := NewProviderCommandFlags(flags)

		c := api.NewConfig()

		mergeContainer := NewMergeConfigContainer(configFlags, globalFlags, providerFlags)
		err := mergeContainer.Merge(c)
		if err != nil {
			exitWithError(err.Error())
		}

		zone, _ := cmd.Flags().GetString("zone")
		if zone != "" {
			c.CloudConfig.Zone = zone
		}

		lru, _ := cmd.Flags().GetString("lru")
		assumeYes, _ := cmd.Flags().GetBool("assume-yes")
		forceFlag, _ := cmd.Flags().GetBool("force")

		p, ctx, err := getProviderAndContext(c, c.CloudConfig.Platform)
		if err != nil {
			exitWithError(err.Error())
		}

		// Check if image being used
		images, err := p.GetImages(ctx, "")
		if err != nil {
			exitWithError(err.Error())
		}

		imageMap := make(map[string]string)
		for _, name := range args {
			for _, img := range images {
				if img.Name == name {
					imageMap[name] = img.Path
					break
				}
			}
		}

		if !forceFlag {

			instances, err := p.GetInstances(ctx)
			if err != nil {
				exitWithError(err.Error())
			}

			if len(instances) > 0 {
				for imgName, imgPath := range imageMap {
					for _, is := range instances {
						var matchedImage bool
						if c.CloudConfig.Platform == onprem.ProviderName {
							matchedImage = (is.Image == imgPath)
						} else {
							matchedImage = (is.Image == imgName)
						}

						if matchedImage {
							fmt.Printf("image '%s' is being used\n", imgName)
							os.Exit(1)
						}
					}
				}
			}

		}

		imagesToDelete := []string{}

		if lru != "" {
			olderThanDate, err := SubtractTimeNotation(time.Now(), lru)
			if err != nil {
				exitWithError(fmt.Errorf("failed getting date from lru flag: %s", err).Error())
			}

			for _, image := range images {
				if image.Created.Before(olderThanDate) {
					if image.ID != "" {
						imagesToDelete = append(imagesToDelete, image.ID)
					} else {
						imagesToDelete = append(imagesToDelete, image.Name)
					}
				}
			}
		}

		if len(args) > 0 {
			imagesToDelete = append(imagesToDelete, args...)
		}

		if len(imagesToDelete) == 0 {
			log.Info("There are no images to delete")
			return
		}

		if assumeYes != true {
			fmt.Printf("You are about to delete the next images:\n")
			for _, i := range imagesToDelete {
				fmt.Println(i)
			}
			fmt.Println("Are you sure? (yes/no)")
			confirmation := askForConfirmation()
			if !confirmation {
				return
			}
		}

		responses := make(chan error)

		deleteImage := func(imageName string) {
			errMsg := p.DeleteImage(ctx, imageName)
			if errMsg != nil {
				errMsg = fmt.Errorf("failed deleting %s: %v", imageName, errMsg)
			}

			responses <- errMsg
		}

		for _, i := range imagesToDelete {
			go deleteImage(i)
		}

		for range imagesToDelete {
			err = <-responses
			if err != nil {
				log.Error(err)
			}

		}
	*/

}

func cronEnableCommand() *cobra.Command {
	var cmdCronEnable = &cobra.Command{
		Use:   "enable <image_name>",
		Short: "enable image from provider",
		Run:   cronEnableCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}

	return cmdCronEnable
}

func cronEnableCommandHandler(cmd *cobra.Command, args []string) {
}

func cronDisableCommand() *cobra.Command {
	var cmdCronDisable = &cobra.Command{
		Use:   "disable <image_name>",
		Short: "disable image from provider",
		Run:   cronDisableCommandHandler,
		Args:  cobra.MinimumNArgs(1),
	}

	return cmdCronDisable
}

func cronDisableCommandHandler(cmd *cobra.Command, args []string) {
}
