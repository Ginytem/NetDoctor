# NetDoctor Go 版（网络环境检测工具 · 图形界面）

原版为纯 BAT+VBScript + IExpress 打包（[Ginytem/NetDoctor](https://github.com/Ginytem/NetDoctor)，现行逻辑在 `pack/net_diag.bat`）。
本目录为 **Go 1.20 重写版**：单文件 EXE、**免安装双击即用、零运行时依赖**（原生 Win32 GUI，不依赖 WebView2/.NET/VC 运行库），可覆盖 **Win7 SP1 ~ Win11**（x86 与 x64 各一个），并有 garble 混淆加固版。

## 快速使用

```
NetDoctor-x64.exe         # 64 位系统运行（Windows 10/11）
NetDoctor-x86.exe         # 32 位系统运行（含 Win7 32 位）
NetDoctor-x64-garble.exe  # 混淆加固版（-literals 字符串混淆 + 符号剥离）
```

### 验证码（防盗用，仅本地构建启用）

**本仓库默认是"公开版"**：`totp.go` 中密钥为空占位，构建出的程序**不弹验证码**，任何用户双击直用。

**启用验证码（本地/交付给客户版）**：

1. 在 `netdoctor-go/` 目录新建 `totp_local.go`（已被 `.gitignore` 排除，**绝不推送到公开仓库**）：

```go
package main

func init() {
	// 此处填写真实密钥（Base32，32 字符）的 XOR(0x7D)+Base64 混淆串，见下方生成方法
	totpSecretEnc = "<你的混淆串>"
}
```

2. 重新构建（构建时自动包含该文件，程序内密钥为 XOR+Base64 混淆存储，**无任何查看入口**）：

```powershell
go build -ldflags "-s -w -H=windowsgui" -o NetDoctor-x64.exe .
```

3. 手机验证器录入密钥（一次性）：

- 密钥：真实密钥只保存在本地 `totp_local.go`（**不写入任何公开文档**；公开仓库 README 使用占位符，防止验证码被克隆者直接破解）
- otpauth 导入：`otpauth://totp/NetDoctor?secret=<你的Base32密钥>&issuer=NetDoctor&period=30&digits=6`（把 `<你的Base32密钥>` 换成真实密钥即可，可生成二维码扫码录入）

4. 客户打开程序 → 输入手机验证器上的 **6 位动态码**（30 秒刷新，±1 窗口容忍，输错最多 3 次）→ 进入主界面。

> **密钥来源（两级）**：程序启动时先自动拉取 `https://tools.ginytem.com/ndkey` 上的密钥；
> 拉取成功（内容校验合法）则以远程密钥为准，服务器可**随时更换/吊销密钥而无需重发 exe**；
> 拉取失败/断网/内容非法时**自动回退内置密钥**（上述本地构建的混淆串），离线照常可用。
> 公开版（无内置密钥）不联网、不拉取、直进主界面。

> **防盗用原理**：内置密钥校验动态码，密钥在二进制内为 **XOR+Base64 混淆存储**（strings 直接提取不到明文），
> 并可用 garble `-literals` 进一步混淆。程序被拷走给他人，没有你的手机验证器就无法生成正确动态码。
> 注意：密钥一旦泄露，改 `totp_local.go` 中的混淆串（用相同 XOR 重新编码）后重新编译更换。

### 服务器端部署密钥（远程拉取，可选）

在 `tools.ginytem.com` 的 web 根目录放一个**静态文件 `ndkey`**（无扩展名），内容为混淆串。程序启动时 GET `https://tools.ginytem.com/ndkey` 拉取。

1. 生成混淆串（PowerShell，`<密钥>` 换成 Base32 密钥，**建议 32 字符**，RFC 6238 推荐 160 bit）：

```powershell
$k = "<密钥>"; $bytes = [Text.Encoding]::UTF8.GetBytes($k) | ForEach-Object { $_ -bxor 0x7D }; [Convert]::ToBase64String([byte[]]$bytes)
```

2. 把输出写入服务器文件 `ndkey`（一行即可）。
3. 验证：浏览器打开 `https://tools.ginytem.com/ndkey` 应显示该串。
4. **轮换/吊销**：换密钥 → 重新生成混淆串 → 覆盖 `ndkey` 文件 → 客户下次启动自动用新密钥；已分发 exe 无法再通过验证（内置兜底需重新编译才能彻底替换）。

> 安全说明：ndkey 返回的是混淆串（非明文密钥），抓包/下载得到的是与 exe 内同级别的密文；远程密钥的真正价值是**服务器可随时更换**，让已分发出去的 exe 可控失效。

## 图形界面

默认 **1024×800**（最小 900×640，可缩放拉伸）：

- **顶部工具栏**：软件名、导出格式选择（HTML/JSON/TXT）、导出报告、重新检测
- **检测结果列表（上，占 8）**：23 项检测**逐项实时出现**，5 种状态色统一（PASS 绿 ✓ / WARN 橙 ▲ / FAIL 红 ✗ / RUNNING 蓝 ● / PENDING 灰 ○），占主要高度可滚动
- **详情与建议（下，占 2）**：选中检测项显示原始信息与修复建议，位于列表下方（紧凑高度）
- **底部状态栏**：当前阶段提示、进度条（已完成/总数）、总耗时

**交互**：
- 启动弹出**「权限验证」**对话框（提示"请输入验证码"），输入验证码**直接回车即可验证**（无需鼠标点按钮；ESC 取消）
- **检测完成后自动弹窗**询问是否导出报告：回车（默认"是"）→ 按当前所选格式直接导出到桌面并提示路径；ESC（"否"）→ 关闭弹窗
- 工具栏「导出报告」按钮仍可随时按所选格式（含保存对话框选路径）导出

> 说明：原设计中左侧分类导航与"验证码配置"查看入口已按需求移除；详情区已从右侧移到检测结果下方并按 8:2 分配高度。

## 报告导出

工具栏选择格式后点「导出报告」，默认桌面、可改路径，导出成功后弹窗提示路径：

- **HTML**：正式文档排版——标题 + 生成时间 + PASS/WARN/FAIL 统计卡片、按分类分节的表格、局域网设备表（MAC/厂商/来源）、**目标相机状态明细表（逐台列出）**、**建议分配 IP 列表**、状态列颜色标识、支持 `@media print` 打印，双击可直接打开；说明列仅呈现**对外建议文案**，不包含指纹/tracert/探测源等检测判断细节
- **JSON**：结构化数据（tool / generated_at / summary / categories / lan_devices，每项含 detail + advice），方便程序二次处理
- **TXT**：GBK+CRLF 原版格式（老记事本兼容）

## 与原 BAT 版的主要差异

| 项目 | 原版（BAT+VBScript） | Go 版 |
|---|---|---|
| 载体 | IExpress 解包 + 多个临时文件 | 单文件 EXE，零依赖 |
| 界面 | 黑框 cmd | 原生 Win32 图形界面（walk） |
| 防盗用 | 自毁删除解包目录（实测失效） | TOTP 动态验证码（密钥加密存储） |
| 系统适配 | MODERN/LEGACY 双分支 | 统一 `GetAdaptersAddresses`，Win7+ 均可 |
| 本机 IP/网关 | 多条命令拼凑 | 直接读网卡 API（含网关标志修正） |
| ping/arp | 反复拉起外部命令（弹黑框） | 原生 `IcmpSendEcho` API，零弹窗、免管理员 |
| 报告编码 | ANSI | TXT=GBK+CRLF；HTML/JSON=UTF-8 |
| 局域网扫描 | 串行，数十秒 | 并发（信号量 12）+ ICMP API，全流程约 13 秒 |
| 已知缺陷 | 如“此时不应有 False。”等 | 已规避 |

## 检测内容（23 项）

- **硬件配置**：操作系统（名称/架构/Build）、CPU、运存、C 盘剩余、显卡
- **运行库依赖**：VC++ 2015-2022（x64/x86）、.NET Framework、WebView2 Runtime
- **网络结构**：活动网卡、上网方式（有线/Wi-Fi）、本机 IP、子网掩码、默认网关、网关类型
  （对齐上游最新判定：路由器指纹组 × 光猫指纹组组合矩阵 + tracert 第 2 跳层级，输出"路由器连接/运营商网关 光猫类/路由光猫一体/疑似二级路由/疑似一级网关"五类）
- **IP 分配**：DHCP/静态、DHCP 服务器
- **外网连通**：出口 IPv4（多源探测）、IP 连通、DNS 解析
- **局域网**：局域网设备表（ping+ARP，**MAC 厂商识别**：内嵌 gomanuf/Wireshark OUI 数据库 47798 条前缀，`go:embed` 打包零外部依赖；**随机/本地管理 MAC 自动识别为"随机 MAC"**，如手机/电脑隐私随机地址，不误判厂商）、目标相机状态（.100/.101/.108/.64）、建议分配 IP（扫描 .171~.254 空闲段）

## 构建方法

工具链：**Go 1.20.14**（官方支持 Win7/8/Server2008/2012 的最后一个版本，Go 1.21+ 最低要求 Win10/Server2016）。

```powershell
# 首次：生成 Common Controls v6 manifest 资源（walk 必需，否则 TTM_ADDTOOL failed）
C:\Users\admin\go\bin\rsrc.exe -manifest app.manifest -o rsrc.syso

$env:CGO_ENABLED = "0"
$env:GOPROXY = "https://goproxy.cn,direct"   # 国内网络拉依赖

# 普通版（剥离符号 + GUI 子系统，无控制台黑框）
$env:GOARCH = "386";   go build -ldflags "-s -w -H=windowsgui" -o NetDoctor-x86.exe .
$env:GOARCH = "amd64"; go build -ldflags "-s -w -H=windowsgui" -o NetDoctor-x64.exe .

# garble 混淆加固版（需 Go 1.20 兼容的 garble v0.9.x）
garble -literals build -ldflags "-s -w -H=windowsgui" -o NetDoctor-x64-garble.exe .
```

依赖：`github.com/lxn/walk`（原生 GUI）、`golang.org/x/sys`、`golang.org/x/text`（v0.13.0）。

单元测试（RFC 6238 官方 TOTP 测试向量 + 密钥解密自检）：

```powershell
go test -v ./...
```

## 源码文件

| 文件 | 职责 |
|---|---|
| `main.go` | 入口：密钥已配置时弹权限验证对话框，否则直进主窗口 |
| `gui.go` | 主界面（工具栏/结果列表/下方详情/状态栏）、权限验证对话框、导出 |
| `items.go` | 检测项模型（CheckItem）、5 种状态常量、6 个分类 |
| `totp.go` | TOTP 动态验证码（RFC 6238）、密钥混淆存储与还原、**远程密钥拉取（失败回退内置）**（公开版为空占位） |
| `totp_local.go` | **本地专用**：真实密钥混淆串（已 gitignore，不入库，启用验证码） |
| `icmp.go` | 原生 ICMP ping（`IcmpSendEcho`，零弹窗零权限） |
| `sysinfo.go` | 操作系统 / CPU / 内存 / 磁盘 / 显卡 |
| `runtimes.go` | VC++ / .NET / WebView2 注册表检测 |
| `network.go` | GetAdaptersAddresses 读网卡、IP、网关、DHCP |
| `gateway.go` | 网关类型指纹判定 + tracert 第 2 跳 |
| `scan.go` | 公网 IP、ping、局域网设备、相机、空闲 IP |
| `devices.go` | 局域网设备结构化表格模型 |
| `vendor.go` | MAC 厂商识别（内嵌 OUI 数据库）+ 随机/本地管理 MAC 检测 |
| `manuf.txt` | OUI 厂商前缀数据库（gomanuf/Wireshark，47798 条，`go:embed` 内嵌） |
| `report.go` | TXT 报告排版与 GBK 写出 |
| `export.go` | HTML / JSON 报告生成 |
| `totp_test.go` | TOTP 单元测试 |
| `vendor_test.go` | 厂商识别 / 随机 MAC 单元测试 |

## 注意事项

- 检测为纯读操作（读网卡 API / 注册表 / ping / ARP / tracert / HTTP 探测），不做任何系统修改
- ping 走 Windows 原生 ICMP API：不需要管理员权限，**全程不弹 cmd 黑框**（唯一外部命令 tracert 已隐藏窗口）
- 公网 IP 探测使用 ip.3322.net / ip.sb / api.ipify.org，需联网
- 空闲 IP 扫描仅在本机网段后半段（如 .171~.254）进行，最多给 5 个建议
- 相机默认禁 ping 属正常现象，详情会提示用厂商工具二次确认
