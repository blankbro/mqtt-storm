#!/bin/bash
# 检查是否存在mqtt-storm进程
if pgrep mqtt-storm > /dev/null
then
    # 如果存在则通过kill命令杀死进程
    echo "关闭已有进程"
    pkill mqtt-storm
fi

# 获取操作系统类型
OS=$(uname)
echo "当前操作系统为${OS}"

OPTS="-broker mqtt://127.0.0.1:1883 -username admin -password admin -c 10"
echo "开始启动"
if [ "${OS}" == "Darwin" ]; then
    echo "Running on macOS"
    nohup ../output/mac/mqtt-storm ${OPTS} > test.log 2>&1 &
elif [ "${OS}" == "Linux" ]; then
    echo "Running on Linux"
    nohup ../output/linux/mqtt-storm ${OPTS} > test.log 2>&1 &
else
    echo "Unsupported operating system: $OS"
    exit 1
fi
echo "Done"
