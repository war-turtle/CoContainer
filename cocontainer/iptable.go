package cocontainer

import (
	"fmt"
	"os"

	"github.com/coreos/go-iptables/iptables"
)

func ipTableOnHost() {
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
}
