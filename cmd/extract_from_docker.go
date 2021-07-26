package cmd

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/docker/distribution/reference"
	dockerTypes "github.com/docker/docker/api/types"
	dockerContainer "github.com/docker/docker/api/types/container"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
	"github.com/nanovms/ops/types"
)

// ExtractFromDockerImage creates a package by extracting an executable and its shared libraries
func ExtractFromDockerImage(imageName string, packageName string, targetExecutable string, quiet bool, verbose bool) (string, string) {
	var err error

	if packageName == "" {
		packageName, err = ImageNameToPackageName(imageName)
		if err != nil {
			log.Fatal(err)
		}
	}

	script := fmt.Sprintf(`{
		colors=""

		read_libs() {
			ldd "$1" | rev | cut -d' ' -f2 | rev | while read lib; do
				if [ "$(echo $lib | cut -c1-1)" = "/" ]; then
					exists=0
					resolved_lib=$(readlink -f $lib)

					for i in $colors; do
						if [ "$i" = "'$lib'" ] || [ "$i" = "'$resolved_lib'" ]; then
							exists=1
							break
						fi
					done

					if [ "$exists" = "0" ]; then
						echo "$resolved_lib => $lib"
						colors="$colors '$lib'"

						read_libs "$resolved_lib"
					fi
				fi
			done
		}

		app="$(command -v "%s")"
		echo "$app"
		# skip statically linked binaries
		if ! ldd "$app" 2>&1 | grep -q "Not a valid dynamic program"; then
			read_libs "$app"
		fi
	}`, targetExecutable)

	ctx, cli, containerInfo, err := createContainer(imageName, []string{"sh", "-c", script}, true, quiet)
	if err != nil {
		log.Fatal(err)
	}
	defer cli.ContainerRemove(ctx, containerInfo.ID, dockerTypes.ContainerRemoveOptions{})

	if err := cli.ContainerStart(ctx, containerInfo.ID, dockerTypes.ContainerStartOptions{}); err != nil {
		log.Fatal(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, containerInfo.ID, dockerContainer.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Fatal(err)
		}
	case <-statusCh:
	}

	outReader, err := cli.ContainerLogs(ctx, containerInfo.ID, dockerTypes.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		log.Fatal(err)
	}
	defer outReader.Close()

	bytes, err := ioutil.ReadAll(outReader)
	if err != nil {
		log.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(bytes)), "\n")
	targetExecutablePath, librariesPath := sanitizeLine(lines[0]), lines[1:]

	tempDirectory, err := ioutil.TempDir("", "*")
	if err != nil {
		log.Fatal(err)
	}

	if verbose {
		fmt.Printf("Extracting files inoto %s\n", tempDirectory)
	}

	sysroot := tempDirectory + "/sysroot"

	copyFromContainer(cli, containerInfo.ID, targetExecutablePath, tempDirectory+"/program")
	if err != nil {
		log.Fatal(err)
	}

	for _, libraryLine := range librariesPath {
		sanitizedLibraryLine := sanitizeLine(libraryLine)

		if verbose {
			fmt.Printf("Line: %s\n", sanitizedLibraryLine)
		}

		parts := strings.Split(sanitizedLibraryLine, " => ")

		if len(parts) != 2 {
			log.Fatalf("Invalid library declaration: %s", libraryLine)
		}

		libraryPath, libraryDestination := parts[0], parts[1]

		err = copyFromContainer(cli, containerInfo.ID, libraryPath, sysroot+libraryDestination)
		if err != nil {
			log.Fatal(err)
		}
	}

	parts := strings.Split(packageName, ":")

	c := &types.Config{
		Program: packageName + "/program",
		Args:    []string{"/program"},
		Version: parts[len(parts)-1],
	}

	json, _ := json.MarshalIndent(c, "", "  ")

	err = ioutil.WriteFile(path.Join(tempDirectory, "package.manifest"), json, 0666)
	if err != nil {
		log.Panic(err)
	}

	packageDirectory := MovePackageFiles(tempDirectory, path.Join(localPackageDirectoryPath(), packageName))

	return packageName, packageDirectory
}

func sanitizeLine(line string) string {
	if strings.HasPrefix(line, string([]byte{1, 0, 0, 0, 0, 0, 0})) {
		return line[8:]
	}

	return line
}

func createContainer(image string, command []string, pull bool, quiet bool) (context.Context, *dockerClient.Client, dockerContainer.ContainerCreateCreatedBody, error) {
	ctx := context.Background()
	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, nil, dockerContainer.ContainerCreateCreatedBody{}, err
	}

	if pull {
		reader, err := cli.ImagePull(ctx, image, dockerTypes.ImagePullOptions{})
		if err != nil {
			return nil, nil, dockerContainer.ContainerCreateCreatedBody{}, err
		}
		defer reader.Close()

		if !quiet {
			termFd, isTerm := term.GetFdInfo(os.Stderr)
			jsonmessage.DisplayJSONMessagesStream(reader, os.Stdout, termFd, isTerm, nil)
		}
	}

	containerInfo, err := cli.ContainerCreate(ctx, &dockerContainer.Config{
		Image: image,
		Cmd:   command,
	}, nil, nil, nil, "")
	if err != nil {
		return nil, nil, dockerContainer.ContainerCreateCreatedBody{}, err
	}

	return ctx, cli, containerInfo, nil
}

func copyFromContainer(cli *dockerClient.Client, containerID string, containerPath string, hostPath string) error {
	err := os.MkdirAll(path.Dir(hostPath), 0764)
	if err != nil {
		log.Fatal(err)
	}

	destination, err := os.Create(hostPath)
	if err != nil {
		return err
	}
	defer destination.Close()

	fileReader, _, err := cli.CopyFromContainer(context.Background(), containerID, containerPath)
	if err != nil {
		return err
	}
	defer fileReader.Close()

	tr := tar.NewReader(fileReader)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal(err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(hostPath); err != nil {
				if err := os.MkdirAll(hostPath, 0755); err != nil {
					return err
				}
			}

		case tar.TypeReg:
			f, err := os.Create(hostPath)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
		}
	}

	return nil
}

// ImageNameToPackageName converts a Docker image name to a package name
func ImageNameToPackageName(imageName string) (string, error) {
	matches := reference.ReferenceRegexp.FindStringSubmatch(imageName)

	if matches == nil {
		if imageName == "" {
			return "", reference.ErrNameEmpty
		}

		if reference.ReferenceRegexp.FindStringSubmatch(strings.ToLower(imageName)) != nil {
			return "", reference.ErrNameContainsUppercase
		}

		return "", reference.ErrReferenceInvalidFormat
	}

	if len(matches[1]) > reference.NameTotalLengthMax {
		return "", reference.ErrNameTooLong
	}

	nameMatches := regexp.MustCompile(`(.*\/)?(.*)$`).FindStringSubmatch(matches[1])

	name := nameMatches[2]
	tag := matches[2]

	return name + "-" + tag, nil
}
