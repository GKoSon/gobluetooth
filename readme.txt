说明

微信文章:

第一次提交:准备简化的代码仓库

项目来源git clone https://github.com/tinygo-org/bluetooth.git
克隆TingGo仓库 
commit e75811786c7ec1f2890e1ff0508cc28d5ac5de62 
    release: update for v0.4.0

因为我是WIN10+VSCODE开发 再拖到树莓派运行
所以我删除无关紧要的东西
执行rm.cmd 
就获得当前项目的一切！
描述一下如何执行这个rm.cmd
{
直接双击rm.cmd 会失败的
因为win的bash不支持通配符*

根据VSCODE的启发(微信文章:) 
可以换一个bash
C:\Program Files\Git\bin
这里有3个 我选择这里的bash.exe
本来准备 C:\Program Files\Git\bin\bash.exe rm.cmd 的
发现路径有问题
直接 bash.exe rm.cmd 就成功了！
应该是因为前面已经把git配置到系统环境变量啦！
}


建立本仓库 是为了方便自己再调用它！
计划修改
1--HCI接口切换
2--断开检查



第二次提交: 全局替换
前面仓库已经推上去 
本地写代码 examples\nusclient\main.go 引用它
以前是 import 	"tinygo.org/x/bluetooth"
我就写 import 	"GKoSon/gobluetooth"
测试失败！
因为我这个项目的mod文件还是原始的
本次提交 做全局替换 tinygo.org/x/bluetooth -> GKoSon/gobluetooth
module tinygo.org/x/bluetooth


第三次提交:全局替换
本地测试还是不行

在本地执行go mod init xx
发现mod文件只有2行 没有关联到我的仓库
我手动修改一下本地的mod文件如图 
...
module xx

go 1.17

require (
	github.com/GKoSon/gobluetooth v1.2.3
)
...
【后面的v1.2.3是随便写的 根据ssh提示来的】
在做单个文件go build main.go 看到有去拉我的代码啦！
go mod download github.com/GKoSon/gobluetooth
root@raspberrypi:/home/pi/PK# go mod download github.com/GKoSon/gobluetooth
go mod download: github.com/GKoSon/gobluetooth@v1.2.3: invalid version: unknown revision v1.2.3
最后失败 换一个姿势
root@raspberrypi:/home/pi/PK# go mod tidy
xx imports
        GKoSon/gobluetooth: github.com/GKoSon/gobluetooth@v1.2.3: reading github.com/GKoSon/gobluetooth/go.mod at revision v1.2.3: unknown revision v1.2.3
也是失败！！！


所以问题是:go.mod文件里面require项目最后的版本填写多少呢?
参考:http://t.zoukankan.com/twoheads-p-12889526.html

https://github.com/GKoSon/gobluetooth/commits/main
看到最后一个commit是 462c67d2a1ba951d39b7587e6e53053764b73284
我就写这个！
...
module xx

go 1.17

require github.com/GKoSon/gobluetooth v0.0.0-20220422050553-462c67d2a1ba
...
执行报错

root@raspberrypi:/home/pi/PK# go mod tidy
xx imports
        GKoSon/gobluetooth: github.com/GKoSon/gobluetooth@v0.0.0-20220422050553-462c67d2a1ba: parsing go.mod:
        module declares its path as: GKoSon/gobluetooth
                but was required as: github.com/GKoSon/gobluetooth



修改以后 再来一次
root@raspberrypi:/home/pi/PK# go mod tidy 
xx imports
        GKoSon/gobluetooth: malformed module path "GKoSon/gobluetooth": missing dot in first path element
root@raspberrypi:/home/pi/PK# cat go.mod 
module xx

go 1.17

require GKoSon/gobluetooth v0.0.0-20220422050553-462c67d2a1ba
root@raspberrypi:/home/pi/PK# 

百度一下 好像是有问题 "一个网络地址去下载,在下载前先CheckPath检查其合法性"
好像明白了 那就再次全局修改一下

以前是 import 	"tinygo.org/x/bluetooth"
前面写 import 	"GKoSon/gobluetooth"
现在写import 	"github.com/GKoSon/gobluetooth"
这就是下载的路径啊！！！！
测试一下
【感觉还是fork一下 在修改fork的仓库+replace的方式比较简单】



第四次提交:成功了
前面提交以后 我自己记录一下commit
手动修改本地文件
...
module xx

go 1.17

require github.com/GKoSon/gobluetooth a31499c265a17b1894e823ef49246b2b45d3a555
...
现在执行 go mod tidy 可以成功了！

root@raspberrypi:/home/pi/PK# go mod tidy      
go: downloading github.com/GKoSon/gobluetooth v0.0.0-20220422055325-a31499c265a1
root@raspberrypi:/home/pi/PK# go build main.go 
# command-line-arguments
./main.go:121:13: device.IsConnected undefined (type *bluetooth.Device has no field or method IsConnected)
root@raspberrypi:/home/pi/PK# 


再看一下 GO帮我做的事情 (其实没有意义)
root@raspberrypi:/home/pi/PK# cat go.mod 
module xx

go 1.17

require github.com/GKoSon/gobluetooth v0.0.0-20220422055325-a31499c265a1

require (
        github.com/fatih/structs v1.1.0 // indirect
        github.com/godbus/dbus/v5 v5.0.3 // indirect
        github.com/konsorten/go-windows-terminal-sequences v1.0.3 // indirect
        github.com/muka/go-bluetooth v0.0.0-20210812063148-b6c83362e27d // indirect
        github.com/sirupsen/logrus v1.6.0 // indirect
        golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6 // indirect
)


本地的单例程序上传上来 
可以独立在PI运行的!
修改需要我自己的函数！


第五次提交:
直接把下面文件 放在台湾树莓派运行
流程如下
    pi@raspberrypi:~/XX $ go build main.go 
    main.go:10:2: no required module provides package github.com/GKoSon/gobluetooth: go.mod file not found in current directory or any parent directory; see 'go help modules'
    pi@raspberrypi:~/XX $ go mod init xx
    go: creating new go.mod: module xx
    go: to add module requirements and sums:
            go mod tidy
    pi@raspberrypi:~/XX $ go mod tidy
    go: finding module for package github.com/GKoSon/gobluetooth
    go: downloading github.com/GKoSon/gobluetooth v0.0.0-20220422070120-588124614c38
    go: found github.com/GKoSon/gobluetooth in github.com/GKoSon/gobluetooth v0.0.0-20220422070120-588124614c38
    go: downloading github.com/godbus/dbus/v5 v5.0.3
    go: downloading github.com/sirupsen/logrus v1.6.0
    go: downloading github.com/stretchr/testify v1.6.1
    go: downloading github.com/konsorten/go-windows-terminal-sequences v1.0.3
    go: downloading golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6
    go: downloading github.com/davecgh/go-spew v1.1.1
    go: downloading github.com/pmezard/go-difflib v1.0.0
    go: downloading gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
    pi@raspberrypi:~/XX $ go build main.go
    main.go:10:2: found packages bluetooth (adapter.go) and main (koson_pi_nus.go) in /home/pi/go/pkg/mod/github.com/!g!ko!son/gobluetooth@v0.0.0-20220422070120-588124614c38
    pi@raspberrypi:~/XX $ 
    这是因为我自己代码里面koson_pi_nus.go也有mian.go
    提交一次del它
    
    
第六次提交:
本地写代码 如下 可以的
操作是和上面说的一样 利用mod 自动拉 
如果更新mod 就手动把V1.2.3修改为最后的commit

package main

// This example implements a NUS (Nordic UART Service) client. See nusserver for
// details.

import (
	"log"
	"time"

	bluetooth "github.com/GKoSon/gobluetooth"
)

var (
	serviceUUID = bluetooth.ServiceUUIDNordicUART
	rxUUID      = bluetooth.CharacteristicUUIDUARTRX
	txUUID      = bluetooth.CharacteristicUUIDUARTTX
)

var adapter = bluetooth.DefaultAdapter

const target_name = "M_KOSON"

var runCnt int64 = 0

func oneloop() {
	// Enable BLE interface.
	//STEP 准备通过这个接口 选择不同HCI
	err := adapter.Enable()
	if err != nil {
		log.Printf("could not enable the BLE stack:%v", err.Error())
		return
	}

	// The address to connect to. Set during scanning and read afterwards.
	var foundDevice bluetooth.ScanResult

	// Scan for NUS peripheral.
	runCnt++
	log.Printf("Scanning...%d", runCnt)
	err = adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if !result.AdvertisementPayload.HasServiceUUID(serviceUUID) {
			return
		}
		/*加强停止Scan函数的限制条件*/
		/*条件1--从机必须有名字 而且符合约定*/
		log.Printf("Scanned name 1-%s 2-%s", foundDevice.AdvertisementPayload.LocalName(), foundDevice.LocalName())
		if target_name != foundDevice.AdvertisementPayload.LocalName() {
			log.Printf("Failed name")
			return
		}

		/*条件2--从机连接成功*/
		d, e := adapter.Connect(foundDevice.Address, bluetooth.ConnectionParams{})
		if e != nil {
			log.Printf("Failed Connect %v %v", d, e)
			return
		}

		foundDevice = result

		// Stop the scan.
		err := adapter.StopScan()
		if err != nil {
			// Unlikely, but we can't recover from this.
			log.Printf("Failed StopScan %v ", err.Error())
		}
	})

	if err != nil {
		log.Printf("Failed Scan %v ", err.Error())
		return
	}

	// Found a NUS peripheral. Connect to it.
	device, err := adapter.Connect(foundDevice.Address, bluetooth.ConnectionParams{})
	if err != nil {
		log.Printf("TG Failed to connect: %v ", err.Error())
		return
	}

	// Connected. Look up the Nordic UART Service.
	log.Printf("Discovering service...")
	services, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		println("Failed to discover the Nordic UART Service:", err.Error())
		return
	}
	service := services[0]

	// Get the two characteristics present in this service.
	chars, err := service.DiscoverCharacteristics([]bluetooth.UUID{rxUUID, txUUID})
	if err != nil {
		println("Failed to discover RX and TX characteristics:", err.Error())
		return
	}
	var rx bluetooth.DeviceCharacteristic
	var tx bluetooth.DeviceCharacteristic
	if chars[0].UUID() == txUUID {
		tx = chars[0]
		rx = chars[1]
	} else {
		tx = chars[1]
		rx = chars[0]
	}
	log.Printf("RX %v\r\n", rx)
	log.Printf("TX %v\r\n", tx)
	log.Printf("DiscoverCharacteristics:%+v\r\n", chars)

	// Enable notifications to receive incoming data.
	err = tx.EnableNotifications(func(value []byte) {
		log.Printf("PI recv %d bytes: %+v\r\n", len(value), value)
	})
	if err != nil {
		log.Printf("Failed EnableNotifications %+v\r\n", err.Error())
		return
	}

	log.Printf("Connected.When NODE disconnect.This pid while Exit\r\n")
	/*等待从机断开 PI从不发消息*/
	for {
		if !device.IsConnected() {
			log.Printf("device GoodBye\r\n")
			return
		}
		time.Sleep(time.Microsecond * 500)
	}
}

func main() {
	for {
		oneloop()
	}
}

/*
程序逻辑
main是死循环 也就是一旦oneloop()调用return那就继续下一次
正常retun是等待BLE链路的标记位

*/







现在小结一下两种思路
1---就是我这个 
把别人的拉下来
rm别人的.git文件
初始化自己的git文件
把代码推上来
全局修改仓库里面的mod的包名
现在可以自己本地引用这个新的包了
和别人的 完全脱钩 你感觉不到别人


2--我另外一个fork的
把别人的fork到自己GitHub
把自己这个仓库拉下来
自己随便ADD函数什么的 自己修改 在推上去

现在不可以自己本地引用这个新的包 因为没有新包呀
本地的代码 和前面一样 还是在使用原始仓库的包
想使用自己fork的这个的修改的话 那就自己把本地文件的mod文件修改一下
采用replace的关键字
从一个云端 切换到 另外一个云端


准备微信写一个文章:攻城狮金华


