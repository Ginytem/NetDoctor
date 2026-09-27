// 阶段 3：网关类型智能判定（对齐上游 2026-09-27「重构网关类型判定逻辑」）
// 证据源 ① 网关 80 端口 HTTP 指纹（http → https → :8080 依次兜底）
//         ② tracert 第 2 跳（hop2）是否为局域网地址（二层拓扑验证）
// 判定矩阵：路由器指纹 / 光猫指纹 命中组合 × hop2 层级，五分支
package main

import (
	"crypto/tls"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

var (
	// 光猫/运营商网关指纹（上游 net_diag.bat 最新版：天翼网关 E8-C ChinaNet 中国移动 中国联通 中国电信 吉比特 沃宽 FiberHome EchoLife ZXHN HGU HG8）
	onuFingerprints = []string{
		"天翼网关", "E8-C", "ChinaNet", "中国移动", "中国联通", "中国电信", "吉比特", "沃宽",
		"FiberHome", "EchoLife", "ZXHN", "HGU", "HG8",
	}
	// 路由器指纹（上游最新版：TP-LINK MERCURY FAST 小米 华为 华硕 腾达 Tenda H3C ASUS 网件 领势 OpenWrt 路由器 Router MiWiFi 水星 迅捷）
	routerFingerprints = []string{
		"TP-LINK", "MERCURY", "FAST", "小米", "华为", "华硕", "腾达", "Tenda", "H3C", "ASUS",
		"网件", "领势", "OpenWrt", "路由器", "Router", "MiWiFi", "水星", "迅捷",
	}
)

func determineGatewayType(gateway string) string {
	if gateway == "" {
		return "未知网关设备"
	}
	body, ok := fetchGatewayPage(gateway)
	hop2 := getHop2()
	lanHop2 := isLANIP(hop2)

	if !ok {
		if lanHop2 {
			return "【疑似二级路由】（页面未抓取）"
		}
		return "【疑似一级网关】（页面未抓取）"
	}
	low := strings.ToLower(body)
	routerHit := containsAny(low, routerFingerprints)
	ontHit := containsAny(low, onuFingerprints)

	switch {
	case routerHit && !ontHit:
		if lanHop2 {
			return "【路由器连接】（二级级联，上层还有设备 " + hop2 + "）"
		}
		return "【路由器连接】（主拨号/光猫已桥接）"
	case ontHit && !routerHit:
		if lanHop2 {
			return "【运营商网关 光猫类】（二级级联，上层还有设备 " + hop2 + "）"
		}
		return "【运营商网关 光猫类】（一级直连拨号）"
	case routerHit && ontHit:
		return "【路由/光猫一体】（疑似运营商智能网关）"
	default:
		if lanHop2 {
			return "【疑似二级路由】（无指纹，第2跳内网）"
		}
		return "【疑似一级网关】（无指纹，第2跳公网/CGN）"
	}
}

// 抓取网关管理页面：http → https（忽略证书）→ :8080，任一响应体 >= 100 字节即成功
func fetchGatewayPage(gateway string) (string, bool) {
	urls := []string{
		"http://" + gateway,
		"https://" + gateway,
		"http://" + gateway + ":8080",
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Timeout: 4 * time.Second, Transport: transport}
	for _, u := range urls {
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
		resp.Body.Close()
		if len(body) >= 100 {
			return string(body), true
		}
	}
	return "", false
}

// tracert -d -h 2 -w 300 223.5.5.5，取第 2 跳 IPv4（原版 findstr "^ *2 " + tokens=8）
func getHop2() string {
	cmd := exec.Command("tracert", "-d", "-h", "2", "-w", "300", "223.5.5.5")
	hideCmd(cmd)
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return ""
	}
	text := decodeGBK(out)
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimLeft(line, " \t\r")
		if trimmed != "2" && !strings.HasPrefix(trimmed, "2 ") && !strings.HasPrefix(trimmed, "2\t") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 8 {
			ip := fields[7]
			if isIPv4Format(ip) {
				return ip
			}
		}
	}
	return ""
}

func containsAny(lowerText string, keys []string) bool {
	for _, k := range keys {
		if strings.Contains(lowerText, strings.ToLower(k)) {
			return true
		}
	}
	return false
}
