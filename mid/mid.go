package mid

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func Hostname() string {
	hn, e := os.Hostname()
	if e != nil {
		return "[" + e.Error() + "]"
	}
	return hn
}

func HardwareAddr(ip string) string {
	ss := "00:00:00:00:00:00"
	s := ""
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if s = iface.HardwareAddr.String(); s != "" && len(s) <= len(ss) {
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4().String() == ip {
						return s
					}
				}
			}
			if ss == "00:00:00:00:00:00" {
				ss = s
			}
		}
	}
	if s == "" {
		s = GetLocalIP()
		var a, b, c, d int
		fmt.Sscanf(s+"\000", "%d.%d.%d.%d\000", &a, &b, &c, &d)
		ss = fmt.Sprintf("FF:FF:%02x:%02x:%02x:%02x", a, b, c, d)
	}
	return ss
}

// GetLocalIP returns the local IP address or loopback IP.
func GetLocalIP() string {
	return localHostIP()
}

const loopbackIP = "127.0.0.1"

func GetLocalIPs() []string {
	ips := []string{}
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	if len(ips) == 0 {
		return []string{loopbackIP}
	}
	return ips
}

// LocalIP returns the local IP address.
func localHostIP() string {
	ss := loopbackIP
	sshconn := os.Getenv("SSH_CONNECTION")
	if sshconn != "" {
		sshconns := strings.Split(sshconn, " ")
		if len(sshconns) == 4 {
			ss = sshconns[2]
		}
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ss
	}
	fs := []string{}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				if ipnet.IP.String() == ss {
					return ss
				}
				fs = append(fs, ipnet.IP.String())
			}
		}
	}
	if len(fs) > 0 {
		return fs[0]
	}
	return ss
}

func IsLocalIP(ip string) bool {
	if loopbackIP == ip {
		return true
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				if ipnet.IP.String() == ip {
					return true
				}
			}
		}
	}

	return false
}
