#!/bin/bash
# 示例脚本 - 显示系统信息

echo "=== 系统信息 ==="
echo "主机名: $(hostname)"
echo "当前用户: $(whoami)"
echo "当前时间: $(date)"
echo "系统负载: $(uptime)"
echo ""
echo "=== 磁盘使用情况 ==="
df -h | head -5
echo ""
echo "脚本执行完成"
