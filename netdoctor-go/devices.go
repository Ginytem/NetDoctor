// 局域网设备：结构化数据（GUI 表格用）+ 报告格式化行
package main

import (
	"fmt"
	"strings"
)

// LanDevice 结构化设备信息
type LanDevice struct {
	IP     string
	MAC    string
	Vendor string
	Status string
	Source string
}

// lanDevicesStructured 扫描并返回结构化设备列表
func lanDevicesStructured(localIP, gateway, gwseg string) []LanDevice {
	// 先 ping 网段广播，填充邻居表（原版行为）
	if gwseg != "" {
		pingOK(gwseg+".255", 1, 120)
	}
	entries := arpTable()
	var devs []LanDevice
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
		devs = append(devs, LanDevice{IP: e.IP, MAC: e.MAC, Vendor: lookupVendor(e.MAC), Status: status, Source: "ARP"})
	}
	return devs
}

// formatLanDevices 把结构化设备列表格式化为报告行（原版表格样式）
func formatLanDevices(devs []LanDevice) []string {
	if len(devs) == 0 {
		return []string{"    - 无"}
	}
	lines := []string{
		fmt.Sprintf("%-15s %-14s %-22s %-18s %s", "IP", "MAC", "厂商", "状态", "来源"),
		fmt.Sprintf("%-15s %-14s %-22s %-18s %s", "----------", "--------------", "----------------------", "-----------------", "--------"),
	}
	for _, d := range devs {
		lines = append(lines, fmt.Sprintf("%-15s %-14s %-22s %-18s %s", d.IP, d.MAC, d.Vendor, d.Status, d.Source))
	}
	return lines
}
