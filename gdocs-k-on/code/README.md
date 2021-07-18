# Naive gDocs

一个做了很多简化设计的类 gDocs 文件：

支持多人协作编辑的简易表格项目

---

## 例子

以下三个部分是用于学习 go 中对 rpc、websocket、zookeeper 的使用

### go_client
是 rpc_client 和连接 zookeeper 的例子

### go_server
是 rpc_server 的例子 注册了两个函数：乘法和除法

### go_websocket_server
是作为与前端进行 websocket 通信的例子

---

# 项目部分

## 底层架构

如果自身的 go 版本不支持 go mod tidy，则启动下列服务时会有缺少的依赖，可根据提示来安装相应的包，例：
```shell
go get github.com/go-zookeeper/zk
```

如果支持 go mod tidy，可在各目录下直接使用 go mod tidy 完成依赖的安装

国内安装依赖可能需要 go proxy：

Powershell (windows) 下：
```shell
$env:GOPROXY = "https://goproxy.io,direct"
```
Bash (Linux or macOS) 下：
```shell
export GOPROXY=https://goproxy.io,direct
```

### chunkserver

chunkserver 部分为管理 chunk 部分的服务器，一般默认启动三个，且启动的端口固定：
```shell
go run ./chunkserver.go ./rpcToMaster.go ./rpcToOtherChunk.go ./lock.go  8100
go run ./chunkserver.go ./rpcToMaster.go ./rpcToOtherChunk.go  ./lock.go  8101
go run ./chunkserver.go ./rpcToMaster.go ./rpcToOtherChunk.go  ./lock.go 8102
```

这样就启动了三个 chunkserver

### master

master 部分为管理 chunk 元数据的服务器，一般默认启动三个，且启动的端口固定：
```shell
go run ./master.go ./lock.go ./rpcToChunkServer.go ./rpcToReplica.go 8095
go run ./master.go ./lock.go ./rpcToChunkServer.go ./rpcToReplica.go 8096
go run ./master.go ./lock.go ./rpcToChunkServer.go ./rpcToReplica.go 8097
```

### webserver

webserver 为后端，同时也作为分布式文件系统的客户端，调用 gfs_api 中的接口，来创建、删除、写入、读取文件
启动方式：
```shell
go run ./webserver.go ./lock.go ./json_struct.go ./gfs_api.go util_ws.go handlers_ws.go
```

### 前端

当上述服务启动完毕后，便可以去 NaiveDocFrontend 文件夹下启动前端。
首先要确保自己装有 yarn。
```shell
yarn install
yarn start
```

---