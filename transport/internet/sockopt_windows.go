package internet

import (
	"context"
	"encoding/binary"
	"net"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/xtls/xray-core/common/errors"
	"golang.org/x/sys/windows"
)

var (
	modIphlpapi                     = windows.NewLazySystemDLL("iphlpapi.dll")
	procConvertInterfaceAliasToLuid = modIphlpapi.NewProc("ConvertInterfaceAliasToLuid")
	procConvertInterfaceLuidToIndex = modIphlpapi.NewProc("ConvertInterfaceLuidToIndex")
)

// interfaceIndex resolves an interface alias to its index. Called for every
// outbound socket, so it uses the two direct IP Helper lookups instead of
// net.InterfaceByName (a full GetAdaptersAddresses walk, tens of ms).
func interfaceIndex(name string) (int, error) {
	if alias, err := windows.UTF16PtrFromString(name); err == nil {
		var luid uint64
		if r1, _, _ := procConvertInterfaceAliasToLuid.Call(uintptr(unsafe.Pointer(alias)), uintptr(unsafe.Pointer(&luid))); r1 == 0 {
			var index uint32
			if r1, _, _ := procConvertInterfaceLuidToIndex.Call(uintptr(unsafe.Pointer(&luid)), uintptr(unsafe.Pointer(&index))); r1 == 0 && index != 0 {
				return int(index), nil
			}
		}
	}
	inf, err := net.InterfaceByName(name)
	if err != nil {
		return 0, err
	}
	return inf.Index, nil
}

const (
	TCP_FASTOPEN    = 15
	IP_UNICAST_IF   = 31
	IPV6_UNICAST_IF = 31
)

func setTFO(fd syscall.Handle, tfo int) error {
	if tfo > 0 {
		tfo = 1
	}
	if tfo >= 0 {
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_FASTOPEN, tfo); err != nil {
			return err
		}
	}
	return nil
}

func applyOutboundSocketOptions(network string, address string, fd uintptr, config *SocketConfig) error {
	if config.Interface != "" {
		infIndex, err := interfaceIndex(config.Interface)
		if err != nil {
			return errors.New("failed to find the interface").Base(err)
		}
		// easy way to check if the address is ipv4
		isV4 := strings.Contains(address, ".")
		// note: DO NOT trust the passed network variable, it can be udp6 even if the address is ipv4
		// because operating system might(always) use ipv6 socket to process ipv4
		host, _, err := net.SplitHostPort(address)
		if isV4 {
			var bytes [4]byte
			binary.BigEndian.PutUint32(bytes[:], uint32(infIndex))
			idx := *(*uint32)(unsafe.Pointer(&bytes[0]))
			if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IP, IP_UNICAST_IF, int(idx)); err != nil {
				return errors.New("failed to set IP_UNICAST_IF").Base(err)
			}
			if ip := net.ParseIP(host); ip != nil && ip.IsMulticast() && isUDPSocket(network) {
				if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IP, syscall.IP_MULTICAST_IF, int(idx)); err != nil {
					return errors.New("failed to set IP_MULTICAST_IF").Base(err)
				}
			}
		} else {
			if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IPV6, IPV6_UNICAST_IF, infIndex); err != nil {
				return errors.New("failed to set IPV6_UNICAST_IF").Base(err)
			}
			if ip := net.ParseIP(host); ip != nil && ip.IsMulticast() && isUDPSocket(network) {
				if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IPV6, syscall.IPV6_MULTICAST_IF, infIndex); err != nil {
					return errors.New("failed to set IPV6_MULTICAST_IF").Base(err)
				}
			}
		}
	}

	if isTCPSocket(network) {
		if err := setTFO(syscall.Handle(fd), config.ParseTFOValue()); err != nil {
			return err
		}
		if config.TcpKeepAliveIdle > 0 {
			if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1); err != nil {
				return errors.New("failed to set SO_KEEPALIVE", err)
			}
		} else if config.TcpKeepAliveIdle < 0 {
			if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 0); err != nil {
				return errors.New("failed to unset SO_KEEPALIVE", err)
			}
		}
	}

	if len(config.CustomSockopt) > 0 {
		for _, custom := range config.CustomSockopt {
			if custom.System != "" && custom.System != runtime.GOOS {
				errors.LogDebug(context.Background(), "CustomSockopt system not match: ", "want ", custom.System, " got ", runtime.GOOS)
				continue
			}
			// Skip unwanted network type
			// network might be tcp4 or tcp6
			// use HasPrefix so that "tcp" can match tcp4/6 with "tcp" if user want to control all tcp (udp is also the same)
			// if it is empty, strings.HasPrefix will always return true to make it apply for all networks
			if !strings.HasPrefix(network, custom.Network) {
				continue
			}
			var level = 0x6 // default TCP
			var opt int
			if len(custom.Opt) == 0 {
				return errors.New("No opt!")
			} else {
				opt, _ = strconv.Atoi(custom.Opt)
			}
			if custom.Level != "" {
				level, _ = strconv.Atoi(custom.Level)
			}
			if custom.Type == "int" {
				value, _ := strconv.Atoi(custom.Value)
				if err := syscall.SetsockoptInt(syscall.Handle(fd), level, opt, value); err != nil {
					return errors.New("failed to set CustomSockoptInt", opt, value, err)
				}
			} else if custom.Type == "str" {
				return errors.New("failed to set CustomSockoptString: Str type does not supported on windows")
			} else {
				return errors.New("unknown CustomSockopt type:", custom.Type)
			}
		}
	}

	return nil
}

func applyInboundSocketOptions(network string, fd uintptr, config *SocketConfig) error {
	if isTCPSocket(network) {
		if err := setTFO(syscall.Handle(fd), config.ParseTFOValue()); err != nil {
			return err
		}
		if config.TcpKeepAliveIdle > 0 {
			if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1); err != nil {
				return errors.New("failed to set SO_KEEPALIVE", err)
			}
		} else if config.TcpKeepAliveIdle < 0 {
			if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 0); err != nil {
				return errors.New("failed to unset SO_KEEPALIVE", err)
			}
		}
	}

	if config.V6Only {
		if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IPV6, syscall.IPV6_V6ONLY, 1); err != nil {
			return errors.New("failed to set IPV6_V6ONLY").Base(err)
		}
	}

	if len(config.CustomSockopt) > 0 {
		for _, custom := range config.CustomSockopt {
			if custom.System != "" && custom.System != runtime.GOOS {
				errors.LogDebug(context.Background(), "CustomSockopt system not match: ", "want ", custom.System, " got ", runtime.GOOS)
				continue
			}
			// Skip unwanted network type
			// network might be tcp4 or tcp6
			// use HasPrefix so that "tcp" can match tcp4/6 with "tcp" if user want to control all tcp (udp is also the same)
			// if it is empty, strings.HasPrefix will always return true to make it apply for all networks
			if !strings.HasPrefix(network, custom.Network) {
				continue
			}
			var level = 0x6 // default TCP
			var opt int
			if len(custom.Opt) == 0 {
				return errors.New("No opt!")
			} else {
				opt, _ = strconv.Atoi(custom.Opt)
			}
			if custom.Level != "" {
				level, _ = strconv.Atoi(custom.Level)
			}
			if custom.Type == "int" {
				value, _ := strconv.Atoi(custom.Value)
				if err := syscall.SetsockoptInt(syscall.Handle(fd), level, opt, value); err != nil {
					return errors.New("failed to set CustomSockoptInt", opt, value, err)
				}
			} else if custom.Type == "str" {
				return errors.New("failed to set CustomSockoptString: Str type does not supported on windows")
			} else {
				return errors.New("unknown CustomSockopt type:", custom.Type)
			}
		}
	}

	return nil
}

func bindAddr(fd uintptr, ip []byte, port uint32) error {
	return nil
}

func setReuseAddr(fd uintptr) error {
	return nil
}

func setReusePort(fd uintptr) error {
	return nil
}
