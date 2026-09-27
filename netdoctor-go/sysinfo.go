// 阶段 1：硬件配置检测（OS/架构/CPU/内存/C盘/显卡）
// 原版用 VBScript/WMI + reg query，本版改用 Windows API + 注册表，更快更稳
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// registry 包未导出的 WOW64 视图常量（标准值）
const (
	keyWow6464Key = 0x0100
	keyWow6432Key = 0x0200
)

type SysInfo struct {
	OSName   string
	OSArch   string
	OSType   string // MODERN / LEGACY（原版双分支标记，仅展示用）
	OSBuild  string
	CPUName  string
	RAMSize  string
	DiskFree string
	GPUName  string
}

func collectSysInfo() SysInfo {
	si := SysInfo{
		OSName:   "Windows",
		OSArch:   osArch(),
		OSType:   "MODERN",
		CPUName:  "通用处理器",
		RAMSize:  "未知",
		DiskFree: "未知",
		GPUName:  "标准显示适配器",
	}
	si.OSName, si.OSType, si.OSBuild = osInfo()
	if c := cpuName(); c != "" {
		si.CPUName = c
	}
	if r := ramMB(); r != "" {
		si.RAMSize = r
	}
	if d := diskFreeGB(); d != "" {
		si.DiskFree = d
	}
	if g := gpuNames(); g != "" {
		si.GPUName = g
	}
	return si
}

// 架构判定（与原版环境变量逻辑一致）
func osArch() string {
	if os.Getenv("PROCESSOR_ARCHITEW6432") != "" {
		return "64位"
	}
	switch os.Getenv("PROCESSOR_ARCHITECTURE") {
	case "AMD64":
		return "64位"
	case "ARM64":
		return "64位 (ARM)"
	}
	return "32位"
}

// 操作系统名称/分支/构建号
func osInfo() (name, typ, build string) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		registry.QUERY_VALUE|keyWow6464Key)
	if err != nil {
		return "Windows", "MODERN", ""
	}
	defer k.Close()

	name, _, _ = k.GetStringValue("ProductName")
	if name == "" {
		name = "Windows"
	}

	if _, _, err := k.GetIntegerValue("CurrentMajorVersionNumber"); err == nil {
		typ = "MODERN"
		if curBuild, _, _ := k.GetStringValue("CurrentBuild"); curBuild != "" {
			build = "(Build " + curBuild + ")"
		}
		// Win11 判定：Build >= 22000 且非 Server（注册表 ProductName 常仍写 Windows 10）
		curBuildNum, _, _ := k.GetStringValue("CurrentBuildNumber")
		cb, _ := strconv.Atoi(curBuildNum)
		instType, _, _ := k.GetStringValue("InstallationType")
		if !strings.EqualFold(instType, "Server") && cb >= 22000 {
			if dispVer, _, _ := k.GetStringValue("DisplayVersion"); dispVer != "" {
				name = "Windows 11 " + dispVer
			} else {
				name = "Windows 11"
			}
		}
	} else {
		typ = "LEGACY"
		if csd, _, _ := k.GetStringValue("CSDVersion"); csd != "" {
			build = "(" + csd + ")"
		}
	}
	return name, typ, build
}

func cpuName() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor\0`,
		registry.QUERY_VALUE|keyWow6464Key)
	if err != nil {
		return ""
	}
	defer k.Close()
	name, _, _ := k.GetStringValue("ProcessorNameString")
	return strings.TrimSpace(name)
}

// memoryStatusEx 对应 kernel32.GlobalMemoryStatusEx 的输入输出结构
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

// 物理内存总量（与原版 MB 取整一致）
func ramMB() string {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	proc := kernel32.NewProc("GlobalMemoryStatusEx")
	var ms memoryStatusEx
	ms.Length = uint32(unsafe.Sizeof(ms))
	r1, _, _ := proc.Call(uintptr(unsafe.Pointer(&ms)))
	if r1 == 0 {
		return ""
	}
	return fmt.Sprintf("%d MB", ms.TotalPhys/1048576)
}

// C 盘可用空间（保留 1 位小数）
func diskFreeGB() string {
	var free, total, totalFree uint64
	pathp, err := windows.UTF16PtrFromString(`C:\`)
	if err != nil {
		return ""
	}
	if err := windows.GetDiskFreeSpaceEx(pathp, &free, &total, &totalFree); err != nil {
		return ""
	}
	return fmt.Sprintf("%.1f GB 剩余", float64(totalFree)/1073741824)
}

// 显卡设备：注册表类键遍历 DriverDesc，核显+独显，过滤虚拟显示
func gpuNames() string {
	classKey := `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, classKey,
		registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE|keyWow6464Key)
	if err != nil {
		return ""
	}
	defer k.Close()
	subs, err := k.ReadSubKeyNames(0)
	if err != nil {
		return ""
	}
	excluded := []string{"Remote", "Basic", "Virtual", "Oray", "ToDesk", "Parsec", "Mirror"}
	var names []string
	for _, sub := range subs {
		sk, err := registry.OpenKey(k, sub, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		desc, _, err := sk.GetStringValue("DriverDesc")
		sk.Close()
		if err != nil || desc == "" {
			continue
		}
		low := strings.ToLower(desc)
		skip := false
		for _, ex := range excluded {
			if strings.Contains(low, strings.ToLower(ex)) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		dup := false
		for _, n := range names { // 原版 findstr 全串匹配去重（大小写敏感）
			if strings.Contains(n, desc) {
				dup = true
				break
			}
		}
		if !dup {
			names = append(names, desc)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, " / ")
}
