// 阶段 3：扫描探测（外网连通/公网 IP/局域网设备/相机/空闲 IP）
// ping / arp 沿用原版命令判定；空闲 IP 扫描改为并发（原版串行最坏数十秒）
package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// ---------- 编码 ----------
func decodeGBK(b []byte) string {
	dec := simplifiedchinese.GBK.NewDecoder()
	s, err := dec.String(string(b))
	if err != nil {
		return string(b)
	}
	return s
}

// ---------- IP 工具 ----------
func isIPv4Format(s string) bool {
	if s == "" {
		return false
	}
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "" || len(p) > 3 {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
		n, _ := strconv.Atoi(p)
		if n > 255 {
			return false
		}
	}
	return true
}

// 局域网地址判定：192.168.* / 10.* / 172.16-31.*（原版前缀匹配）
func isLANIP(ip string) bool {
	if !isIPv4Format(ip) {
		return false
	}
	parts := strings.Split(ip, ".")
	a, _ := strconv.Atoi(parts[0])
	b, _ := strconv.Atoi(parts[1])
	switch {
	case a == 192 && b == 168:
		return true
	case a == 10:
		return true
	case a == 172 && b >= 16 && b <= 31:
		return true
	}
	return false
}

func segOf(ip string) string {
	if !isIPv4Format(ip) {
		return ""
	}
	parts := strings.Split(ip, ".")
	return parts[0] + "." + parts[1] + "." + parts[2]
}

func lastOctet(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ""
	}
	return parts[3]
}

func firstToken(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}

// ---------- ping / arp ----------
// pingOK 纯 Windows ICMP API 实现：GUI 下不弹 cmd 窗口，无需管理员权限
func pingOK(host string, n, timeoutMs int) bool {
	ip := host
	if !isIPv4Format(host) {
		ip = resolveV4(host)
		if ip == "" {
			return false
		}
	}
	for i := 0; i < n; i++ {
		ok, _, _ := pingICMP(ip, timeoutMs)
		if ok {
			return true
		}
	}
	return false
}

// hideCmd 隐藏子进程控制台窗口（CREATE_NO_WINDOW），防止 GUI 下弹 cmd 黑框
func hideCmd(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}

// arp -a <ip> 是否存在动态条目（原版 findstr dynamic/动态）
func arpHasDynamic(ip string) bool {
	cmd := exec.Command("arp", "-a", ip)
	hideCmd(cmd)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	text := strings.ToLower(decodeGBK(out))
	return strings.Contains(text, "dynamic") || strings.Contains(text, "动态")
}

type arpEntry struct {
	IP  string
	MAC string
}

// arp -a 全表动态条目
func arpTable() []arpEntry {
	cmd := exec.Command("arp", "-a")
	hideCmd(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	text := decodeGBK(out)
	var entries []arpEntry
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		ip := fields[0]
		mac := fields[1]
		if !isIPv4Format(ip) || !isMACFormat(mac) {
			continue
		}
		typ := strings.ToLower(fields[2])
		if typ != "dynamic" && typ != "动态" {
			continue
		}
		entries = append(entries, arpEntry{ip, mac})
	}
	return entries
}

func isMACFormat(s string) bool {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == ':' })
	if len(parts) != 6 {
		return false
	}
	for _, p := range parts {
		if len(p) != 2 {
			return false
		}
		for _, c := range p {
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
				return false
			}
		}
	}
	return true
}

// ---------- 外网连通 / 公网 IP ----------
func publicIP() string {
	sources := []string{
		"https://ip.3322.net",
		"https://ip.sb",
		"https://api.ipify.org",
		"http://ip.3322.net",
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// 强制 IPv4（原版 curl -4）
			return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "tcp4", addr)
		},
	}
	client := &http.Client{Timeout: 2 * time.Second, Transport: transport}
	for _, src := range sources {
		resp, err := client.Get(src)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		resp.Body.Close()
		ip := strings.TrimSpace(string(body))
		if isIPv4Format(ip) {
			return ip
		}
	}
	return ""
}

// ---------- 局域网发现设备 ----------
func lanDevices(localIP, gateway, gwseg string) []string {
	// 先 ping 网段广播，填充邻居表（原版行为）
	if gwseg != "" {
		pingOK(gwseg+".255", 1, 120)
	}
	entries := arpTable()
	var lines []string
	count := 0
	for _, e := range entries {
		if !isLANIP(e.IP) {
			continue
		}
		last := lastOctet(e.IP)
		if last == "255" || last == "0" {
			continue
		}
		if strings.EqualFold(e.IP, localIP) || strings.EqualFold(e.IP, gateway) {
			continue
		}
		status := "ARP 缓存未应答"
		if pingOK(e.IP, 2, 300) {
			status = "在线应答"
		}
		lines = append(lines, fmt.Sprintf("%-15s %-14s %-4s %-18s %s", e.IP, e.MAC, "-", status, "ARP"))
		count++
	}
	if count == 0 {
		return []string{"    - 无"}
	}
	return append([]string{
		fmt.Sprintf("%-15s %-14s %-4s %-18s %s", "IP", "MAC", "厂商", "状态", "来源"),
		fmt.Sprintf("%-15s %-14s %-4s %-18s %s", "----------", "--------------", "----", "-----------------", "--------"),
	}, lines...)
}

// ---------- 目标相机状态 ----------
func cameraStatus(gwseg, localIP string) []string {
	if gwseg == "" {
		return []string{"    - 未探测（未取到网段）"}
	}
	var lines []string
	for _, last := range []string{"100", "101", "108", "64"} {
		cip := gwseg + "." + last
		state := "ICMP 无响应，需厂商工具二次确认"
		if pingOK(cip, 1, 400) {
			state = "在线（ping 通信正常）"
		}
		if !strings.EqualFold(cip, localIP) && arpHasDynamic(cip) {
			state = "在线（二层有应答，设备可能禁用了ping）"
		}
		lines = append(lines, "    - "+cip+"  -- "+state)
	}
	lines = append(lines, "    注：多数工业/监控相机默认禁 ping，最终以厂商 IP 搜索工具或 ONVIF 探测为准。")
	return lines
}

// ---------- 空闲 IP 扫描（171~254，ping+ARP 双验证，找满 5 个即停） ----------
func freeIPs(baseIP, localIP, gateway string, prefixLen int) []string {
	if !isIPv4Format(baseIP) {
		return []string{"    - 未扫描到可用 IP"}
	}
	parts := strings.Split(baseIP, ".")
	seg := parts[0] + "." + parts[1] + "." + parts[2]
	var lines []string
	if prefixLen != 24 {
		lines = append(lines, "    （当前网络非 /24，以下按前三段推测，请现场复核）")
	}

	type probeResult struct {
		ip   string
		free bool
	}
	var (
		mu      sync.Mutex
		results []probeResult
		wg      sync.WaitGroup
	)
	sem := make(chan struct{}, 12)
	for n := 171; n <= 254; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			tip := fmt.Sprintf("%s.%d", seg, n)
			busy := strings.EqualFold(tip, localIP) || strings.EqualFold(tip, gateway)
			if !busy && pingOK(tip, 1, 400) {
				busy = true
			}
			if !busy && arpHasDynamic(tip) {
				busy = true
			}
			mu.Lock()
			results = append(results, probeResult{tip, !busy})
			mu.Unlock()
		}(n)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		a, _ := strconv.Atoi(lastOctet(results[i].ip))
		b, _ := strconv.Atoi(lastOctet(results[j].ip))
		return a < b
	})
	found := 0
	for _, r := range results {
		if !r.free {
			continue
		}
		lines = append(lines, "    - "+r.ip)
		found++
		if found >= 5 {
			break
		}
	}
	if found == 0 {
		return []string{"    - 未扫描到可用 IP"}
	}
	return lines
}
