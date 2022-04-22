说明

微信文章:

第一次提交:
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



第二次提交:全局替换
仓库已经推上去 
本地写代码 examples\nusclient\main.go 应用它
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

