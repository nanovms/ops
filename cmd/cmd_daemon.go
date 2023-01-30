package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
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

func daemonizeCommandHandler(cmd *cobra.Command, args []string) {
	fmt.Println("Note: If on a mac this expects ops to have suid bit set for networking.")
	fmt.Println("if you used the installer you are set otherwise run the following command\n" +
		"\tsudo chown -R root /usr/local/bin/qemu-system-x86_64\n" +
		"\tsudo chmod u+s /usr/local/bin/qemu-system-x86_64")
	fmt.Println("daemonizing")
}
