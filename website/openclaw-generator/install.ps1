# OpenClaw 一键安装脚本 (Windows PowerShell)
# 使用方法：powershell -c "Invoke-RestMethod 'https://cloudmesh.top/openclaw-generator/install.ps1' -OutFile install.ps1; .\install.ps1"

# 颜色函数
function Write-Info { Write-Host "ℹ️  $args" -ForegroundColor Cyan }
function Write-Success { Write-Host "✅  $args" -ForegroundColor Green }
function Write-Warning { Write-Host "⚠️  $args" -ForegroundColor Yellow }
function Write-Error-Custom { Write-Host "❌  $args" -ForegroundColor Red }
function Write-Step { Write-Host "➜  $args" -ForegroundColor Blue }

Write-Host ""
Write-Host "╔════════════════════════════════════════╗"
Write-Host "║     🐾 OpenClaw 一键安装脚本          ║"
Write-Host "╚════════════════════════════════════════╝"
Write-Host ""

# 设置 TLS 1.2
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# 检查 Node.js
function Check-NodeJS {
    Write-Step "检查 Node.js..."
    
    try {
        $nodeVersion = node --version
        Write-Success "Node.js 已安装：$nodeVersion"
    } catch {
        Write-Warning "未检测到 Node.js，正在安装..."
        
        try {
            winget install OpenJS.NodeJS.LTS --silent --accept-package-agreements --accept-source-agreements
            Write-Success "Node.js 安装完成"
        } catch {
            $installerUrl = "https://nodejs.org/dist/v20.11.0/node-v20.11.0-x64.msi"
            $installerPath = "$env:TEMP\node-installer.msi"
            
            Invoke-WebRequest -Uri $installerUrl -OutFile $installerPath
            Start-Process msiexec.exe -Wait -ArgumentList "/i $installerPath /quiet"
            Remove-Item $installerPath -Force
            
            Write-Success "Node.js 安装完成"
        }
        
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
    }
}

# 检查 pnpm
function Check-PNPM {
    Write-Step "检查 pnpm..."
    
    try {
        $pnpmVersion = pnpm --version
        Write-Success "pnpm 已安装：$pnpmVersion"
    } catch {
        Write-Warning "安装 pnpm..."
        npm install -g pnpm
        Write-Success "pnpm 安装完成"
    }
}

# 创建目录
function Setup-Directories {
    Write-Step "创建工作目录..."
    
    $WorkDir = "$env:USERPROFILE\.openclaw"
    New-Item -ItemType Directory -Force -Path "$WorkDir\workspace" | Out-Null
    New-Item -ItemType Directory -Force -Path "$WorkDir\logs" | Out-Null
    
    Write-Success "工作目录：$WorkDir"
    
    return $WorkDir
}

# 安装 OpenClaw
function Install-OpenClaw {
    Write-Step "安装 OpenClaw..."
    
    npm install -g openclaw
    
    if ($LASTEXITCODE -eq 0) {
        Write-Success "OpenClaw 安装成功"
    } else {
        Write-Error-Custom "OpenClaw 安装失败"
        exit 1
    }
}

# 引导配置
function Guide-Config {
    Write-Step "配置 OpenClaw"
    Write-Host ""
    Write-Host "请选择通讯平台："
    Write-Host "  1) 钉钉"
    Write-Host "  2) 飞书"
    Write-Host "  3) 企业微信"
    Write-Host "  4) Telegram"
    Write-Host "  5) Discord"
    Write-Host "  6) Slack"
    Write-Host "  7) 稍后手动配置"
    Write-Host ""
    
    $choice = Read-Host "请选择 [1-7]"
    
    $envContent = "# OpenClaw 环境变量`n"
    
    switch ($choice) {
        1 {
            $appkey = Read-Host "请输入钉钉 AppKey"
            $appsecret = Read-Host "请输入钉钉 AppSecret"
            $envContent += "PLATFORM_PROVIDER=dingtalk`nDINGTALK_APPKEY=$appkey`nDINGTALK_APPSECRET=$appsecret`n"
            Write-Success "钉钉配置已保存"
        }
        2 {
            $appid = Read-Host "请输入飞书 App ID"
            $appsecret = Read-Host "请输入飞书 App Secret"
            $envContent += "PLATFORM_PROVIDER=feishu`nFEISHU_APPID=$appid`nFEISHU_APPSECRET=$appsecret`n"
            Write-Success "飞书配置已保存"
        }
        3 {
            $botid = Read-Host "请输入企业微信 BotID"
            $secret = Read-Host "请输入企业微信 Secret"
            $envContent += "PLATFORM_PROVIDER=wechat`nWECHAT_BOTID=$botid`nWECHAT_SECRET=$secret`n"
            Write-Success "企业微信配置已保存"
        }
        4 {
            $token = Read-Host "请输入 Telegram Bot Token"
            $envContent += "PLATFORM_PROVIDER=telegram`nTELEGRAM_TOKEN=$token`n"
            Write-Success "Telegram 配置已保存"
        }
        5 {
            $token = Read-Host "请输入 Discord Bot Token"
            $envContent += "PLATFORM_PROVIDER=discord`nDISCORD_TOKEN=$token`n"
            Write-Success "Discord 配置已保存"
        }
        6 {
            $token = Read-Host "请输入 Slack Bot Token"
            $envContent += "PLATFORM_PROVIDER=slack`nSLACK_TOKEN=$token`n"
            Write-Success "Slack 配置已保存"
        }
        default {
            Write-Warning "跳过配置，请手动编辑 .env 文件"
            $envContent += @"
PLATFORM_PROVIDER=dingtalk

# 钉钉配置
# DINGTALK_APPKEY=your_appkey
# DINGTALK_APPSECRET=your_appsecret

# 飞书配置
# FEISHU_APPID=your_appid
# FEISHU_APPSECRET=your_appsecret

# 企业微信配置
# WECHAT_BOTID=your_botid
# WECHAT_SECRET=your_secret

# Telegram 配置
# TELEGRAM_TOKEN=your_token

# Discord 配置
# DISCORD_TOKEN=your_token

# Slack 配置
# SLACK_TOKEN=your_token
"@
        }
    }
    
    $envContent | Out-File -FilePath "$WorkDir\workspace\.env" -Encoding UTF8
    
    # 创建 YAML 配置
    $yamlContent = @"
# OpenClaw 配置文件
platform:
  provider: dingtalk

agents:
  - name: Assistant
    role: 智能助手
    skills:
      - conversation
      - memory
      - web_search
    model: bailian/qwen3.5-plus

orchestration:
  mode: sequential
  timeout: 3600

runtime:
  workdir: $WorkDir\workspace
  default_model: bailian/qwen3.5-plus
  thinking: auto
"@
    
    $yamlContent | Out-File -FilePath "$WorkDir\workspace\openclaw.yaml" -Encoding UTF8
    Write-Success "配置文件已创建"
}

# 创建开机自启
function Create-StartupShortcut {
    Write-Step "创建开机自启快捷方式..."
    
    $WshShell = New-Object -ComObject WScript.Shell
    $ShortcutPath = "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Startup\OpenClaw.lnk"
    $Shortcut = $WshShell.CreateShortcut($ShortcutPath)
    $Shortcut.TargetPath = "C:\Windows\System32\cmd.exe"
    $Shortcut.Arguments = "/c `"openclaw start`""
    $Shortcut.WorkingDirectory = "$env:USERPROFILE\.openclaw\workspace"
    $Shortcut.Save()
    
    Write-Success "开机自启快捷方式已创建"
}

# 显示完成信息
function Show-Complete {
    Write-Host ""
    Write-Host "╔════════════════════════════════════════╗"
    Write-Host "║       🎉 OpenClaw 安装完成！          ║"
    Write-Host "╚════════════════════════════════════════╝"
    Write-Host ""
    Write-Info "工作目录：$WorkDir\workspace"
    Write-Info "配置文件：$WorkDir\workspace\openclaw.yaml"
    Write-Host ""
    Write-Info "管理命令:"
    Write-Host "  openclaw start    - 启动服务"
    Write-Host "  openclaw stop     - 停止服务"
    Write-Host "  openclaw restart  - 重启服务"
    Write-Host "  openclaw status   - 查看状态"
    Write-Host "  openclaw logs     - 查看日志"
    Write-Host ""
    Write-Info "文档：https://docs.openclaw.ai"
    Write-Info "Skill 市场：https://cloudmesh.top/skills/"
    Write-Info "配置生成器：https://cloudmesh.top/openclaw-generator/"
    Write-Host ""
    Write-Info "提示：请重启 PowerShell 后运行 openclaw start 启动服务"
}

# 主流程
function Main {
    Check-NodeJS
    Check-PNPM
    $WorkDir = Setup-Directories
    Install-OpenClaw
    Guide-Config
    Create-StartupShortcut
    Show-Complete
}

# 执行
Main
