package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/coreos/go-iptables/iptables"
	"github.com/docker/docker/pkg/reexec"
	"github.com/vishvananda/netlink"
)

func init() {
	reexec.Register("nsInitialisation", nsInitialisation)
	if reexec.Init() {
		os.Exit(0)
	}
}

func main() {
	rootfsPath := "/home/war-turtle/Desktop/ns-process/rootfs"
	fmt.Println("Running: /bin/sh")
	cmd := reexec.Command("nsInitialisation", rootfsPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = []string{"PS1=-[ns-process]-# "}
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

	fmt.Println(cmd.Process.Pid)

	vethLinkAttrs := netlink.NewLinkAttrs()
	vethLinkAttrs.Name = "vm1"

	veth := &netlink.Veth{
		LinkAttrs: vethLinkAttrs,
		PeerName:  "vm2",
	}

	netlink.LinkAdd(veth)

	ln, _ := netlink.LinkByName("vm2")
	fmt.Println(ln)

	if err := netlink.LinkSetNsPid(ln, cmd.Process.Pid); err != nil {
		fmt.Printf("Error in setting ns by pid %s\n", err)
		os.Exit(1)
	}

	lnvm1, _ := netlink.LinkByName("vm1")

	addr, _ := netlink.ParseAddr("10.200.1.1/24")
	netlink.AddrAdd(lnvm1, addr)
	netlink.LinkSetUp(lnvm1)

	iptable, _ := iptables.New()
	iptable.ChangePolicy("filter", "FORWARD", "DROP")

	if err := iptable.ClearChain("filter", "FORWARD"); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	chainString, _ := iptable.ListChains("nat")
	for _, chain := range chainString {
		iptable.ClearChain("nat", chain)
	}

	if err := iptable.AppendUnique("nat", "POSTROUTING", "-s", "10.200.1.0/255.255.255.0", "-o", "enp3s0", "-j", "MASQUERADE"); err != nil {
		fmt.Println(err)
	}

	if err := iptable.AppendUnique("filter", "FORWARD", "-i", "enp3s0", "-o", "vm1", "-j", "ACCEPT"); err != nil {
		fmt.Println(err)
	}

	if err := iptable.AppendUnique("filter", "FORWARD", "-o", "enp3s0", "-i", "vm1", "-j", "ACCEPT"); err != nil {
		fmt.Println(err)
	}

	if err := cmd.Wait(); err != nil {
		fmt.Println("Error Waiting for process")
		os.Exit(1)
	}
}

func nsInitialisation() {
	newrootPath := os.Args[1]

	if err := mountProc(newrootPath); err != nil {
		fmt.Printf("Error mounting /proc - %s\n", err)
		os.Exit(1)
	}

	if err := pivotRoot(newrootPath); err != nil {
		fmt.Printf("Error running pivot_root - %s\n", err)
		os.Exit(1)
	}

	lnvm2, _ := netlink.LinkByName("vm2")
	addr, _ := netlink.ParseAddr("10.200.1.2/24")
	netlink.AddrAdd(lnvm2, addr)
	netlink.LinkSetUp(lnvm2)

	// addrvm1 := net.IPNet{
	// 	IP:   []byte{10, 200, 1, 1},
	// 	Mask: []byte{255, 255, 255, 0},
	// }

	route := netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: lnvm2.Attrs().Index,
		Gw:        []byte{10, 200, 1, 1},
	}
	netlink.RouteAdd(&route)

	nsRun()
}

func nsRun() {
	cmd := exec.Command("/bin/sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = []string{"PS1=ns-process#"}

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running the command %s\n", err)
		os.Exit(1)
	}
}

func pivotRoot(newroot string) error {
	putold := filepath.Join(newroot, "/.pivot_root")

	// bind mount newroot to itself - this is a slight hack
	// needed to work around a pivot_root requirement
	if err := syscall.Mount(
		newroot,
		newroot,
		"",
		syscall.MS_BIND|syscall.MS_REC,
		"",
	); err != nil {
		return err
	}

	// create putold directory
	if err := os.MkdirAll(putold, 0700); err != nil {
		return err
	}

	// call pivot_root
	if err := syscall.PivotRoot(newroot, putold); err != nil {
		return err
	}

	// ensure current working directory is set to new root
	if err := os.Chdir("/"); err != nil {
		return err
	}

	// umount putold, which now lives at /.pivot_root
	putold = "/.pivot_root"
	if err := syscall.Unmount(
		putold,
		syscall.MNT_DETACH,
	); err != nil {
		return err
	}

	// remove putold
	if err := os.RemoveAll(putold); err != nil {
		return err
	}

	return nil
}

func mountProc(newroot string) error {
	source := "proc"
	target := filepath.Join(newroot, "/proc")
	fstype := "proc"
	flags := 0
	data := ""

	os.MkdirAll(target, 0755)
	if err := syscall.Mount(
		source,
		target,
		fstype,
		uintptr(flags),
		data,
	); err != nil {
		return err
	}

	return nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
