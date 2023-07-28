# MQTT-Storm: MQTT Stress Testing Tool

像风暴一样冲击和评估MQTT服务器的性能

## Build

```shell
$ git clone https://github.com/timeway/mqtt-storm.git
$ cd mqtt-storm
$ ./build.sh
```

## Run

```shell
$ ./control.sh stop
$ ./control.sh start
$ ./control.sh restart
```

## Storm

see [http-client.http](http-client.http)

## Custom Mock

update [customocker.go](internal/customocker/customocker.go)
