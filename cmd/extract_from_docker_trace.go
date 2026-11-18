package cmd

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/tls"
	"debug/elf"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	dockerContainer "github.com/docker/docker/api/types/container"
	dockerImage "github.com/docker/docker/api/types/image"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/nanovms/ops/types"
)

// The rule: the package carries the files the program reaches.
//
// A program reaches a file in two ways. It asks for it, with a call that names it: open, stat,
// readlink, execve. An image run under strace gives that list, from the first instruction on: the
// libraries the program is linked against and the ones it loads by name, the configuration it
// reads, the data it opens. And the kernel asks for a file on its behalf when it executes a
// program: the interpreter that program names, which is the dynamic loader of an ELF file or the
// program after #! in a script. No call of the program names that file, so it is added for every
// file the program executed, and for the interpreter of that interpreter, and so on.
//
// Nothing else of the image is carried.

// traceMount is where the directory holding strace, and the list it writes, is seen from inside
// the container.
const traceMount = "/.ops-trace"

// stracePackager is the image strace is taken from. Alpine's is built against musl alone, which
// makes it a program that can be carried into an image of any other distribution along with the
// musl loader.
const stracePackager = "alpine:3.20"

// tracer is a directory on this side that holds strace and the loader to run it with, and gets the
// list of files the traced program opened.
type tracer struct {
	dir    string
	loader string
}

// prepareTracer fetches strace and what it is linked against into a directory of its own.
func prepareTracer(ctx context.Context, cli *dockerClient.Client) (*tracer, error) {
	dir, err := os.MkdirTemp("", "ops-trace-*")
	if err != nil {
		return nil, err
	}
	// whoever the traced program runs as writes the list and reads strace
	if err := os.Chmod(dir, 0777); err != nil {
		return nil, err
	}

	pulled, err := cli.ImagePull(ctx, stracePackager, dockerImage.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("fetching %s for strace: %v", stracePackager, err)
	}
	io.Copy(io.Discard, pulled)
	pulled.Close()

	script := `set -e
apk add --no-cache strace >/dev/null
mkdir -p /out
cp -L /usr/bin/strace /out/strace
cp -L /lib/ld-musl-*.so.1 /out/
for lib in $(ldd /usr/bin/strace | awk '/=>/ {print $3}'); do cp -Ln "$lib" /out/ 2>/dev/null || true; done`

	created, err := cli.ContainerCreate(ctx, &dockerContainer.Config{
		Image: stracePackager,
		Cmd:   []string{"sh", "-c", script},
	}, nil, nil, nil, "")
	if err != nil {
		return nil, err
	}
	defer cli.ContainerRemove(ctx, created.ID, dockerContainer.RemoveOptions{Force: true})

	if err := cli.ContainerStart(ctx, created.ID, dockerContainer.StartOptions{}); err != nil {
		return nil, err
	}
	waited, errs := cli.ContainerWait(ctx, created.ID, dockerContainer.WaitConditionNotRunning)
	select {
	case result := <-waited:
		if result.StatusCode != 0 {
			return nil, fmt.Errorf("getting strace out of %s stopped with %d", stracePackager, result.StatusCode)
		}
	case err := <-errs:
		return nil, err
	}

	// strace and what it is linked against, in the directory out under dir
	if err := copyDirFromContainer(cli, created.ID, "/out", dir); err != nil {
		return nil, err
	}

	loaders, _ := filepath.Glob(filepath.Join(dir, "out", "ld-musl-*.so.1"))
	if len(loaders) == 0 {
		return nil, fmt.Errorf("no musl loader came with strace")
	}
	if _, err := os.Stat(filepath.Join(dir, "out", "strace")); err != nil {
		return nil, fmt.Errorf("strace did not come out of %s", stracePackager)
	}

	// created ahead so that a program that is not root can write to it
	if err := os.WriteFile(filepath.Join(dir, "files"), nil, 0666); err != nil {
		return nil, err
	}
	os.Chmod(filepath.Join(dir, "files"), 0666)

	return &tracer{dir: dir, loader: filepath.Base(loaders[0])}, nil
}

// wrap puts strace in front of what the image starts. It follows every process and thread the
// program makes and writes down the calls that name a file.
func (t *tracer) wrap(argv []string) []string {
	wrapped := []string{
		traceMount + "/out/" + t.loader, "--library-path", traceMount + "/out", traceMount + "/out/strace",
		"-f", "-q", "-s", "4096", "-e", "trace=file", "-e", "signal=none",
		"-o", traceMount + "/files", "--",
	}
	return append(wrapped, argv...)
}

func (t *tracer) bind() string {
	return t.dir + ":" + traceMount
}

// trace is what strace wrote down.
func (t *tracer) trace() []byte {
	raw, _ := os.ReadFile(filepath.Join(t.dir, "files"))
	return raw
}

var traceCall = regexp.MustCompile(`^(?:\[pid +\d+\] +|\d+ +)([a-z_0-9]+)\(`)

// firstQuoted reads the path a call names: its first argument, or the one after a descriptor.
func firstQuoted(rest string) (name string, after string, relativeToCwd bool, ok bool) {
	relativeToCwd = true
	if strings.HasPrefix(rest, "AT_FDCWD, ") {
		rest = rest[len("AT_FDCWD, "):]
	} else if !strings.HasPrefix(rest, "\"") {
		// the first argument is a descriptor: a path that is not absolute has no telling
		relativeToCwd = false
		if comma := strings.Index(rest, ", "); comma >= 0 {
			rest = rest[comma+2:]
		}
	}
	if !strings.HasPrefix(rest, "\"") {
		return "", "", false, false
	}

	end := 1
	for end < len(rest) && rest[end] != '"' {
		if rest[end] == '\\' {
			end++
		}
		end++
	}
	if end >= len(rest) {
		return "", "", false, false
	}
	name = rest[1:end]
	if unquoted, err := strconv.Unquote("\"" + name + "\""); err == nil {
		name = unquoted
	}

	return name, rest[end+1:], relativeToCwd, true
}

// resultOf is what a call came back with, and whether the trace has it.
func resultOf(after string) (string, bool) {
	at := strings.LastIndex(after, ") = ")
	if at < 0 {
		return "", false
	}
	result := after[at+len(") = "):]
	if space := strings.IndexAny(result, " \t"); space >= 0 {
		result = result[:space]
	}
	return result, true
}

// touchedPaths reads what strace wrote and returns the files named by the calls that found them.
// A call that came back with an error found nothing; one the trace cut in two - unfinished, with
// its end written on another line - is taken, since a file looked for is a file that may be needed.
// A path that is not absolute is read against workingDir when it was relative to the working
// directory, and left out when it was relative to a file descriptor that is not known here.
// What happened before line from is not read: a launcher script runs tools before it hands over to
// the program, and a unikernel runs no script, so what they opened is not what the program needs.
func touchedPaths(trace []byte, workingDir string, from int) (paths []string, executed []string) {
	seen := map[string]bool{}
	ran := map[string]bool{}

	for i, line := range strings.Split(string(trace), "\n") {
		if i < from {
			continue
		}
		call := traceCall.FindStringSubmatch(line)
		if call == nil {
			continue
		}

		name, after, relativeToCwd, ok := firstQuoted(line[len(call[0]):])
		if !ok {
			continue
		}
		if !strings.Contains(after, "<unfinished ...>") {
			result, found := resultOf(after)
			if !found || strings.HasPrefix(result, "-") {
				continue
			}
		}

		if !strings.HasPrefix(name, "/") {
			if !relativeToCwd || workingDir == "" || name == "" {
				continue
			}
			name = path.Join(workingDir, name)
		}
		seen[path.Clean(name)] = true
		if call[1] == "execve" {
			ran[path.Clean(name)] = true
		}
	}

	for p := range seen {
		paths = append(paths, p)
	}
	for p := range ran {
		executed = append(executed, p)
	}
	sort.Strings(paths)
	sort.Strings(executed)

	return paths, executed
}

// notCarried says whether a path is one the kernel of the machine makes or the container runtime
// puts in, which no package has to bring.
func notCarried(p string) bool {
	for _, prefix := range []string{"/proc", "/sys", "/dev", traceMount} {
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			return true
		}
	}
	return false
}

// copyHostFile puts a copy of a file that is in the sysroot under another name.
func copyHostFile(sysroot string, from string, to string) {
	source := filepath.Join(sysroot, from)
	destination := filepath.Join(sysroot, to)
	if _, err := os.Lstat(destination); err == nil {
		return
	}
	info, err := os.Stat(source)
	if err != nil || os.MkdirAll(path.Dir(destination), 0755) != nil {
		return
	}
	in, err := os.Open(source)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY, info.Mode()|0600)
	if err != nil {
		return
	}
	io.Copy(out, in)
	out.Close()
}

// copyTouched copies the files that were opened out of the container into sysroot. A package does
// not carry a symbolic link: an absolute one names a path of the machine that builds the image, not
// of the image. What a link leads to is carried instead, under the name of the link and under its
// own.
func copyTouched(ctx context.Context, cli *dockerClient.Client, containerID string, sysroot string, paths []string, verbose bool) {
	done := map[string]bool{}
	// the names a path was reached by, through links
	aliases := map[string][]string{}
	queue := append([]string{}, paths...)

	for len(queue) > 0 {
		p := path.Clean(queue[0])
		queue = queue[1:]
		if done[p] || notCarried(p) {
			continue
		}
		done[p] = true

		reader, stat, err := cli.CopyFromContainer(ctx, containerID, p)
		if err != nil {
			// a file that was looked for and was there at the time, but is not any more
			if verbose {
				fmt.Printf("not carried %s: %v\n", p, err)
			}
			continue
		}

		target := filepath.Join(sysroot, p)
		tr := tar.NewReader(reader)
		header, err := tr.Next()
		if err != nil {
			reader.Close()
			continue
		}

		switch {
		case stat.Mode&os.ModeSymlink != 0 || header.Typeflag == tar.TypeSymlink:
			link := header.Linkname
			if link == "" {
				link = stat.LinkTarget
			}
			if !strings.HasPrefix(link, "/") {
				link = path.Join(path.Dir(p), link)
			}
			link = path.Clean(link)
			names := append([]string{p}, aliases[p]...)
			aliases[link] = append(aliases[link], names...)
			if notCarried(link) {
				// a log that is a link to the console of the container: an empty file takes its place,
				// so that opening it works
				for _, name := range names {
					if os.MkdirAll(filepath.Join(sysroot, path.Dir(name)), 0755) == nil {
						os.WriteFile(filepath.Join(sysroot, name), nil, 0644)
					}
				}
			} else if done[link] {
				// it was carried already, under its own name
				for _, name := range names {
					copyHostFile(sysroot, link, name)
				}
			} else {
				queue = append(queue, link)
			}
		case header.Typeflag == tar.TypeDir:
			os.MkdirAll(target, 0755)
		case header.Typeflag == tar.TypeReg:
			if os.MkdirAll(path.Dir(target), 0755) != nil {
				break
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode)|0600)
			if err != nil {
				break
			}
			io.Copy(out, tr)
			out.Close()
			for _, name := range aliases[p] {
				copyHostFile(sysroot, p, name)
			}
		}
		reader.Close()
	}
}

// inSysroot follows the symbolic links of a path that was carried, inside the sysroot rather than
// on this machine, and gives the path of the file they lead to.
func inSysroot(sysroot string, p string) string {
	for i := 0; i < 16; i++ {
		info, err := os.Lstat(filepath.Join(sysroot, p))
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			return p
		}
		link, err := os.Readlink(filepath.Join(sysroot, p))
		if err != nil {
			return p
		}
		if !strings.HasPrefix(link, "/") {
			link = path.Join(path.Dir(p), link)
		}
		p = path.Clean(link)
	}
	return p
}

// interpreterOf is the file the kernel loads to run a program: the dynamic loader named by an ELF
// file, or the program a script names after #!. A file that names none, a static binary or a
// library, gives "".
func interpreterOf(file string) string {
	if f, err := elf.Open(file); err == nil {
		defer f.Close()
		if section := f.Section(".interp"); section != nil {
			if raw, err := section.Data(); err == nil {
				return strings.TrimRight(string(raw), "\x00")
			}
		}
		return ""
	}

	in, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer in.Close()
	head := make([]byte, 256)
	n, _ := in.Read(head)
	if n < 2 || head[0] != '#' || head[1] != '!' {
		return ""
	}
	line := strings.SplitN(string(head[2:n]), "\n", 2)[0]
	if fields := strings.Fields(line); len(fields) > 0 {
		return fields[0]
	}
	return ""
}

// carryInterpreters copies, for each file that was executed, the interpreter the kernel loads for
// it, then the interpreter of that one if it has one. It is the second half of the rule.
func carryInterpreters(ctx context.Context, cli *dockerClient.Client, containerID string, sysroot string, executed []string, verbose bool) {
	done := map[string]bool{}
	queue := append([]string{}, executed...)

	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]

		interp := interpreterOf(filepath.Join(sysroot, inSysroot(sysroot, p)))
		if interp == "" || done[interp] {
			continue
		}
		done[interp] = true

		if verbose {
			fmt.Printf("%s is run by %s\n", p, interp)
		}
		copyTouched(ctx, cli, containerID, sysroot, []string{interp}, verbose)
		queue = append(queue, interp)
	}
}

// imageProgram is what an image starts when it is not told otherwise, and the directory and
// environment it starts it in.
func imageProgram(ctx context.Context, cli *dockerClient.Client, image string) (argv []string, workingDir string, env []string, exposed []int, system string, err error) {
	inspected, err := cli.ImageInspect(ctx, image)
	if err != nil {
		return nil, "", nil, nil, "", err
	}
	config := inspected.Config
	if config == nil {
		return nil, "", nil, nil, "", fmt.Errorf("%s has no configuration", image)
	}
	for port := range config.ExposedPorts {
		number, protocol, _ := strings.Cut(string(port), "/")
		if n, err := strconv.Atoi(number); err == nil && (protocol == "" || protocol == "tcp") {
			exposed = append(exposed, n)
		}
	}
	sort.Ints(exposed)
	return append(append([]string{}, config.Entrypoint...), config.Cmd...), config.WorkingDir, config.Env, exposed, inspected.Os, nil
}

// whereIs finds a program the way a shell would, among the files already carried.
func whereIs(sysroot string, program string, env []string) string {
	if strings.HasPrefix(program, "/") {
		return program
	}
	searched := "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			searched = e[len("PATH="):]
		}
	}
	for _, dir := range strings.Split(searched, ":") {
		candidate := path.Join(dir, program)
		if _, err := os.Stat(filepath.Join(sysroot, candidate)); err == nil {
			return candidate
		}
	}
	return program
}

// execIn runs a command in the container, as root, and brings back what it prints.
func execIn(ctx context.Context, cli *dockerClient.Client, containerID string, cmd []string) ([]byte, error) {
	created, err := cli.ContainerExecCreate(ctx, containerID, dockerContainer.ExecOptions{
		User:         "root",
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return nil, err
	}
	attached, err := cli.ContainerExecAttach(ctx, created.ID, dockerContainer.ExecAttachOptions{})
	if err != nil {
		return nil, err
	}
	defer attached.Close()

	out := &bytes.Buffer{}
	if _, err := stdcopy.StdCopy(out, io.Discard, attached.Reader); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// listening are the tcp ports the container has a server on.
func listening(ctx context.Context, cli *dockerClient.Client, containerID string) []int {
	var ports []int
	for _, table := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		raw, err := execIn(ctx, cli, containerID, []string{"cat", table})
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n") {
			fields := strings.Fields(line)
			// the fourth column is the state, and 0A is listening
			if len(fields) < 4 || fields[3] != "0A" {
				continue
			}
			if _, hex, found := strings.Cut(fields[1], ":"); found {
				if n, err := strconv.ParseUint(hex, 16, 32); err == nil {
					ports = append(ports, int(n))
				}
			}
		}
	}
	return ports
}

// watch lets the program run for up to seconds. A server opens some of what it needs only when it
// answers, so every port it listens on, or the image says it does, is asked for its front page, and
// the time is cut short once they have been asked and nothing new has shown up.
func watch(ctx context.Context, cli *dockerClient.Client, containerID string, tr *tracer, exposed []int, seconds int, verbose bool) {
	deadline := time.Now().Add(time.Duration(seconds) * time.Second)
	asked := map[int]bool{}
	var settled time.Time
	// a program that is still starting goes on writing what it opens, and the time to stop is when
	// it has not for a while
	var size int64 = -1
	grew := time.Now()
	// the container is the one asked, and it is the one that is being looked at
	client := http.Client{
		Timeout:   15 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}

	for time.Now().Before(deadline) {
		info, err := cli.ContainerInspect(ctx, containerID)
		if err != nil || !info.State.Running {
			return
		}
		host := info.NetworkSettings.IPAddress

		fresh := false
		for _, port := range append(listening(ctx, cli, containerID), exposed...) {
			if asked[port] {
				continue
			}
			address := net.JoinHostPort(host, strconv.Itoa(port))
			conn, err := net.DialTimeout("tcp", address, 2*time.Second)
			if err != nil {
				continue
			}
			conn.Close()
			asked[port] = true
			fresh = true
			if verbose {
				fmt.Printf("asking %s for /\n", address)
			}
			if response, err := client.Get("http://" + address + "/"); err == nil {
				io.Copy(io.Discard, response.Body)
				response.Body.Close()
			}
		}

		if fresh {
			settled = time.Now().Add(5 * time.Second)
		}
		if stat, err := os.Stat(filepath.Join(tr.dir, "files")); err == nil && stat.Size() != size {
			size = stat.Size()
			grew = time.Now()
		}
		if len(asked) > 0 && time.Now().After(settled) && time.Since(grew) > 10*time.Second {
			return
		}
		time.Sleep(2 * time.Second)
	}
}

var execveLine = regexp.MustCompile(`^(?:\[pid +(\d+)\] +|(\d+) +)execve\("((?:[^"\\]|\\.)*)", .*\) = 0 *$`)
var exitedLine = regexp.MustCompile(`^(?:\[pid +(\d+)\] +|(\d+) +)\+\+\+ (?:exited|killed)`)

// finalProgram is the program the image ends up running: the last one started that is still running
// when the trace ends, or the last one started when none is. An entrypoint script starts tools, and
// then the program, and the program is what a unikernel runs. It gives the process, the path the
// program was started by and the line of the trace where it was.
func finalProgram(trace []byte) (pid string, program string, line int, found bool) {
	type started struct {
		program string
		line    int
	}
	last := map[string]started{}
	gone := map[string]bool{}

	for i, text := range strings.Split(string(trace), "\n") {
		if m := execveLine.FindStringSubmatch(text); m != nil {
			name := m[3]
			if unquoted, err := strconv.Unquote("\"" + name + "\""); err == nil {
				name = unquoted
			}
			last[m[1]+m[2]] = started{program: name, line: i}
			delete(gone, m[1]+m[2])
		} else if m := exitedLine.FindStringSubmatch(text); m != nil {
			gone[m[1]+m[2]] = true
		}
	}

	best, alive := -1, false
	for process, s := range last {
		running := !gone[process]
		if (running && !alive) || (running == alive && s.line > best) {
			pid, program, best, alive = process, s.program, s.line, running
		}
	}

	return pid, program, best, best >= 0
}

// splitNul reads what /proc keeps as strings ended by a zero byte.
func splitNul(raw []byte) []string {
	var parts []string
	for _, part := range strings.Split(string(raw), "\x00") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

// belongsToTheContainer are the variables a container is given by the runtime or its shell, which
// the program of an image does not ask for.
var belongsToTheContainer = []string{"HOSTNAME", "SHLVL", "OLDPWD", "_"}

// ExtractFromDockerImageByTrace creates a package from a docker image by running it and carrying the
// files its program reaches. The image is run the way it starts, and the program is the one that is
// left running at the end of it, unless targetExecutable is given: that followed by args is run
// instead.
func ExtractFromDockerImageByTrace(imageName string, packageName string, parch string, targetExecutable string, quiet bool, verbose bool, seconds int, args []string) (string, string) {
	packageName, version := packageNameAndVersion(imageName, packageName)
	// like docker if the user doesn't provide version of the image we consider "latest" as the version
	if version == "" {
		version = "latest"
	}

	ctx := context.Background()
	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}

	if !strings.Contains(imageName, ":") {
		imageName += ":latest"
	}
	if err := pullImage(ctx, cli, imageName, true, quiet); err != nil {
		log.Fatal(err)
	}

	argv, workingDir, env, exposed, system, err := imageProgram(ctx, cli, imageName)
	if err != nil {
		log.Fatal(err)
	}
	if system != "" && system != "linux" {
		log.Fatalf("%s is an image for %s, and a unikernel here runs what was made for linux", imageName, system)
	}
	if targetExecutable != "" {
		argv = append([]string{targetExecutable}, args...)
	} else {
		argv = append(argv, args...)
	}
	if len(argv) == 0 {
		log.Fatalf("%s starts nothing: say what to run with --file", imageName)
	}

	tr, err := prepareTracer(ctx, cli)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tr.dir)

	created, err := cli.ContainerCreate(ctx, &dockerContainer.Config{
		Image:      imageName,
		Entrypoint: tr.wrap(argv),
		Cmd:        []string{},
	}, &dockerContainer.HostConfig{
		Binds:       []string{tr.bind()},
		CapAdd:      []string{"SYS_PTRACE"},
		SecurityOpt: []string{"seccomp=unconfined"},
	}, nil, nil, "")
	if err != nil {
		log.Fatal(err)
	}
	defer cli.ContainerRemove(ctx, created.ID, dockerContainer.RemoveOptions{Force: true})

	if err := cli.ContainerStart(ctx, created.ID, dockerContainer.StartOptions{}); err != nil {
		log.Fatal(err)
	}

	watch(ctx, cli, created.ID, tr, exposed, seconds, verbose)

	// the list is written as the program goes, so what it has opened so far is in it
	trace := tr.trace()
	pid, program, from, found := finalProgram(trace)
	if !found {
		log.Fatalf("%s started no program in %d seconds", imageName, seconds)
	}
	if verbose {
		fmt.Printf("the program is %s, process %s\n", program, pid)
	}

	// what the process says about itself, while it is there to ask: the command line it was given
	// by the script that started it, the variables it was given, and where it is
	runningEnv := map[string]string{}
	cwd := ""
	exe := ""
	if info, err := cli.ContainerInspect(ctx, created.ID); err == nil && info.State.Running {
		base := "/proc/" + pid + "/"
		if raw, err := execIn(ctx, cli, created.ID, []string{"cat", base + "cmdline"}); err == nil && len(raw) > 0 {
			argv = splitNul(raw)
		}
		if raw, err := execIn(ctx, cli, created.ID, []string{"cat", base + "environ"}); err == nil && len(raw) > 0 {
			env = splitNul(raw)
		}
		if raw, err := execIn(ctx, cli, created.ID, []string{"readlink", base + "cwd"}); err == nil {
			cwd = strings.TrimSpace(string(raw))
		}
		if raw, err := execIn(ctx, cli, created.ID, []string{"readlink", base + "exe"}); err == nil {
			exe = strings.TrimSpace(string(raw))
		}
	}
	if cwd == "" {
		cwd = workingDir
	}
	for _, e := range env {
		if name, value, ok := strings.Cut(e, "="); ok && !containsString(belongsToTheContainer, name) {
			runningEnv[name] = value
		}
	}

	touched, executed := touchedPaths(trace, cwd, from)
	if verbose {
		fmt.Printf("the program opened %d files\n", len(touched))
	}

	tempDirectory, err := os.MkdirTemp("", "*")
	if err != nil {
		log.Fatal(err)
	}
	sysroot := tempDirectory + "/sysroot"
	if err := os.MkdirAll(sysroot, 0755); err != nil {
		log.Fatal(err)
	}

	copyTouched(ctx, cli, created.ID, sysroot, touched, verbose)
	carryInterpreters(ctx, cli, created.ID, sysroot, executed, verbose)

	if !strings.HasPrefix(program, "/") {
		program = whereIs(sysroot, program, env)
	}
	// the file the program was started through can be a link, and a program finds its libraries
	// beside the file it really is
	if exe != "" && exe != program {
		if _, err := os.Stat(filepath.Join(sysroot, exe)); err == nil {
			program = exe
		}
	}
	if _, err := os.Stat(filepath.Join(sysroot, program)); err != nil {
		log.Fatalf("%s did not open %s: say it with --file", imageName, program)
	}

	c := &types.Config{
		Program: program,
		Args:    append([]string{program}, argv[1:]...),
		Version: version,
		Env:     runningEnv,
	}

	if configs := configurationRead(sysroot); !quiet && len(configs) > 0 {
		fmt.Println("the program read these files of configuration, which the config of the user can replace with its own:")
		for i, name := range configs {
			if i == 20 {
				fmt.Printf("  and %d more\n", len(configs)-i)
				break
			}
			fmt.Println("  " + name)
		}
	}

	return writePackage(tempDirectory, packageName, parch, c)
}

// configurationRead says which of the files the package carries are the ones a user changes: what
// the program read from /etc, and the files named as configuration. A file that is put in the
// image with the same name in the configuration of the user replaces the one in the package.
func configurationRead(sysroot string) []string {
	var found []string

	filepath.Walk(sysroot, func(file string, info os.FileInfo, err error) error {
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		name := "/" + filepath.ToSlash(strings.TrimPrefix(file, sysroot+"/"))
		switch {
		case strings.HasPrefix(name, "/etc/ssl/certs/"), strings.HasPrefix(name, "/etc/ld.so"), name == "/etc/hosts", name == "/etc/hostname":
		case strings.HasPrefix(name, "/etc/"):
			found = append(found, name)
		case containsString([]string{".conf", ".cnf", ".ini", ".yaml", ".yml", ".toml", ".properties"}, path.Ext(name)):
			found = append(found, name)
		}
		return nil
	})

	sort.Strings(found)

	return found
}
