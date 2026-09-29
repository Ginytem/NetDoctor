// Windows 原生 ICMP ping（iphlpapi.dll IcmpSendEcho）
// 不启动外部进程 → GUI 下不弹 cmd 窗口、无需管理员权限、比 exec ping 快
package main

import (
	"fmt"
	"net"
	"strings"
	"syscall"
	"unsafe"
)

var (
	icmpDll        = syscall.NewLazyDLL("iphlpapi.dll")
	procIcmpCreate = icmpDll.NewProc("IcmpCreateFile")
	procIcmpClose  = icmpDll.NewProc("IcmpCloseHandle")
	procIcmpSend   = icmpDll.NewProc("IcmpSendEcho")
)

// pingICMP 对单个 IPv4 执行一次 ICMP echo，超时 timeoutMs 毫秒
// 返回 (是否收到应答, 往返毫秒, TTL)
func pingICMP(ipStr string, timeoutMs int) (bool, uint32, byte) {
	ip := net.ParseIP(ipStr).To4()
	if ip == nil {
		return false, 0, 0
	}
	// IPAddr = s_addr（网络字节序）：a.b.c.d -> d<<24|c<<16|b<<8|a
	addr := uint32(ip[3])<<24 | uint32(ip[2])<<16 | uint32(ip[1])<<8 | uint32(ip[0])

	h, _, _ := procIcmpCreate.Call()
	if h == 0 {
		return false, 0, 0
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
		return false, 0, 0
	}
	// ICMP_ECHO_REPLY 布局：Address@0(4B) Status@4(4B) RoundTripTime@8(4B)
	// DataSize@12(2B) Reserved@14(2B) Data@16(8B) Options@24(Ttl@24)
	status := *(*uint32)(unsafe.Pointer(&reply[4]))
	rtt := *(*uint32)(unsafe.Pointer(&reply[8]))
	ttl := reply[24]
	return status == 0, rtt, ttl
}

// PingResult 一次 Ping 检测的汇总结果
type PingResult struct {
	Host      string // 用户输入（IP 或域名）
	IP        string // 解析出的 IPv4（域名时才有意义）
	ResolveOK bool   // 地址解析是否成功
	Sent      int    // 发送包数
	Received  int    // 收到应答数
	MinMs     int    // 最小延迟（毫秒）
	MaxMs     int    // 最大延迟（毫秒）
	AvgMs     int    // 平均延迟（毫秒）
	TTL       int    // 最近一次成功应答的 TTL
}

// pingStats 对 IP/域名连续 ping count 次（超时 timeoutMs 毫秒/次），返回汇总
func pingStats(host string, count, timeoutMs int) PingResult {
	r := PingResult{Host: host, Sent: count}
	ip := resolveV4(host)
	if ip == "" {
		return r
	}
	r.IP = ip
	r.ResolveOK = true

	sum := uint64(0)
	received := 0
	for i := 0; i < count; i++ {
		ok, rtt, ttl := pingICMP(ip, timeoutMs)
		if !ok {
			continue
		}
		ms := int(rtt)
		if received == 0 {
			r.MinMs, r.MaxMs = ms, ms
		} else {
			if ms < r.MinMs {
				r.MinMs = ms
			}
			if ms > r.MaxMs {
				r.MaxMs = ms
			}
		}
		sum += uint64(ms)
		r.TTL = int(ttl)
		received++
	}
	r.Received = received
	if received > 0 {
		r.AvgMs = int(sum / uint64(received))
	}
	return r
}

// pingLossPercent 丢包百分比（0~100）
// hostLabel 显示用主机名：纯 IP 输入只显示 IP，域名输入显示"域名（IP）"
func (r *PingResult) hostLabel() string {
	if r.IP == "" || strings.EqualFold(r.Host, r.IP) {
		return r.Host
	}
	return r.Host + "（" + r.IP + "）"
}

func (r *PingResult) lossPercent() int {
	if r.Sent <= 0 {
		return 0
	}
	return (r.Sent - r.Received) * 100 / r.Sent
}

// pingSummary 把 Ping 结果翻译成人话（短行排版：每行控制在面板宽度内，
// 配合自动换行不会在中间折断，也不依赖横向滚动）
func (r *PingResult) pingSummary() string {
	if !r.ResolveOK {
		return fmt.Sprintf("地址解析失败\n地址：%s\n\n可能原因：\n· 地址拼写错误\n· 域名不存在\n· 本机 DNS 设置有问题", r.Host)
	}
	if r.Sent <= 0 {
		return "参数错误：未发起检测。"
	}

	loss := r.lossPercent()
	if r.Received == 0 {
		return fmt.Sprintf("Ping 不通：%s\n4 个数据包全部超时\n\n可能原因：\n· 目标设备未开机或已关机\n· IP 地址或域名写错\n· 目标不在当前网络内\n· 防火墙或安全软件拦截\n· 设备禁用了 ICMP 应答\n（部分设备默认不回，属正常）", r.hostLabel())
	}

	var b strings.Builder
	label := r.hostLabel()
	switch {
	case loss > 0:
		fmt.Fprintf(&b, "网络不稳定：%s\n%d 个包丢失 %d 个（丢包 %d%%）\n可能原因：\n· 无线信号弱\n· 网线接触不良\n· 设备负载高或网络拥塞\n", label, r.Sent, r.Sent-r.Received, loss)
	default:
		switch {
		case r.AvgMs <= 20:
			fmt.Fprintf(&b, "Ping 通了：%s\n4 个数据包全部到达\n网络非常通畅\n平均延迟 %d 毫秒\n", label, r.AvgMs)
		case r.AvgMs <= 100:
			fmt.Fprintf(&b, "Ping 通了：%s\n4 个数据包全部到达\n网络质量良好\n平均延迟 %d 毫秒\n", label, r.AvgMs)
		case r.AvgMs <= 300:
			fmt.Fprintf(&b, "Ping 通了：%s\n目标 IP：%s\n延迟偏高：平均 %d 毫秒\n网络可能有点慢\n（跨地域访问或无线干扰）\n", r.Host, r.IP, r.AvgMs)
		default:
			fmt.Fprintf(&b, "Ping 通了：%s\n目标 IP：%s\n延迟很高：平均 %d 毫秒\n网络明显缓慢\n（严重拥塞或长距离转发）\n", r.Host, r.IP, r.AvgMs)
		}
	}
	if r.Sent > 1 && r.MinMs >= 0 && r.MaxMs >= 0 {
		fmt.Fprintf(&b, "延迟范围：%d~%d 毫秒\n", r.MinMs, r.MaxMs)
	}
	fmt.Fprintf(&b, "TTL %d", r.TTL)
	if r.TTL > 0 && r.TTL < 64 {
		b.WriteString("（经过较多路由器转发）")
	}
	return b.String()
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
