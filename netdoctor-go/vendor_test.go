// MAC 厂商识别测试
package main

import (
	"strings"
	"testing"
)

func TestLookupVendor(t *testing.T) {
	cases := []struct {
		mac   string
		want  string
		check func(got string) bool
	}{
		// 真实 OUI：7C:C2:94 = 小米（Wireshark manuf 内嵌数据）
		{"7c-c2-94-43-45-24", "", func(g string) bool { return strings.Contains(g, "Xiaomi") }},
		{"7C:C2:94:43:45:24", "", func(g string) bool { return strings.Contains(g, "Xiaomi") }},
		// 随机/本地管理 MAC（U/L 位 = 1）：3A、B6 开头
		{"3a-61-af-11-22-33", "随机 MAC", func(g string) bool { return g == "随机 MAC" }},
		{"b6-f0-a4-44-55-66", "随机 MAC", func(g string) bool { return g == "随机 MAC" }},
		// 保留前缀 00:00:00 → Xerox（真实命中）
		{"00:00:00:00:00:01", "", func(g string) bool { return strings.Contains(g, "Xerox") }},
		// 非法格式
		{"not-a-mac", "-", func(g string) bool { return g == "-" }},
	}
	for _, c := range cases {
		got := lookupVendor(c.mac)
		if !c.check(got) {
			t.Errorf("lookupVendor(%q) = %q, 未通过校验", c.mac, got)
		}
	}
}

func TestIsRandomMAC(t *testing.T) {
	if !isRandomMAC("3A:61:AF:11:22:33") {
		t.Error("3A 开头应为随机/本地管理 MAC")
	}
	if !isRandomMAC("B6:F0:A4:11:22:33") {
		t.Error("B6 开头应为随机/本地管理 MAC")
	}
	if isRandomMAC("7C:C2:94:11:22:33") {
		t.Error("7C:C2:94 是注册 OUI，不应判为随机 MAC")
	}
}
