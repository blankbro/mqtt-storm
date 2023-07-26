# 检查是否存在mqtt-storm进程
if pgrep mqtt-storm > /dev/null
then
    # 如果存在则通过kill命令杀死进程
    echo "开始关闭"
    pkill mqtt-storm
    echo "Done"
else
    echo "进程已关闭"
fi
