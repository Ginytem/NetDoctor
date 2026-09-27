// 阶段 2：网络核心（本机 IP/掩码/网关/DHCP/活动网卡/上网方式）
// 原版分 MODERN（netsh）/ LEGACY（route print + ipconfig）双分支，
// 本版统一用 GetAdaptersAddresses（Vista+ 原生 API，Win7~Win11 均可），
// 语义与原版一致：选“第一个有非 0.0.0.0 默认网关的已连接接口”为主网卡。
package main

import (
	"fmt"
	"os/exec"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

type NetInfo struct {
	LocalIP    string
	Gateway    string
	Mask       string
	PrefixLen  int
	DHCP       string
	DHCPServer string
	Adapter    string
	NetType    string
}

const (
	gaaFlagsIncludePrefix  = 0x00000010 // GAA_FLAG_INCLUDE_PREFIX
	gaaFlagsIncludeGateway = 0x00000080 // GAA_FLAG_INCLUDE_GATEWAYS（网关默认不返回，必须加此标志）
	ipAdapterDHCPEnabled   = 0x00000004 // IP_ADAPTER_DHCP_ENABLED
	ifTypeIEEE80211        = 71         // IF_TYPE_IEEE80211（Wi-Fi）
	ifOperStatusUp         = 1          // IfOperStatusUp
)

func collectNetwork() NetInfo {
	ni := NetInfo{
		LocalIP:    "127.0.0.1",
		Mask:       "255.255.255.0",
		PrefixLen:  24,
		DHCP:       "手动指定（静态）",
		DHCPServer: "无",
		Adapter:    "以太网",
		NetType:    "有线连接（网线）",
	}

	lip, gw, prefix, dhcpOn, adapterName, ifType, ok := getSelectedAdapter()
	ni.DHCPServer = dhcpServerFromIpconfig()

	if ok {
		if lip != "" {
			ni.LocalIP = lip
		}
		if gw != "" {
			ni.Gateway = gw
		}
		ni.PrefixLen = int(prefix)
		ni.Mask = maskFromPrefix(int(prefix))
		if adapterName != "" {
			ni.Adapter = adapterName
		}
		ni.NetType = "有线连接（网线）"
		if ifType == ifTypeIEEE80211 {
			ni.NetType = "Wi-Fi（无线网络）"
		}
		if dhcpOn {
			ni.DHCP = "自动获取（DHCP）"
		}
	} else if ni.DHCPServer != "无" {
		// API 不可用时的兜底（与原版 LEGACY 分支一致：有 DHCP 服务器即视为 DHCP）
		ni.DHCP = "自动获取（DHCP）"
	}

	// 网关兜底：未取到默认网关时用 DHCP 服务器（原版行为）
	if ni.Gateway == "" && ni.DHCPServer != "无" {
		ni.Gateway = ni.DHCPServer
	}
	// 上网方式：与原版一致，只要存在已连接的 Wi-Fi 就报无线
	if isWifiConnected() {
		ni.NetType = "Wi-Fi（无线网络）"
	}
	return ni
}

// GetAdaptersAddresses：选中第一个“已连接且有非 0.0.0.0 默认网关”的 IPv4 接口
func getSelectedAdapter() (localIP, gateway string, prefix uint8, dhcpEnabled bool, adapterName string, ifType uint32, ok bool) {
	flags := uint32(gaaFlagsIncludePrefix | gaaFlagsIncludeGateway)
	size := uint32(0)
	if err := windows.GetAdaptersAddresses(windows.AF_INET, flags, 0, nil, &size); err != nil {
		if err != windows.ERROR_BUFFER_OVERFLOW {
			return
		}
	}
	if size == 0 {
		return
	}
	buf := make([]byte, size)
	aa := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
	if err := windows.GetAdaptersAddresses(windows.AF_INET, flags, 0, aa, &size); err != nil {
		return
	}
	for p := aa; p != nil; p = p.Next {
		if p.OperStatus != ifOperStatusUp {
			continue
		}
		gw := ""
		for ga := p.FirstGatewayAddress; ga != nil; ga = ga.Next {
			ip, ok := ipv4FromSocketAddress(&ga.Address)
			if !ok {
				continue
			}
			if ip != "0.0.0.0" && ip != "on-link" {
				gw = ip
				break
			}
		}
		if gw == "" {
			continue
		}
		lip := ""
		var pre uint8
		for ua := p.FirstUnicastAddress; ua != nil; ua = ua.Next {
			ip, ok := ipv4FromSocketAddress(&ua.Address)
			if !ok {
				continue
			}
			lip = ip
			pre = ua.OnLinkPrefixLength
			break
		}
		return lip, gw, pre, p.Flags&ipAdapterDHCPEnabled != 0,
			windows.UTF16PtrToString(p.FriendlyName), p.IfType, true
	}
	return
}

// SocketAddress.Sockaddr 为 *syscall.RawSockaddrAny，Data[2..5] 即 IPv4 地址
// 注意：syscall.RawSockaddr.Data 是 [14]int8（有符号），需转 uint8 再拼 IP
func ipv4FromSocketAddress(sa *windows.SocketAddress) (string, bool) {
	if sa == nil || sa.Sockaddr == nil || sa.Sockaddr.Addr.Family != windows.AF_INET {
		return "", false
	}
	d := sa.Sockaddr.Addr.Data
	return fmt.Sprintf("%d.%d.%d.%d", uint8(d[2]), uint8(d[3]), uint8(d[4]), uint8(d[5])), true
}

// 前缀长度 -> 子网掩码（原版只有固定表，本版通用计算，/23 等也能正确显示）
func maskFromPrefix(prefix int) string {
	if prefix <= 0 {
		return "0.0.0.0"
	}
	if prefix >= 32 {
		return "255.255.255.255"
	}
	m := ^uint32(0) << (32 - uint(prefix))
	return fmt.Sprintf("%d.%d.%d.%d", byte(m>>24), byte(m>>16), byte(m>>8), byte(m))
}

// DHCP 服务器：ipconfig /all 解析（原版方式，中英文双匹配）
func dhcpServerFromIpconfig() string {
	cmd := exec.Command("ipconfig", "/all")
	hideCmd(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "无"
	}
	text := decodeGBK(out)
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "DHCP 服务器") && !strings.Contains(line, "DHCP Server") {
			continue
		}
		idx := strings.LastIndex(line, ":")
		if idx < 0 {
			continue
		}
		ip := firstToken(strings.TrimSpace(line[idx+1:]))
		if isIPv4Format(ip) && ip != "0.0.0.0" && ip != "255.255.255.255" {
			return ip
		}
	}
	return "无"
}

// 上网方式：只要存在已连接的 Wi-Fi 接口即报无线（原版 netsh wlan 判定）
func isWifiConnected() bool {
	cmd := exec.Command("netsh", "wlan", "show", "interfaces")
	hideCmd(cmd)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	text := strings.ToLower(decodeGBK(out))
	return strings.Contains(text, "已连接") || strings.Contains(text, "connected")
}
