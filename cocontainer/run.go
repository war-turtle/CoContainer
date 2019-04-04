package cocontainer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
)

func init() {
	reexec.Register("nsInitialisation", nsInitialisation)
	if reexec.Init() {
		os.Exit(0)
	}
}

func nsInitialisation() {
	newrootPath := os.Args[1]
	commandString := os.Args[2]
	vethOnContainer := os.Args[3]

	if err := mountProc(newrootPath); err != nil {
		fmt.Printf("Error mounting /proc - %s\n", err)
		os.Exit(1)
	}

	if err := pivotRoot(newrootPath); err != nil {
		fmt.Printf("Error running pivot_root - %s\n", err)
		os.Exit(1)
	}

	vethSetupOnContainer(vethOnContainer)

	nsRun(commandString)
}

func nsRun(command string) {
	cmd := exec.Command(command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = []string{"PS1=-[cocontainer]- # "}

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running the command %s\n", err)
		os.Exit(1)
	}
}

//Run is function to run container
func Run(command []string, vethPrefix string) {
	vethOnHost := vethPrefix + "1"
	vethOnContainer := vethPrefix + "2"
	commandString := strings.Join(command, " ")

	rootfsPath := "/home/war-turtle/Desktop/ns-process/rootfs"
	cmd := reexec.Command("nsInitialisation", rootfsPath, commandString, vethOnContainer)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getegid(),
				Size:        1,
			},
		},
	}

	if err := cmd.Start(); err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}

	vethSetupOnHost(cmd.Process.Pid, vethOnHost, vethOnContainer)

	// ipTableOnHost()

	if err := cmd.Wait(); err != nil {
		fmt.Println("Error Waiting for process")
		os.Exit(1)
	}
}
