#!/bin/bash
# OpenClaw 一键安装脚本 (Linux / macOS)
# 使用方法：curl -fsSL "https://cloudmesh.top/openclaw-generator/install.sh" | bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 打印函数
print_info() { echo -e "${CYAN}ℹ️  $1${NC}"; }
print_success() { echo -e "${GREEN}✅  $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌  $1${NC}"; }
print_step() { echo -e "${BLUE}➜  $1${NC}"; }

echo ""
echo "╔════════════════════════════════════════╗"
echo "║     🐾 OpenClaw 一键安装脚本          ║"
echo "╚════════════════════════════════════════╝"
echo ""

# 检查系统
OS=""
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
    print_info "检测到 Linux 系统"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="macos"
    print_info "检测到 macOS 系统"
else
    print_error "不支持的操作系统"
    exit 1
fi

# 检查依赖
check_dependencies() {
    print_step "检查系统依赖..."
    
    # 检查 Node.js
    if ! command -v node &> /dev/null; then
        print_warning "未检测到 Node.js，正在安装..."
        if [[ "$OS" == "linux" ]]; then
            if command -v apt &> /dev/null; then
                curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
                sudo apt-get install -y nodejs
            elif command -v yum &> /dev/null; then
                curl -fsSL https://rpm.nodesource.com/setup_20.x | sudo bash -
                sudo yum install -y nodejs
            elif command -v pacman &> /dev/null; then
                sudo pacman -S nodejs npm
            fi
        elif [[ "$OS" == "macos" ]]; then
            if ! command -v brew &> /dev/null; then
                /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
            fi
            brew install node
        fi
        print_success "Node.js 安装完成"
    else
        NODE_VERSION=$(node --version)
        print_success "Node.js 已安装：$NODE_VERSION"
    fi
    
    # 检查 npm/pnpm
    if ! command -v pnpm &> /dev/null; then
        print_info "安装 pnpm..."
        npm install -g pnpm
        print_success "pnpm 安装完成"
    else
        PNPM_VERSION=$(pnpm --version)
        print_success "pnpm 已安装：$PNPM_VERSION"
    fi
}

# 创建目录
setup_directories() {
    print_step "创建工作目录..."
    WORKDIR="$HOME/.openclaw"
    mkdir -p "$WORKDIR/workspace"
    mkdir -p "$WORKDIR/logs"
    print_success "工作目录：$WORKDIR"
}

# 安装 OpenClaw
install_openclaw() {
    print_step "安装 OpenClaw..."
    npm install -g openclaw
    if [ $? -eq 0 ]; then
        print_success "OpenClaw 安装成功"
        OPENCLAW_VERSION=$(openclaw --version 2>/dev/null || echo "latest")
        print_info "版本：$OPENCLAW_VERSION"
    else
        print_error "OpenClaw 安装失败"
        exit 1
    fi
}

# 引导配置
guide_config() {
    print_step "配置 OpenClaw"
    echo ""
    echo "请选择通讯平台："
    echo "  1) 钉钉"
    echo "  2) 飞书"
    echo "  3) 企业微信"
    echo "  4) Telegram"
    echo "  5) Discord"
    echo "  6) Slack"
    echo "  7) 稍后手动配置"
    echo ""
    read -p "请选择 [1-7]: " platform_choice
    
    case $platform_choice in
        1)
            read -p "请输入钉钉 AppKey: " appkey
            read -p "请输入钉钉 AppSecret: " appsecret
            cat > "$WORKDIR/workspace/.env" << EOF
PLATFORM_PROVIDER=dingtalk
DINGTALK_APPKEY=$appkey
DINGTALK_APPSECRET=$appsecret
EOF
            print_success "钉钉配置已保存"
            ;;
        2)
            read -p "请输入飞书 App ID: " appid
            read -p "请输入飞书 App Secret: " appsecret
            cat > "$WORKDIR/workspace/.env" << EOF
PLATFORM_PROVIDER=feishu
FEISHU_APPID=$appid
FEISHU_APPSECRET=$appsecret
EOF
            print_success "飞书配置已保存"
            ;;
        3)
            read -p "请输入企业微信 BotID: " botid
            read -p "请输入企业微信 Secret: " secret
            cat > "$WORKDIR/workspace/.env" << EOF
PLATFORM_PROVIDER=wechat
WECHAT_BOTID=$botid
WECHAT_SECRET=$secret
EOF
            print_success "企业微信配置已保存"
            ;;
        4)
            read -p "请输入 Telegram Bot Token: " token
            cat > "$WORKDIR/workspace/.env" << EOF
PLATFORM_PROVIDER=telegram
TELEGRAM_TOKEN=$token
EOF
            print_success "Telegram 配置已保存"
            ;;
        5)
            read -p "请输入 Discord Bot Token: " token
            cat > "$WORKDIR/workspace/.env" << EOF
PLATFORM_PROVIDER=discord
DISCORD_TOKEN=$token
EOF
            print_success "Discord 配置已保存"
            ;;
        6)
            read -p "请输入 Slack Bot Token: " token
            cat > "$WORKDIR/workspace/.env" << EOF
PLATFORM_PROVIDER=slack
SLACK_TOKEN=$token
EOF
            print_success "Slack 配置已保存"
            ;;
        *)
            print_warning "跳过配置，请手动编辑 $WORKDIR/workspace/.env"
            cat > "$WORKDIR/workspace/.env" << EOF
# OpenClaw 环境变量
# 请根据实际使用的平台填写配置

# 平台选择：dingtalk | feishu | wechat | telegram | discord | slack
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
# DISCORD_APPID=your_appid

# Slack 配置
# SLACK_TOKEN=your_token
EOF
            ;;
    esac
    
    # 创建基础配置文件
    cat > "$WORKDIR/workspace/openclaw.yaml" << EOF
# OpenClaw 配置文件
# 生成时间：$(date -Iseconds)

platform:
  provider: \${PLATFORM_PROVIDER}

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
  workdir: $WORKDIR/workspace
  default_model: bailian/qwen3.5-plus
  thinking: auto
EOF
    print_success "配置文件已创建"
}

# 初始化
initialize() {
    print_step "初始化 OpenClaw..."
    cd "$WORKDIR/workspace"
    openclaw init 2>/dev/null || true
    print_success "初始化完成"
}

# 启动服务
start_service() {
    echo ""
    read -p "是否现在启动 OpenClaw 服务？[Y/n]: " start_choice
    if [[ $start_choice != "n" && $start_choice != "N" ]]; then
        print_step "启动 OpenClaw 服务..."
        cd "$WORKDIR/workspace"
        openclaw start &
        print_success "OpenClaw 已启动"
        
        # 创建 systemd 服务文件（Linux）
        if [[ "$OS" == "linux" ]] && command -v systemctl &> /dev/null; then
            echo ""
            read -p "是否设置开机自启？[Y/n]: " autostart
            if [[ $autostart != "n" && $autostart != "N" ]]; then
                sudo tee /etc/systemd/system/openclaw.service > /dev/null << EOF
[Unit]
Description=OpenClaw Service
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$WORKDIR/workspace
ExecStart=$(which openclaw) start
Restart=always
Environment=PATH=/usr/bin:/usr/local/bin

[Install]
WantedBy=multi-user.target
EOF
                sudo systemctl daemon-reload
                sudo systemctl enable openclaw
                print_success "已设置开机自启"
            fi
        fi
    fi
}

# 显示完成信息
show_complete() {
    echo ""
    echo "╔════════════════════════════════════════╗"
    echo "║       🎉 OpenClaw 安装完成！          ║"
    echo "╚════════════════════════════════════════╝"
    echo ""
    print_info "工作目录：$WORKDIR/workspace"
    print_info "配置文件：$WORKDIR/workspace/openclaw.yaml"
    echo ""
    print_info "管理命令:"
    echo "  openclaw start    - 启动服务"
    echo "  openclaw stop     - 停止服务"
    echo "  openclaw restart  - 重启服务"
    echo "  openclaw status   - 查看状态"
    echo "  openclaw logs     - 查看日志"
    echo ""
    print_info "文档：https://docs.openclaw.ai"
    print_info "Skill 市场：https://cloudmesh.top/skills/"
    print_info "配置生成器：https://cloudmesh.top/openclaw-generator/"
    echo ""
}

# 主流程
main() {
    check_dependencies
    setup_directories
    install_openclaw
    guide_config
    initialize
    start_service
    show_complete
}

# 执行
main
