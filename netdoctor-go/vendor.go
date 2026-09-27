// MAC 厂商识别（OUI）
// 数据来源：gomanuf（github.com/timest/gomanuf，源自 Wireshark manuf 数据库，47798 条前缀）
// 通过 go:embed 内嵌，保持单文件 EXE 零外部依赖；运行时不再读取任何外部数据文件。
// 同时检测随机化/本地管理 MAC（IEEE U/L 位 = 1），此类地址无法也不应识别厂商。
package main

import (
	_ "embed"
	"strings"
)

//go:embed manuf.txt
var manufData string

// macKey 前缀键：掩码后的 MAC 值 + 前缀长度
type macKey struct {
	masked uint64
	cidr   uint8
}

var ouiMap map[macKey]string

// init 解析内嵌 manuf 数据（格式：前缀\t短名\t厂商全名，前缀可带 /28 /36 掩码）
func init() {
	ouiMap = make(map[macKey]string)
	for _, line := range strings.Split(manufData, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			continue
		}
		prefix := fields[0]
		org := fields[2]
		cidr := uint8(24)
		if i := strings.IndexByte(prefix, '/'); i >= 0 {
			n, err := atoui8(prefix[i+1:])
			if err != nil {
				continue
			}
			cidr = n
			prefix = prefix[:i]
		}
		if len(prefix) == 0 || len(prefix) > 17 {
			continue
		}
		v, ok := macPrefixToUint64(prefix)
		if !ok {
			continue
		}
		ouiMap[macKey{maskMac(v, cidr), cidr}] = org
	}
}

// atoui8 简易 uint8 解析
func atoui8(s string) (uint8, error) {
	var n uint16
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errNotNumber
		}
		n = n*10 + uint16(c-'0')
		if n > 255 {
			return 0, errNotNumber
		}
	}
	return uint8(n), nil
}

var errNotNumber = errStr("not a number")

type errStr string

func (e errStr) Error() string { return string(e) }

// macPrefixToUint64 解析前缀（如 "00:00:0A" 或 "00:00:0A:00:00:00"）为 uint64
func macPrefixToUint64(s string) (uint64, bool) {
	parts := strings.Split(s, ":")
	var v uint64
	for i, p := range parts {
		if len(p) != 2 {
			return 0, false
		}
		b, ok := hexByte(p)
		if !ok {
			return 0, false
		}
		if i >= 6 {
			return 0, false
		}
		v |= uint64(b) << uint8((6-i-1)*8)
	}
	return v, true
}

// hexByte 解析两位十六进制
func hexByte(s string) (byte, bool) {
	var b byte
	for i := 0; i < 2; i++ {
		c := s[i]
		var v byte
		switch {
		case c >= '0' && c <= '9':
			v = c - '0'
		case c >= 'a' && c <= 'f':
			v = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			v = c - 'A' + 10
		default:
			return 0, false
		}
		b = b<<4 | v
	}
	return b, true
}

// maskMac 保留高 cidr 位（IPv4 式掩码思路：/24 = 保留高 24 位）
func maskMac(v uint64, cidr uint8) uint64 {
	if cidr >= 48 {
		return v
	}
	return (v >> (48 - cidr)) << (48 - cidr)
}

// isRandomMAC 判断随机化/本地管理 MAC：IEEE U/L 位（首字节第 2 位）= 1
// 典型来源：Windows/macOS/手机"随机硬件地址"、虚拟机、网卡本地管理地址
func isRandomMAC(mac string) bool {
	parts := strings.Split(strings.ToUpper(mac), ":")
	if len(parts) < 6 {
		return false
	}
	b, ok := hexByte(parts[0])
	if !ok {
		return false
	}
	return b&0x02 != 0
}

// lookupVendor 根据 MAC 返回厂商名：
//  - 随机/本地管理 MAC → "随机 MAC"（不识别厂商）
//  - 查表命中 → 厂商名（如 Xiaomi / TP-Link）
//  - 未命中 → "-"
func lookupVendor(mac string) string {
	norm := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(mac), "-", ":"))
	if len(norm) != 17 {
		return "-"
	}
	if isRandomMAC(norm) {
		return "随机 MAC"
	}
	v, ok := macPrefixToUint64(norm) // 完整 6 字节，供 /24 /28 /36 掩码匹配
	if !ok {
		return "-"
	}
	// 精确度从高到低：/36 → /28 → /24
	for _, cidr := range []uint8{36, 28, 24} {
		if org, ok := ouiMap[macKey{maskMac(v, cidr), cidr}]; ok {
			return org
		}
	}
	return "-"
}
