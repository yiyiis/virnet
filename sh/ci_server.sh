#!/bin/bash
set -e  # 遇到错误立即退出脚本

# 定义颜色变量用于输出提示（可选，增强可读性）
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # 无颜色

echo -e "${GREEN}===== 开始执行部署脚本 =====${NC}"

# 1. 克隆代码仓库
echo -e "${GREEN}1. 克隆代码仓库...${NC}"
rm -rf /root/gamevpn
if ! git clone git@xplaza.cn:klen/gamevpn.git; then
    echo -e "${RED}错误：克隆仓库失败，请检查仓库地址和权限${NC}"
    exit 1
fi

# 2. 进入server目录（注意：克隆后会生成gamevpn文件夹，需先进入该文件夹）
echo -e "${GREEN}2. 进入server目录...${NC}"
cd gamevpn || {
    echo -e "${RED}错误：进入gamevpn目录失败${NC}"
    exit 1
}
cd server || {
    echo -e "${RED}错误：进入server目录失败${NC}"
    exit 1
}

# 3. 执行go build编译
echo -e "${GREEN}3. 编译server程序...${NC}"
if ! go build; then
    echo -e "${RED}错误：go build编译失败，请检查Go环境和代码${NC}"
    exit 1
fi

# 4. 终止原有server进程（精确匹配目标程序路径，避免误杀）
echo -e "${GREEN}5. 终止原有进程...${NC}"
pid=$(ps aux | grep './server'  | head -n 1 |  awk '{print $2}')
if [ -n "$pid" ]; then
    echo -e "发现原有进程，PID: $pid，正在终止..."
    if ! kill "$pid"; then
        echo -e "${RED}警告：终止进程失败，可能进程已退出${NC}"
    fi
else
    echo -e "未发现原有进程，无需终止"
fi

# 5. 移动编译产物到目标路径
echo -e "${GREEN}4. 移动程序到/root/vpnserver...${NC}"
if ! mv server /root/vpnserver; then
    echo -e "${RED}错误：移动程序失败，请检查目标路径权限${NC}"
    exit 1
fi


# 6. 添加执行权限（修正原步骤路径错误）
echo -e "${GREEN}6. 添加执行权限...${NC}"
if ! chmod +x /root/vpnserver/server; then
    echo -e "${RED}错误：添加执行权限失败${NC}"
    exit 1
fi

# 7. 后台启动程序并记录日志
echo -e "${GREEN}7. 启动服务...${NC}"
nohup /root/vpnserver/server > /root/vpn_run.log 2>&1 &


# 检查启动是否成功
sleep 2
new_pid=$(ps aux | grep './server'  | head -n 1 |  awk '{print $2}')
if [ -n "$new_pid" ]; then
    echo -e "${GREEN}部署成功！服务进程PID: $new_pid，日志路径：/root/vpn_run.log${NC}"
else
    echo -e "${RED}错误：服务启动失败，请检查日志${NC}"
    exit 1
fi

