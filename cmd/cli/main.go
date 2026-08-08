// Package main 是 KvirNet 命令行客户端入口。
//
// 启动流程：加载 .env -> 解析 flag -> 启动 client.Run()。
// 服务器地址优先级：命令行 -server > 环境变量 VIRNET_SERVER（含 .env）> 默认硬编码。
package main

import (
	"flag"

	"virtualnet/client"
	"virtualnet/common"
)

func main() {
	// 先加载 .env，再解析 flag，保证 flag 默认值能读到 .env 注入的环境变量
	common.LoadEnv()
	flag.Parse()

	client.Run()
}
