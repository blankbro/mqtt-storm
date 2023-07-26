#!/bin/bash
Stop(){
  # 检查是否存在mqtt-storm进程
  if pgrep mqtt-storm > /dev/null
  then
      # 如果存在则通过kill命令杀死进程
      echo "开始关闭"
      pkill mqtt-storm
      for ((i=0; i<10; ++i))
        do
          sleep 1
          if pgrep mqtt-storm > /dev/null
          then
            echo -e ".\c"
          else
            echo 'Stop Success!'
            break;
          fi
        done

      if pgrep mqtt-storm > /dev/null
      then
        echo 'Kill Process!'
        pkill -9 mqtt-storm
      fi
  else
      echo "进程已关闭"
  fi
}

Start(){
  OS=$(uname)
  echo "当前操作系统为${OS}"

  OPTS="-broker mqtt://127.0.0.1:1883 -username admin -password admin -c 10"
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
}

case $1 in
    "start" )
        Start
    ;;
    "stop" )
       Stop
    ;;
    "restart" )
       Stop
       Start
    ;;
    * )
        echo "unknown command"
        exit 1
    ;;
esac