package cocontainer

import (
	"fmt"
	"os"

	"github.com/vishvananda/netlink"
)

func vethSetupOnHost(pid int, vethOnHost, vethOnContainer string) {
	vethLinkAttrs := netlink.NewLinkAttrs()
	vethLinkAttrs.Name = vethOnHost

	veth := &netlink.Veth{
		LinkAttrs: vethLinkAttrs,
		PeerName:  vethOnContainer,
	}

	netlink.LinkAdd(veth)

	ln, _ := netlink.LinkByName(vethOnContainer)
	fmt.Println(ln)

	if err := netlink.LinkSetNsPid(ln, pid); err != nil {
		fmt.Printf("Error in setting ns by pid %s\n", err)
		os.Exit(1)
	}

	lnvm1, _ := netlink.LinkByName(vethOnHost)

	addr, _ := netlink.ParseAddr("10.200.1.1/24")
	netlink.AddrAdd(lnvm1, addr)
	netlink.LinkSetUp(lnvm1)
}

func vethSetupOnContainer(vethOnContainer string) {
	lnvm2, _ := netlink.LinkByName(vethOnContainer)
	addr, _ := netlink.ParseAddr("10.200.1.2/24")
	netlink.AddrAdd(lnvm2, addr)
	netlink.LinkSetUp(lnvm2)

	route := netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: lnvm2.Attrs().Index,
		Gw:        []byte{10, 200, 1, 1},
	}
	netlink.RouteAdd(&route)
}
