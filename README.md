# NetDoctor - Windows 网络环境检测工具

一键检测 Windows 系统配置、网络拓扑、运行库依赖，自动生成诊断报告。适用于现场技术支持快速排查客户机环境问题。

## 版本说明

| 版本 | 形态 | 支持系统 | 状态 |
|---|---|---|---|
| **Go 版（netdoctor-go）** | 原生 Win32 GUI 单 EXE，免安装、零运行库依赖 | **Windows 7 SP1 及以上**（32/64 位） | **现行推荐版本** |
| 旧版（BAT） | BAT + VBScript 源码（附 IExpress 打包配置，exe 需自行打包） | Windows XP 及以上 | 保留，供 XP 等老设备使用 |

> **兼容性说明**：Go 版基于 Go 1.20 编译，官方支持下限为 Windows 7 SP1，**不支持 Windows XP**；旧版为纯 BAT/VBScript（零 PowerShell 依赖，cscript 为 XP 自带组件），XP 及以上系统均可运行。

---

## Go 版（新版，netdoctor-go）

原生 Windows 图形界面，双击即用，无需安装、不依赖任何运行库。

### 系统要求
- Windows 7 SP1 / 8 / 8.1 / 10 / 11（32 位与 64 位均支持）
- 不支持 Windows XP

### 功能
- **逐项实时检测**：约 23 项检测流式逐条显示，五项状态（待检测/检测中/通过/警告/失败）实时刷新，单项失败不影响其他项继续
- **Ping 工具**：输入 IP 或域名即测，结果翻译成人话（通畅 / 延迟偏高 / 丢包 / 不通 / 解析失败），带延迟范围与 TTL 说明
- **局域网设备发现**：IP / MAC / 厂商识别（内嵌 OUI 数据库，含随机 MAC 检测）/ 状态表格
- **网关类型判定**：光猫直连 / 路由器二级级联 / 主路由拨号 / 独立企业网关
- **报告导出**：HTML / JSON / TXT 三种格式，导出后提示并可直接打开所在文件夹
- **检测覆盖**：硬件配置、运行库依赖（VC++ / .NET / WebView2）、网络结构、外网连通性、局域网设备、相机检测、IP 分配建议

### 使用
1. 双击 `NetDoctor-x64.exe`（或 x86 版）运行
2. 点击「开始检测」，结果逐项实时出现
3. 检测完成后可选择导出报告（HTML/JSON/TXT）

### 构建（可选）
```bat
rem 1. 生成图标/版本/manifest 资源（仅首次或改配置后需要；已提交的 .syso 可直接构建）
go-winres make -in winres\winres.json -out rsrc_windows_amd64.syso -arch amd64
go-winres make -in winres\winres.json -out rsrc_windows_386.syso -arch 386

rem 2. 编译（x64 / x86）
go build -ldflags "-s -w -H=windowsgui" -o NetDoctor-x64.exe .
set GOARCH=386 && go build -ldflags "-s -w -H=windowsgui" -o NetDoctor-x86.exe .
```

> 资源说明：仓库已包含 `winres/` 配置与生成好的 `.syso` 资源文件（程序图标、版本信息 1.0.0、manifest）。**为什么 `.syso` 直接入库**：① `go build` 不会自动生成资源文件，不入库则 clone 后直接构建会丢失图标/版本/manifest；② manifest 声明使用新版通用控件（comctl32 v6），**缺失时部分对话框控件会创建失败导致程序退出**（曾实际踩坑），入库可保证任何人构建出的 exe 行为一致。`.syso` 由 `go-winres make` 生成，如需修改资源请改 `winres/winres.json` 后重新生成。

### 更新记录
- **2026-10-04**：清理仓库根目录垃圾/构建产物（`$log`、`b`、旧版 `NetDoctor.exe`）；公开版 totp.go 占位精简（移除弱混淆示例代码）；旧版 BAT 改为源码+打包配置提供。
- **2026-09-29**：新增程序图标（地球+扫描雷达）、版本信息 1.0.0；构建资源改为 go-winres 管理（icon / version / manifest 一体化）。

---

## 旧版（BAT 版）

IExpress 打包成单文件 EXE，XP 及以上系统可用。

### 功能特性

#### 硬件配置检测
- 操作系统版本与架构（32/64位，Win11 自动识别）
- CPU 型号
- 运存（物理内存）
- C 盘剩余空间
- 显卡设备（核显+独显，过滤虚拟显示）

#### 运行库检测
- VC++ 2015-2022 (x64/x86)
- .NET Framework（4.5 ~ 4.8.1 全版本识别）
- WebView2 Runtime（自动识别版本号）

#### 网络结构分析
- 本机 IP / 子网掩码 / 默认网关（netsh + ipconfig 双保险获取）
- 活动网卡与上网方式（有线/Wi-Fi）
- **网关类型智能判定**：光猫直连 / 路由器二级级联 / 主路由拨号 / 独立企业网关（页面指纹 + 二层拓扑双验证）
- DHCP 状态与服务器
- 外网连通性与公网 IP（多源获取）
- 局域网发现设备（IP / MAC / 厂商 / 状态 / 来源 表格）
- 常用 IP 段相机检测
- 建议分配 IP（171~254 双验证）

### 双分支兼容

- **MODERN 分支**：Win8+/10/11/Server2012+，用 netsh 高精度接口
- **LEGACY 分支**：WinXP/Win7/Server2003-2008R2，用 route print + ipconfig 原生命令
- 自动按系统版本分流，老电脑新电脑都能跑

### 自毁机制

- 用 IExpress 打包成单文件 EXE
- 运行结束后后台自动删除 EXE 本体
- 通过进程 PID 追溯，不依赖 EXE 文件名，改名也能自毁
- 不弹额外黑窗口，报告打开后自动退出

### 技术栈

- 纯批处理（BAT），GBK/ANSI 编码
- VBScript 辅助（进度条动画、WMI 查询、自毁清理）
- 零依赖，不需要 PowerShell（老电脑 XP 也能跑）

### 使用方式

1. 用 `NetDoctor.SED`（IExpress 打包配置）将 `网络环境诊断_修复版.bat` 打包为 `NetDoctor.exe` 后运行
2. 等待进度条走完
3. 自动打开桌面生成的检测报告 txt 文件
4. 运行结束后 EXE 自动删除

### 报告输出

报告自动保存到桌面，文件名带时间戳（避免覆盖）：
```
网络环境检测报告_20260919230411.txt
```

报告内容：
- 客户端硬件配置（操作系统 / CPU / 运存 / C盘剩余 / 显卡）
- 必备运行库依赖（VC++ / .NET / WebView2）
- 网络结构分析（网卡 / 上网方式 / IP / 掩码 / 网关 / 网关类型）
- IP 分配与建议（DHCP / 静态）
- 外网连通状态（公网 IP / 连通性 / DNS）
- 局域网发现设备（表格）
- 目标相机状态
- 建议分配 IP
