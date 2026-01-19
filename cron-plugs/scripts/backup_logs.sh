#!/bin/bash
# 日志备份脚本示例

LOG_PATH=${1:-"/var/log"}
BACKUP_DIR="./backups"

echo "开始备份日志..."
echo "源目录: $LOG_PATH"
echo "备份目录: $BACKUP_DIR"

# 创建备份目录
mkdir -p "$BACKUP_DIR"

# 压缩备份(示例)
BACKUP_FILE="$BACKUP_DIR/logs_$(date +%Y%m%d_%H%M%S).tar.gz"
echo "备份文件: $BACKUP_FILE"

# 这里只是示例,实际使用时根据需求修改
echo "备份完成(示例)"
exit 0
