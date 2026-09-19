@echo off
chcp 936 >nul
setlocal enabledelayedexpansion
title NetDoctor
mode con cols=62 lines=7
color 0A
:: ============================================================
:: 阶段 0: 桌面路径定位
:: ============================================================
set "desktop_dir="
for /f "tokens=2*" %%a in ('reg query "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders" /v Desktop 2^>nul') do set "desktop_dir=%%b"
if defined desktop_dir (
    for /f "delims=" %%i in ('call echo !desktop_dir!') do set "desktop_dir=%%i"
)
if not defined desktop_dir set "desktop_dir=%USERPROFILE%\Desktop"
:: 时间戳（wmic 原生，格式固定，与区域无关）
set "ts="
for /f "tokens=1" %%t in ('wmic os get localdatetime 2^>nul ^| findstr /r "[0-9]"') do set "ts=%%t"
if not defined ts set "ts=!RANDOM!!RANDOM!"
set "ts=!ts:~0,14!"
if "!ts:~0,4!" gtr "2000" (
    set "nowstr=!ts:~0,4!-!ts:~4,2!-!ts:~6,2! !ts:~8,2!:!ts:~10,2!:!ts:~12,2!"
) else (
    set "nowstr=%date% %time%"
)
set "report_file=!desktop_dir!\网络环境检测报告_!ts!.txt"
cls
echo.
:: 生成进度条动画组件（cscript 原生，XP 可用）
> "%temp%\prog.vbs" echo On Error Resume Next
>> "%temp%\prog.vbs" echo Dim a, b, i, bar
>> "%temp%\prog.vbs" echo a = CInt(WScript.Arguments(0))
>> "%temp%\prog.vbs" echo If a > b Then a = b
>> "%temp%\prog.vbs" echo b = CInt(WScript.Arguments(1))
>> "%temp%\prog.vbs" echo For i = a To b
>> "%temp%\prog.vbs" echo     bar = "[" ^& String(CInt(i / 2), "#") ^& String(50 - CInt(i / 2), " ") ^& "] " ^& i ^& "%%"
>> "%temp%\prog.vbs" echo     WScript.StdOut.Write Chr(13) ^& bar
>> "%temp%\prog.vbs" echo     WScript.Sleep 60
>> "%temp%\prog.vbs" echo Next
>> "%temp%\prog.vbs" echo WScript.StdOut.Write Chr(13) ^& bar ^& " " ^& Chr(13)

echo    正在检测中···请耐心等待···
echo    ----------------------------------------------------------
set "p_last=0"
cscript //nologo "%temp%\prog.vbs" 0 8
set "p_last=8"
:: ============================================================
:: 阶段 1: 优先检测电脑配置（全原生命令，秒级完成）
:: ============================================================
:: 1. 架构位数
set "os_arch=32位"
if defined PROCESSOR_ARCHITEW6432 (
    set "os_arch=64位"
) else (
    if "%PROCESSOR_ARCHITECTURE%"=="AMD64" set "os_arch=64位"
    if "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "os_arch=64位 (ARM)"
)
:: 2. 操作系统与版本分轨判定
set "os_type=MODERN"
set "os_name=Windows"
for /f "tokens=2*" %%a in ('reg query "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion" /v ProductName 2^>nul') do set "os_name=%%b"
set "os_build="
reg query "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion" /v CurrentMajorVersionNumber >nul 2>nul
if !errorlevel! equ 0 (
    set "os_type=MODERN"
    for /f "tokens=2*" %%a in ('reg query "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion" /v CurrentBuild 2^>nul') do set "os_build=(Build %%b)"
    :: Win11 判定：Build >= 22000（注册表 ProductName 在 Win11 上仍可能写 Windows 10）
    set "cur_build=0"
    for /f "tokens=2*" %%a in ('reg query "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion" /v CurrentBuildNumber 2^>nul') do set "cur_build=%%b"
    set /a cb=!cur_build!
    :: Server 优先：InstallationType=Server 时保留 ProductName 完整名称
    set "inst_type="
    for /f "tokens=2*" %%a in ('reg query "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion" /v InstallationType 2^>nul') do set "inst_type=%%b"
    if /i not "!inst_type!"=="Server" if !cb! geq 22000 (
        set "disp_ver="
        for /f "tokens=2*" %%a in ('reg query "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion" /v DisplayVersion 2^>nul') do set "disp_ver=%%b"
        if defined disp_ver (set "os_name=Windows 11 !disp_ver!") else (set "os_name=Windows 11")
    )
) else (
    set "os_type=LEGACY"
    for /f "tokens=2*" %%a in ('reg query "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion" /v CSDVersion 2^>nul') do set "os_build=(%%b)"
)
:: 3. CPU
set "cpu_name=通用处理器"
for /f "tokens=2*" %%a in ('reg query "HKLM\HARDWARE\DESCRIPTION\System\CentralProcessor\0" /v ProcessorNameString 2^>nul') do set "cpu_name=%%b"
:: 4. 物理内存 + C 盘剩余（VBScript 精确换算，零 PowerShell；cscript 为 XP 自带系统组件）
set "ram_size=未知"
set "disk_free=未知"
> "%temp%\sysinfo.vbs" echo On Error Resume Next
>> "%temp%\sysinfo.vbs" echo Set w = GetObject("winmgmts:\\.\root\cimv2")
>> "%temp%\sysinfo.vbs" echo Set c1 = w.ExecQuery("SELECT TotalPhysicalMemory FROM Win32_ComputerSystem")
>> "%temp%\sysinfo.vbs" echo For Each o In c1
>> "%temp%\sysinfo.vbs" echo   WScript.Echo "MEM=" ^& Round(o.TotalPhysicalMemory / 1048576, 0)
>> "%temp%\sysinfo.vbs" echo Next
>> "%temp%\sysinfo.vbs" echo Set c2 = w.ExecQuery("SELECT FreeSpace FROM Win32_LogicalDisk WHERE DeviceID='C:'")
>> "%temp%\sysinfo.vbs" echo For Each o In c2
>> "%temp%\sysinfo.vbs" echo   WScript.Echo "DISK=" ^& Round(o.FreeSpace / 1073741824, 1)
>> "%temp%\sysinfo.vbs" echo Next
for /f "tokens=1,2 delims==" %%a in ('cscript //nologo "%temp%\sysinfo.vbs" 2^>nul') do (
    if "%%a"=="MEM" set "ram_size=%%b MB"
    if "%%a"=="DISK" set "disk_free=%%b GB 可用"
)
:: wmic 兜底（wmic 输出值可能带尾随 CR，先剥离再截断）
if "!ram_size!"=="未知" (
    for /f "tokens=2 delims==" %%a in ('wmic ComputerSystem get TotalPhysicalMemory /value 2^>nul ^| findstr /i "TotalPhysicalMemory"') do (
        set "ram=%%a"
        for /f "delims=" %%x in ("!ram!") do set "ram=%%x"
        if defined ram set "ram_size=!ram:~0,-6! MB（约）"
    )
)
if "!disk_free!"=="未知" (
    for /f "tokens=2 delims==" %%a in ('wmic LogicalDisk where "DeviceID='C:'" get FreeSpace /value 2^>nul ^| findstr /i "FreeSpace"') do (
        set "df=%%a"
        for /f "delims=" %%x in ("!df!") do set "df=%%x"
        if defined df set "disk_free=!df:~0,-9! GB 可用（约）"
    )
)
:: systeminfo 最终兜底（慢，仅在前两者都失败时）
if "!ram_size!"=="未知" (
    for /f "tokens=2 delims=:" %%a in ('systeminfo 2^>nul ^| findstr /i /c:"物理内存总量" /c:"Total Physical Memory"') do (
        set "val=%%a"
        set "ram_size=!val: =!"
    )
)
:: 5. 显卡设备遍历（核显+独显）
set "gpu_name="
for /f "tokens=*" %%k in ('reg query "HKLM\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}" /s /v DriverDesc 2^>nul ^| findstr /i "DriverDesc"') do (
    for /f "tokens=2*" %%x in ("%%k") do (
        set "cur_gpu=%%y"
        echo !cur_gpu! | findstr /i "Remote Basic Virtual Oray ToDesk Parsec Mirror" >nul
        if !errorlevel! equ 1 (
            if not defined gpu_name (
                set "gpu_name=!cur_gpu!"
            ) else (
                echo !gpu_name! | findstr /c:"!cur_gpu!" >nul
                if !errorlevel! equ 1 set "gpu_name=!gpu_name! / !cur_gpu!"
            )
        )
    )
)
if not defined gpu_name set "gpu_name=标准显示适配器"
:: 6. VC++ 运行库
set "vc_x64=【缺失】"
set "vc_x86=【缺失】"
reg query "HKLM\SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\x64" /v Installed 2>nul | findstr "0x1" >nul && set "vc_x64=【已安装】"
if /i "!os_arch!"=="32位" (
    reg query "HKLM\SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\x86" /v Installed 2>nul | findstr "0x1" >nul && set "vc_x86=【已安装】"
) else (
    reg query "HKLM\SOFTWARE\WOW6432Node\Microsoft\VisualStudio\14.0\VC\Runtimes\x86" /v Installed 2>nul | findstr "0x1" >nul && set "vc_x86=【已安装】"
)
:: 7. .NET Framework（完整阈值表）
set "net_version=【缺失/未安装 .NET 4.5+】"
reg query "HKLM\SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full" /v Release >nul 2>nul
if !errorlevel! equ 0 (
    for /f "tokens=3" %%a in ('reg query "HKLM\SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full" /v Release 2^>nul') do (
        set /a rel=%%a
        if !rel! geq 533320 set "net_version=【已安装】（.NET 4.8.1）"
        if !rel! geq 528040 if !rel! lss 533320 set "net_version=【已安装】（.NET 4.8）"
        if !rel! geq 461808 if !rel! lss 528040 set "net_version=【已安装】（.NET 4.7.2）"
        if !rel! geq 461308 if !rel! lss 461808 set "net_version=【已安装】（.NET 4.7.1）"
        if !rel! geq 460798 if !rel! lss 461308 set "net_version=【已安装】（.NET 4.7）"
        if !rel! geq 394802 if !rel! lss 460798 set "net_version=【已安装】（.NET 4.6.2）"
        if !rel! geq 394271 if !rel! lss 394802 set "net_version=【已安装】（.NET 4.6.1）"
        if !rel! geq 393295 if !rel! lss 394271 set "net_version=【已安装】（.NET 4.6）"
        if !rel! lss 393295 set "net_version=【已安装】（.NET 4.5.x）"
    )
)
:: 8. WebView2 Runtime
set "webview2=【缺失】"
reg query "HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" /v pv >nul 2>nul
if !errorlevel! equ 0 (
    for /f "tokens=2*" %%a in ('reg query "HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" /v pv 2^>nul') do set "wv_ver=%%b"
    if defined wv_ver set "webview2=【已安装】(!wv_ver!)"
) else (
    reg query "HKLM\SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" /v pv >nul 2>nul
    if !errorlevel! equ 0 (
        for /f "tokens=2*" %%a in ('reg query "HKLM\SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" /v pv 2^>nul') do set "wv_ver=%%b"
        if defined wv_ver set "webview2=【已安装】(!wv_ver!)"
    )
)
cls
echo.
echo    正在检测中···请耐心等待···
echo    ----------------------------------------------------------
cscript //nologo "%temp%\prog.vbs" !p_last! 30
set "p_last=30"
:: ============================================================
:: 阶段 2: 根据配置结果，动态跳转专属网络探测分支
:: ============================================================
if "!os_type!"=="MODERN" goto :NETWORK_MODERN
goto :NETWORK_LEGACY
:: ------------------------------------------------------------
:: 分支 A: 现代 Windows (Win8+/10/11/Server2012+) 专有方案
:: netsh interface ipv4 show addresses 一次拿全 IP/掩码/网关/DHCP
:: （默认路由接口 = 物理活动网卡，虚拟网卡无默认路由天然被过滤）
:: ------------------------------------------------------------
:NETWORK_MODERN
set "localip=127.0.0.1"
set "gateway="
set "mask=255.255.255.0"
set "prefixlen=24"
set "dhcpen="
set "cand_ip="
set "cand_pl="
set "sel="
for /f "tokens=1,* delims=:" %%a in ('netsh interface ipv4 show addresses 2^>nul') do (
    set "k=%%a"
    set "v=%%b"
    echo !k! | findstr /i /c:"for interface" /c:"的接口" >nul
    if !errorlevel! equ 0 if not defined sel (
        set "cand_ip="
        set "cand_pl="
        set "dhcpen="
    )
    echo !k! | findstr /i /c:"IP Address" /c:"IP 地址" >nul
    if !errorlevel! equ 0 if not defined sel (
        for /f "tokens=1" %%x in ("!v!") do set "cand_ip=%%x"
    )
    echo !k! | findstr /i /c:"Subnet Prefix" /c:"子网前缀" >nul
    if !errorlevel! equ 0 if not defined sel (
        for /f "tokens=2 delims=/" %%x in ("!v!") do (
            for /f "tokens=1" %%y in ("%%x") do set "cand_pl=%%y"
        )
    )
    echo !k! | findstr /i /c:"Default Gateway" /c:"默认网关" >nul
    if !errorlevel! equ 0 (
        for /f "tokens=1" %%x in ("!v!") do if not defined sel (
            if not "%%x"=="0.0.0.0" if not "%%x"=="on-link" (
                set "sel=1"
                set "localip=!cand_ip!"
                set "prefixlen=!cand_pl!"
                set "gateway=%%x"
            )
        )
    )
    echo !k! | findstr /i /c:"DHCP enabled" /c:"DHCP 已启用" >nul
    if !errorlevel! equ 0 if not defined sel (
        for /f "tokens=1" %%x in ("!v!") do set "dhcpen=%%x"
    )
)
:: 掩码换算
if defined prefixlen (
    if "!prefixlen!"=="8" set "mask=255.0.0.0"
    if "!prefixlen!"=="16" set "mask=255.255.0.0"
    if "!prefixlen!"=="24" set "mask=255.255.255.0"
    if "!prefixlen!"=="25" set "mask=255.255.255.128"
    if "!prefixlen!"=="26" set "mask=255.255.255.192"
    if "!prefixlen!"=="27" set "mask=255.255.255.224"
    if "!prefixlen!"=="28" set "mask=255.255.255.240"
    if "!prefixlen!"=="29" set "mask=255.255.255.248"
    if "!prefixlen!"=="30" set "mask=255.255.255.252"
    if "!prefixlen!"=="31" set "mask=255.255.255.254"
    if "!prefixlen!"=="32" set "mask=255.255.255.255"
)
:: 活动网卡名（State=Connected）
set "adapter="
for /f "skip=3 tokens=2,4" %%a in ('netsh interface show interface 2^>nul') do (
    if not defined adapter (
        if /i "%%a"=="connected" set "adapter=%%b"
        if /i "%%a"=="已连接" set "adapter=%%b"
    )
)
if not defined adapter set "adapter=以太网"
:: DHCP 判定
set "dhcp=手动指定（静态）"
if /i "!dhcpen!"=="Yes" set "dhcp=自动获取（DHCP）"
if /i "!dhcpen!"=="是" set "dhcp=自动获取（DHCP）"
:: 判定网络类型（匹配连接状态，未连接不误判）
set "nettype=有线连接（网线）"
netsh wlan show interfaces 2>nul | findstr /i /c:"已连接" /c:": connected" >nul
if !errorlevel! equ 0 set "nettype=Wi-Fi（无线网络）"
goto :NETWORK_COMMON
:: ------------------------------------------------------------
:: 分支 B: 老旧 Windows (XP/Win7/Win8/Server2003-2008R2) 原生命令兼容流
:: ------------------------------------------------------------
:NETWORK_LEGACY
set "localip=127.0.0.1"
set "gateway="
set "mask=255.255.255.0"
set "prefixlen=24"
:: route print 默认路由行一次取网关+本机IP（修掉 findstr $ 锚定被 CR 卡死的问题；排除 0.0.0.0）
for /f "tokens=1,2,3,4,5" %%a in ('route print 0.0.0.0 2^>nul ^| findstr /r /c:"0\.0\.0\.0 *0\.0\.0\.0"') do (
    if not defined gateway (
        echo %%c | findstr /r /c:"^[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*" >nul
        if !errorlevel! equ 0 (
            if /i not "%%c"=="0.0.0.0" (
                set "gateway=%%c"
                set "localip=%%d"
            )
        )
    )
)
:: 掩码精确解析（ipconfig /all 首个 IPv4 对应掩码）
set "mask2="
for /f "tokens=2 delims=:" %%a in ('ipconfig /all 2^>nul ^| findstr /i /c:"子网掩码" /c:"Subnet Mask"') do (
    for /f "tokens=1" %%b in ("%%a") do if not defined mask2 set "mask2=%%b"
)
if defined mask2 (
    set "mask=!mask2!"
    set "pl=0"
    set "m2=!mask2:.= !"
    for %%b in (!m2!) do (
        set "v=%%b"
        if !v! geq 128 (set /a v-=128 & set /a pl+=1)
        if !v! geq 64 (set /a v-=64 & set /a pl+=1)
        if !v! geq 32 (set /a v-=32 & set /a pl+=1)
        if !v! geq 16 (set /a v-=16 & set /a pl+=1)
        if !v! geq 8 (set /a v-=8 & set /a pl+=1)
        if !v! geq 4 (set /a v-=4 & set /a pl+=1)
        if !v! geq 2 (set /a v-=2 & set /a pl+=1)
        if !v! geq 1 (set /a pl+=1)
    )
    set "prefixlen=!pl!"
)
:: 本机 IP 兜底（route print 失败时）
if "!localip!"=="127.0.0.1" (
    for /f "tokens=2 delims=:" %%a in ('ipconfig 2^>nul ^| findstr /i "IPv4"') do (
        for /f "tokens=1" %%b in ("%%a") do if "!localip!"=="127.0.0.1" set "localip=%%b"
    )
)
:: 活动网卡名
set "adapter="
for /f "tokens=2* delims=:" %%a in ('ipconfig /all 2^>nul ^| findstr /i /c:"适配器" /c:"adapter"') do (
    for /f "tokens=*" %%b in ("%%a") do if not defined adapter set "adapter=%%b"
)
echo !adapter! | findstr /i /c:"Hyper-V" /c:"vEthernet" /c:"Virtual" /c:"Loopback" /c:"Bluetooth" >nul
if !errorlevel! equ 0 set "adapter=以太网"
if not defined adapter set "adapter=以太网"
:: DHCP 判定
set "dhcp=手动指定（静态）"
set "dhcpserver=无"
for /f "tokens=2 delims=:" %%a in ('ipconfig /all 2^>nul ^| findstr /i /c:"DHCP 服务器" /c:"DHCP Server"') do (
    for /f "tokens=1" %%b in ("%%a") do if "!dhcpserver!"=="无" (
        if /i not "%%b"=="0.0.0.0" if /i not "%%b"=="255.255.255.255" set "dhcpserver=%%b"
    )
)
if not "!dhcpserver!"=="无" set "dhcp=自动获取（DHCP）"
if not defined gateway if not "!dhcpserver!"=="无" set "gateway=!dhcpserver!"
:: 判定网络类型
set "nettype=有线连接（网线）"
netsh wlan show interfaces 2>nul | findstr /i /c:"已连接" /c:": connected" >nul
if !errorlevel! equ 0 set "nettype=Wi-Fi（无线网络）"
goto :NETWORK_COMMON
:: ------------------------------------------------------------
:: 阶段 3: 汇聚层（DHCP/拓扑嗅探/外网/相机/可用 IP 探测）
:: ------------------------------------------------------------
:NETWORK_COMMON
:: DHCP 服务器（MODERN 分支补取）
if "!os_type!"=="MODERN" (
    set "dhcpserver=无"
    for /f "tokens=2 delims=:" %%a in ('ipconfig /all 2^>nul ^| findstr /i /c:"DHCP 服务器" /c:"DHCP Server"') do (
        for /f "tokens=1" %%b in ("%%a") do if "!dhcpserver!"=="无" (
            if /i not "%%b"=="0.0.0.0" if /i not "%%b"=="255.255.255.255" set "dhcpserver=%%b"
        )
    )
)
if not defined gateway if not "!dhcpserver!"=="无" set "gateway=!dhcpserver!"
set "ipadvice=局域网采用 DHCP 动态分配，可直接选用后段空闲 IP"
if "!dhcp!"=="手动指定（静态）" set "ipadvice=局域网可能存在严格固定 IP 规则，建议向现场管理员报备"
:: 网关段
set "gwseg="
if defined gateway (
    for /f "tokens=1-3 delims=." %%a in ("!gateway!") do set "gwseg=%%a.%%b.%%c"
)
if not defined gwseg (
    for /f "tokens=1-3 delims=." %%a in ("!localip!") do set "gwseg=%%a.%%b.%%c"
)
cls
echo.
echo    正在检测中···请耐心等待···
echo    ----------------------------------------------------------
cscript //nologo "%temp%\prog.vbs" !p_last! 55
set "p_last=55"
:: 网关类型判定
set "gwtype=未知网关设备"
set "http_dump=%temp%\gw_http.txt"
if defined gateway (
    where curl >nul 2>nul
    if !errorlevel! equ 0 (
        curl -s -L -k -A "Mozilla/5.0" --max-time 4 "http://!gateway!" > "%http_dump%" 2>nul
        for %%z in ("%http_dump%") do if %%~zz lss 100 (
            curl -s -k -L -A "Mozilla/5.0" --max-time 4 "https://!gateway!" > "%http_dump%" 2>nul
        )
        for %%z in ("%http_dump%") do if %%~zz lss 100 (
            curl -s -L -A "Mozilla/5.0" --max-time 4 "http://!gateway!:8080" > "%http_dump%" 2>nul
        )
    )
    for %%z in ("%http_dump%") do if %%~zz lss 100 (
        > "%temp%\gw_fetch.vbs" echo On Error Resume Next
        >> "%temp%\gw_fetch.vbs" echo Set x = CreateObject("MSXML2.XMLHTTP"^)
        >> "%temp%\gw_fetch.vbs" echo x.open "GET", "http://!gateway!", False
        >> "%temp%\gw_fetch.vbs" echo x.setTimeouts 8000, 8000, 8000, 8000
        >> "%temp%\gw_fetch.vbs" echo x.send
        >> "%temp%\gw_fetch.vbs" echo If x.Status = 200 Then WScript.Echo x.responseText
        cscript //nologo "%temp%\gw_fetch.vbs" > "%http_dump%" 2>nul
        del "%temp%\gw_fetch.vbs" 2>nul
    )
    for %%z in ("%http_dump%") do if %%~zz lss 100 (
        > "%temp%\gw_fetch.vbs" echo On Error Resume Next
        >> "%temp%\gw_fetch.vbs" echo Set x = CreateObject("MSXML2.XMLHTTP"^)
        >> "%temp%\gw_fetch.vbs" echo x.open "GET", "http://!gateway!:8080", False
        >> "%temp%\gw_fetch.vbs" echo x.setTimeouts 8000, 8000, 8000, 8000
        >> "%temp%\gw_fetch.vbs" echo x.send
        >> "%temp%\gw_fetch.vbs" echo If x.Status = 200 Then WScript.Echo x.responseText
        cscript //nologo "%temp%\gw_fetch.vbs" > "%http_dump%" 2>nul
        del "%temp%\gw_fetch.vbs" 2>nul
    )
    set "gw_fetch_ok=0"
    if exist "%http_dump%" for %%z in ("%http_dump%") do if %%~zz geq 100 set "gw_fetch_ok=1"
    set "hop2="
    for /f "tokens=*" %%r in ('tracert -d -h 2 -w 300 223.5.5.5 2^>nul ^| findstr /r /c:"^ *2 "') do (
        for /f "tokens=8" %%h in ("%%r") do (
            echo %%h | findstr /r /c:"^[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*$" >nul
            if !errorlevel! equ 0 set "hop2=%%h"
        )
    )
    set "is_lan_hop2=0"
    if defined hop2 (
        echo !hop2! | findstr /r /c:"^192\.168\." /c:"^10\." >nul
        if !errorlevel! equ 0 set "is_lan_hop2=1"
        for /f "tokens=1,2 delims=." %%p in ("!hop2!") do (
            if "%%p"=="172" if %%q geq 16 if %%q leq 31 set "is_lan_hop2=1"
        )
    )
    if "!gw_fetch_ok!"=="0" (
        set "gwtype=【未识别】（网关页面未抓取到，指纹判定跳过）"
    ) else (
        findstr /i "天翼网关 E8-C ChinaNet 中国移动 吉比特 沃宽网关 FiberHome EchoLife ZXHN GPON EPON 光猫 光纤 HGU HG8 cgi-bin" "%http_dump%" >nul 2>nul
        if !errorlevel! equ 0 (
            set "gwtype=【确认为光猫直连】"
        ) else (
            findstr /i "TP-LINK MERCURY FAST ASUS MiWiFi 小米 华硕 腾达 Tenda H3C OpenWrt 路由器 Router" "%http_dump%" >nul 2>nul
            if !errorlevel! equ 0 (
                if "!is_lan_hop2!"=="1" (
                    set "gwtype=【路由器连接】（二级级联路由，上层还有设备 !hop2!）"
                ) else (
                    set "gwtype=【路由器连接】（路由器负责主拨号/光猫已设为桥接）"
                )
            ) else (
                if "!is_lan_hop2!"=="1" (
                    set "gwtype=【路由器连接】（二级级联网络，上级网关 !hop2!）"
                ) else (
                    set "gwtype=【独立主路由/企业网关】（已是一级网络，未暴露光猫特征）"
                )
            )
        )
    )
)
del "%http_dump%" 2>nul
:: 外网连通性
set "inet_ip=不通"
set "inet_dns=异常"
ping -n 1 -w 1200 223.5.5.5 >nul 2>nul && set "inet_ip=正常（223.5.5.5 可达）"
ping -n 1 -w 1500 www.baidu.com >nul 2>nul && set "inet_dns=正常（域名解析与外网访问正常）"
cls
echo.
echo    正在检测中···请耐心等待···
echo    ----------------------------------------------------------
cscript //nologo "%temp%\prog.vbs" !p_last! 80
set "p_last=80"
:: 局域网在线设备
if defined gwseg (
    ping -n 1 -w 120 !gwseg!.255 >nul 2>nul
)
> "%temp%\landev_list.txt" (
    arp -a > "%temp%\arp_raw.txt" 2>nul
    echo IP              MAC             厂商  状态              来源
    echo ----------       --------------  ----  -----------------  --------
    for /f "tokens=1,2" %%a in ('type "%temp%\arp_raw.txt" ^| findstr /i /c:"dynamic" /c:"动态" 2^>nul') do (
        set "islan=0"
        echo %%a | findstr /r /c:"^192\.168\." /c:"^10\." >nul
        if !errorlevel! equ 0 set "islan=1"
        for /f "tokens=1,2 delims=." %%p in ("%%a") do (
            if "%%p"=="172" if %%q geq 16 if %%q leq 31 set "islan=1"
        )
        if "!islan!"=="1" (
            for /f "tokens=4 delims=." %%o in ("%%a") do (
                if not "%%o"=="255" if not "%%o"=="0" (
                    if /i not "%%a"=="!localip!" if /i not "%%a"=="!gateway!" (
                        ping -n 2 -w 300 %%a >nul 2>nul
                        if !errorlevel! equ 0 (
                            echo %%a   %%b   -    在线应答          ARP
                        ) else (
                            echo %%a   %%b   -    ARP 缓存未应答    ARP
                        )
                    )
                )
            )
        )
    )
)
:: 外网 IP（curl 多源优先，VBS 兜底；VBS 不依赖 PowerShell，XP 自带）
set "pubip=无法获取（无 curl 且 cscript 不可用）"
set "pubip2="
where curl >nul 2>nul
if !errorlevel! equ 0 (
    for %%u in ("https://ip.3322.net" "https://ip.sb" "https://api.ipify.org") do (
        if not defined pubip2 for /f "usebackq tokens=1 delims= " %%a in (`curl -4 -s --max-time 2 %%~u 2^>nul`) do (
            echo %%a | findstr /r /c:"^[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*" >nul
            if !errorlevel! equ 0 set "pubip2=%%a"
        )
    )
)
if defined pubip2 (
    set "pubip=!pubip2!"
) else (
    > "%temp%\getip.vbs" echo On Error Resume Next
    >> "%temp%\getip.vbs" echo Set x = CreateObject("MSXML2.XMLHTTP"^)
    >> "%temp%\getip.vbs" echo x.open "GET", "http://ip.3322.net", False
    >> "%temp%\getip.vbs" echo x.setTimeouts 8000, 8000, 8000, 8000
    >> "%temp%\getip.vbs" echo x.send
    >> "%temp%\getip.vbs" echo If x.Status = 200 Then WScript.Echo x.responseText
    for /f "usebackq tokens=*" %%a in (`cscript //nologo "%temp%\getip.vbs" 2^>nul`) do (
        echo %%a | findstr /r /c:"^[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*" >nul
        if !errorlevel! equ 0 set "pubip=%%a"
    )
    del "%temp%\getip.vbs" 2>nul
)
:: 相机状态（按网关网段自动探测常见相机地址，不再硬编码）
set "camseg="
if defined gwseg set "camseg=!gwseg!"
> "%temp%\cam_list.txt" (
    if defined camseg (
        for %%n in (100 101 108 64) do (
            set "cip=!camseg!.%%n"
            set "cstate=ICMP 无响应，需厂商工具二次确认"
            ping -n 1 -w 400 !cip! >nul 2>nul
            if !errorlevel! equ 0 set "cstate=在线（ping 通信正常）"
            if /i not "!cip!"=="!localip!" (
                arp -a !cip! 2>nul | findstr /i /c:"dynamic" /c:"动态" >nul
                if !errorlevel! equ 0 set "cstate=在线（二层有应答，设备可能禁用了ping）"
            )
            echo    - !cip!  -- !cstate!
        )
        echo    注：多数工业/监控相机默认禁 ping，最终以厂商 IP 搜索工具或 ONVIF 探测为准。
    )
)
:: 空闲 IP 扫描（DHCP 池外 99-2 倒序，ping+ARP 双判，找满 5 个即停）
set "found=0"
set "baseip=!gateway!"
if not defined baseip set "baseip=!localip!"
> "%temp%\freeip_list.txt" (
    if defined baseip (
        for /f "tokens=1-3 delims=." %%a in ("!baseip!") do (
            set "seg=%%a.%%b.%%c"
            if not "!prefixlen!"=="24" echo    （当前网络非 /24，以下按前三段推测，请现场复核）
            for /l %%n in (171,1,254) do (
                if !found! lss 5 (
                    set "tip=!seg!.%%n"
                    set "busy=0"
                    if /i "!tip!"=="!localip!" set "busy=1"
                    if /i "!tip!"=="!gateway!" set "busy=1"
                    if "!busy!"=="0" (
                        ping -n 1 -w 400 !tip! >nul 2>nul
                        arp -a !tip! 2>nul | findstr /i /c:"dynamic" /c:"动态" >nul
                        if !errorlevel! equ 0 set "busy=1"
                    )
                    if "!busy!"=="0" (
                        echo    - !tip!
                        set /a found+=1
                    )
                )
            )
        )
    )
)
:: ============================================================
:: 阶段 4: 输出标准化诊断报告（带时间戳，不覆盖旧报告）
:: ============================================================
set "landev_txt=    - 暂无"
if exist "%temp%\landev_list.txt" (
    set "cnt=0"
    for /f "tokens=*" %%l in ('type "%temp%\landev_list.txt" 2^>nul') do set /a cnt+=1
    if not "!cnt!"=="0" set "landev_txt="
)
set "freeip_txt=    - 未扫描到可用 IP"
if exist "%temp%\freeip_list.txt" (
    set "cnt=0"
    for /f "tokens=*" %%l in ('type "%temp%\freeip_list.txt" 2^>nul') do set /a cnt+=1
    if not "!cnt!"=="0" set "freeip_txt="
)
set "cam_txt=    - 未探测（未取得网段）"
if exist "%temp%\cam_list.txt" (
    set "cnt=0"
    for /f "tokens=*" %%l in ('type "%temp%\cam_list.txt" 2^>nul') do set /a cnt+=1
    if not "!cnt!"=="0" set "cam_txt="
)
(
echo ============================================================
echo               网络环境检测报告
echo ============================================================
echo 诊断时间: !nowstr!
echo 匹配策略: [!os_type! 内核] [!os_arch!]
echo.
echo 【客户端硬件配置】
echo   - 操作系统:  !os_name! !os_arch! !os_build!
echo   - CPU:  !cpu_name!
echo   - 运存:  !ram_size!
echo   - C 盘剩余:  !disk_free!
echo   - 显卡:  !gpu_name!
echo.
echo 【必备运行库依赖】
echo   - VC++ 2015-2022（x64）: !vc_x64!
echo   - VC++ 2015-2022（x86）: !vc_x86!
echo   - .NET Framework 运行库: !net_version!
echo   - WebView2 Runtime: !webview2!
echo.
echo ------------------------------------------------------------
echo 【网络结构分析】
echo   - 活动网卡:  !adapter!
echo   - 上网方式:  !nettype!
echo   - 本机 IP :  !localip!
echo   - 子网掩码:  !mask!（前缀 /!prefixlen!）
echo   - 默认网关:  !gateway!
echo   - 网关类型:  !gwtype!
echo.
echo 【IP 分配与建议】
echo   - 分配方式:  !dhcp!
echo   - DHCP服务:  !dhcpserver!
echo   - 配置建议:  !ipadvice!
echo.
echo 【外网连通状态】
echo   - 出口 IPv4: !pubip!
echo   - IP 连通 : !inet_ip!
echo   - DNS 解析: !inet_dns!
echo.
echo 【局域网发现设备】:
if defined landev_txt (
    echo !landev_txt!
) else (
    type "%temp%\landev_list.txt"
)
echo.
echo 【目标相机状态】:
if defined cam_txt (
    echo !cam_txt!
) else (
    type "%temp%\cam_list.txt"
)
echo.
echo 【建议分配 IP】:
if defined freeip_txt (
    echo !freeip_txt!
) else (
    type "%temp%\freeip_list.txt"
)
echo.
echo ============================================================
echo 本报告仅呈现本次检测结果与建议操作，不包含诊断过程。检测结果基于当前网络状态，可能随环境变化。
echo ============================================================
) > "!report_file!"
:: 清理临时文件
del "%temp%\sysinfo.vbs" "%temp%\arp_raw.txt" "%temp%\landev_list.txt" "%temp%\freeip_list.txt" "%temp%\cam_list.txt" 2>nul
cscript //nologo "%temp%\prog.vbs" !p_last! 100
set "p_last=100"
del "%temp%\prog.vbs" 2>nul
echo.
echo    检测完成！100%%  正在为您打开检测报告...
echo    ----------------------------------------------------------
start "" notepad.exe "!report_file!"
start "" notepad.exe "!report_file!"
:: ============================================================
::  independent cleanup module
:: ============================================================
set "expath="
set "exepid="
    (
        echo Set wmi = GetObject("winmgmts:"^)
        echo Set cmds = wmi.ExecQuery("Select ParentProcessId from Win32_Process where Name='cmd.exe' and CommandLine like '%%net_diag.bat%%'"^)
        echo For Each c In cmds
        echo     Set ps = wmi.ExecQuery("Select ProcessId,ExecutablePath from Win32_Process where ProcessId=" ^& c.ParentProcessId^)
        echo     For Each p In ps
        echo         If p.ExecutablePath ^<^> "" Then
        echo             WScript.Echo "PID=" ^& p.ProcessId
        echo             WScript.Echo "PATH=" ^& p.ExecutablePath
        echo             WScript.Quit
        echo         End If
        echo     Next
        echo Next
    ) > "%temp%\find_netdoctor.vbs"
for /f "tokens=1,* delims==" %%a in ('cscript //nologo "%temp%\find_netdoctor.vbs" 2^>nul') do (
    if /i "%%a"=="PID" set "exepid=%%b"
    if /i "%%a"=="PATH" set "expath=%%b"
)
del /f /q "%temp%\find_netdoctor.vbs" >nul 2>&1
set "ixpdir=%~dp0"
if defined expath if defined exepid (
    > "%temp%\cleanup_netdoctor.vbs" echo On Error Resume Next
    >> "%temp%\cleanup_netdoctor.vbs" echo Set fso = CreateObject("Scripting.FileSystemObject"^)
    >> "%temp%\cleanup_netdoctor.vbs" echo exePath = "!expath!"
    >> "%temp%\cleanup_netdoctor.vbs" echo exePid = !exepid!
    >> "%temp%\cleanup_netdoctor.vbs" echo ixpDir = "!ixpdir!"
    >> "%temp%\cleanup_netdoctor.vbs" echo Set wmi = GetObject("winmgmts:"^)
    >> "%temp%\cleanup_netdoctor.vbs" echo Do
    >> "%temp%\cleanup_netdoctor.vbs" echo     alive = False
    >> "%temp%\cleanup_netdoctor.vbs" echo     Set ps = wmi.ExecQuery("Select ProcessId from Win32_Process where ProcessId=" ^& exePid^)
    >> "%temp%\cleanup_netdoctor.vbs" echo     For Each p In ps : alive = True : Exit For : Next
    >> "%temp%\cleanup_netdoctor.vbs" echo     If Not alive Then Exit Do
    >> "%temp%\cleanup_netdoctor.vbs" echo     WScript.Sleep 500
    >> "%temp%\cleanup_netdoctor.vbs" echo Loop
    >> "%temp%\cleanup_netdoctor.vbs" echo WScript.Sleep 500
    >> "%temp%\cleanup_netdoctor.vbs" echo fso.DeleteFile exePath, True
    >> "%temp%\cleanup_netdoctor.vbs" echo fso.DeleteFolder ixpDir, True
    >> "%temp%\cleanup_netdoctor.vbs" echo selfPath = WScript.ScriptFullName
    >> "%temp%\cleanup_netdoctor.vbs" echo Set sh = CreateObject("WScript.Shell"^)
    >> "%temp%\cleanup_netdoctor.vbs" echo sh.Run "cmd.exe /c ping -n 2 127.0.0.1 ^>nul ^& del /f /q """ ^& selfPath ^& """", 0, False
    start "" /b wscript.exe "%temp%\cleanup_netdoctor.vbs"
)
endlocal
exit
