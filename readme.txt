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



第二次提交:
仓库已经推上去 
本地写代码 examples\nusclient\main.go 应用它
以前是 import 	"tinygo.org/x/bluetooth"
我就写 import 	"GKoSon/gobluetooth"
测试失败！
因为我这个项目的mod文件还是原始的
本次提交 做全局替换 tinygo.org/x/bluetooth -> GKoSon/gobluetooth
module tinygo.org/x/bluetooth
