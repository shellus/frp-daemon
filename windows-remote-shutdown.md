# Windows远程关机设置向导
本文在全新安装关闭防火墙的`Windows10 LTSC 2019`上测试通过(SW_DVD9_WIN_ENT_LTSC_2021_64BIT_ChnSimp_MLF_X22-84402.ISO)

## 1. 主控端安装远程关机服务
```bash
apt update && apt install -y samba-common-bin
```

## 2. 被控端配置远程关机服务
```powershell
# 检查并设置 LocalAccountTokenFilterPolicy=1
if ((Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System" -Name "LocalAccountTokenFilterPolicy" -ErrorAction SilentlyContinue) -eq $null) {
    New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System" -Name "LocalAccountTokenFilterPolicy" -Value 1 -PropertyType DWORD
} elseif ((Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System" -Name "LocalAccountTokenFilterPolicy").LocalAccountTokenFilterPolicy -ne 1) {
    Set-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System" -Name "LocalAccountTokenFilterPolicy" -Value 1
}

# 设置为自动启动
Set-Service -Name RemoteRegistry -StartupType Automatic
# 启动服务
Start-Service RemoteRegistry
```

## 3. 启动远程关机服务
```bash
net rpc shutdown -I <windows IP> -U "<windows-username>%<windows-password>"
```
