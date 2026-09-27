// Windows 原生 ICMP ping（iphlpapi.dll IcmpSendEcho）
// 不启动外部进程 → GUI 下不弹 cmd 窗口、无需管理员权限、比 exec ping 快
package main

import (
	"net"
	"syscall"
	"unsafe"
)

var (
	icmpDll         = syscall.NewLazyDLL("iphlpapi.dll")
	procIcmpCreate  = icmpDll.NewProc("IcmpCreateFile")
	procIcmpClose   = icmpDll.NewProc("IcmpCloseHandle")
	procIcmpSend    = icmpDll.NewProc("IcmpSendEcho")
)

// pingICMP 对单个 IPv4 执行一次 ICMP echo，超时 timeoutMs 毫秒
func pingICMP(ipStr string, timeoutMs int) bool {
	ip := net.ParseIP(ipStr).To4()
	if ip == nil {
		return false
	}
	// IPAddr = s_addr（网络字节序）：a.b.c.d -> d<<24|c<<16|b<<8|a
	addr := uint32(ip[3])<<24 | uint32(ip[2])<<16 | uint32(ip[1])<<8 | uint32(ip[0])

	h, _, _ := procIcmpCreate.Call()
	if h == 0 {
		return false
	}
	defer procIcmpClose.Call(h)

	data := []byte("NetDoctor-ping")
	reply := make([]byte, 2048) // ICMP_ECHO_REPLY 数组缓冲
	r, _, _ := procIcmpSend.Call(
		h,
		uintptr(addr),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		0,
		uintptr(unsafe.Pointer(&reply[0])),
		uintptr(len(reply)),
		uintptr(timeoutMs),
	)
	if r == 0 {
		return false
	}
	// 首个 reply：偏移 4 处为 Status（IP_SUCCESS=0 表示收到应答）
	status := *(*uint32)(unsafe.Pointer(&reply[4]))
	return status == 0
}

// resolveV4 域名/地址 → IPv4 字符串（DNS 由 Go 自身解析，不调用外部进程）
func resolveV4(host string) string {
	if ip := net.ParseIP(host); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			return v4.String()
		}
		return ""
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return ""
	}
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			return v4.String()
		}
	}
	return ""
}
