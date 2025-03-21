package nets

import (
	"net"
	"strings"
)

// var privateBlocks []*net.IPNet

// const (
// 	// Ipv4SplitCharacter use for slipt Ipv4
// 	Ipv4SplitCharacter = "."
// 	// Ipv6SplitCharacter use for slipt Ipv6
// 	Ipv6SplitCharacter = ":"
// )

// // GetLocalIP get local ip
// func GetLocalIP() (string, error) {
// 	faces, err := net.Interfaces()
// 	if err != nil {
// 		return "", err
// 	}

// 	var addr net.IP
// 	for _, face := range faces {
// 		if !isValidNetworkInterface(face) {
// 			continue
// 		}

// 		addrs, err := face.Addrs()
// 		if err != nil {
// 			return "", err
// 		}

// 		if ipv4, ok := getValidIPv4(addrs); ok {
// 			addr = ipv4
// 			if isPrivateIP(ipv4) {
// 				return ipv4.String(), nil
// 			}
// 		}
// 	}

// 	if addr == nil {
// 		return "", errors.New("can not get local IP")
// 	}

// 	return addr.String(), nil
// }

// func isPrivateIP(ip net.IP) bool {
// 	for _, priv := range privateBlocks {
// 		if priv.Contains(ip) {
// 			return true
// 		}
// 	}
// 	return false
// }

// func getValidIPv4(addrs []net.Addr) (net.IP, bool) {
// 	for _, addr := range addrs {
// 		var ip net.IP

// 		switch v := addr.(type) {
// 		case *net.IPNet:
// 			ip = v.IP
// 		case *net.IPAddr:
// 			ip = v.IP
// 		}

// 		if ip == nil || ip.IsLoopback() {
// 			continue
// 		}

// 		ip = ip.To4()
// 		if ip == nil {
// 			// not an valid ipv4 address
// 			continue
// 		}

// 		return ip, true
// 	}
// 	return nil, false
// }

// func isValidNetworkInterface(face net.Interface) bool {
// 	if face.Flags&net.FlagUp == 0 {
// 		// interface down
// 		return false
// 	}

// 	if face.Flags&net.FlagLoopback != 0 {
// 		// loopback interface
// 		return false
// 	}

// 	if strings.Contains(strings.ToLower(face.Name), "docker") {
// 		return false
// 	}

// 	return true
// }

var localIP = getLocalIP()

func GetLocalIP() string {
	return localIP
}

// GetLocalIP returns the non loopback local IP of the host
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// IsSameAddr refer from https://github.com/facebookarchive/grace/blob/master/gracenet/net.go#L180
func IsSameAddr(addr1, addr2 net.Addr) bool {
	if addr1.Network() != addr2.Network() {
		return false
	}

	addr1s := addr1.String()
	addr2s := addr2.String()
	if addr1s == addr2s {
		return true
	}

	// This allows for ipv6 vs ipv4 local addresses to compare as equal. This
	// scenario is common when listening on localhost.
	const ipv6prefix = "[::]"
	addr1s = strings.TrimPrefix(addr1s, ipv6prefix)
	addr2s = strings.TrimPrefix(addr2s, ipv6prefix)
	const ipv4prefix = "0.0.0.0"
	addr1s = strings.TrimPrefix(addr1s, ipv4prefix)
	addr2s = strings.TrimPrefix(addr2s, ipv4prefix)
	return addr1s == addr2s
}
