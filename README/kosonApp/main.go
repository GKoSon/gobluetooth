package main

import (
	"log"
	"os"
	"strings"
	"time"

	bluetooth "github.com/GKoSon/gobluetooth"
)

var (
	serviceUUID = bluetooth.ServiceUUIDNordicUART
	rxUUID      = bluetooth.CharacteristicUUIDUARTRX
	txUUID      = bluetooth.CharacteristicUUIDUARTTX
)

var adapter = bluetooth.DefaultAdapter
var device *bluetooth.Device
var foundDevice bluetooth.ScanResult

//const target_name = "M_IZAR_ESP_TEST"
const target_name = "M_IZAR_TEST"

//const target_name = "M_KOSON"

var runCnt int64 = 0
var yesCnt int64 = 0
var failCnt int64 = 0
var rxdatacount int64 = 0
var rxdatanumber int64 = 0

func measureTime(funcName string) func() {
	start := time.Now()
	return func() {
		log.Printf("Time taken by %s function is %v \n", funcName, time.Since(start))
	}
}
func String_rm_char(a string, b string) string {
	mac := ""
	str := strings.Split(a, b)
	for _, s := range str {
		mac += s
	}
	return mac
}
func hciinit() bool {
	var h string
	if os.Args[1] == string("1") {
		h = "hci1"
	} else if os.Args[1] == string("0") {
		h = "hci0"
	} else {
		log.Printf("please input 0 1 as hci")
		return false
	}
	adapter.Hello()
	adapter.SetHciId(h)
	err := adapter.Enable()
	if err != nil {
		log.Printf("could not enable the BLE stack:%v", err.Error())
		return false
	}
	log.Printf("useing[%s][%s]", h, adapter.Mac)
	M, err := adapter.Address()
	log.Printf("useing[%#v][%v]", M, err)
	log.Printf("useing[%v]", M.MAC)

	adapter.SetTargetName(target_name)
	adapter.Flush()
	return true
}
func oneloop() {
	runCnt++
	adapter.StopScan()

	err := adapter.ScanPlus(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		//defer measureTime("SCAN")()//一旦开启 LOG很多
		//start := time.Now()
		if !result.AdvertisementPayload.HasServiceUUID(serviceUUID) {
			log.Printf("Failed UUID return\r\n")
			return
		}
		/*加强停止Scan函数的限制条件*/
		/*额外 条件1--从机必须有名字 而且符合约定*/
		log.Printf("Good Scanned MAC %s", result.Address.String())
		log.Printf("Scanned name %s MAC %s", result.LocalName(), result.Address.String())
		log.Printf("ScanedTarget %s", String_rm_char(result.Address.String(), ":"))

		foundDevice = result

		err := adapter.StopScan()
		if err != nil {
			log.Printf("Failed StopScan %v ", err.Error())
		}
	})

	if err != nil {
		log.Printf("Failed Scan %v ", err.Error())
		log.Printf("===========END=========(%d)\r\n", failCnt)
		failCnt = 0
		return
	}

	/*MUST 在外面才能成功！！！！*/
	d, e := adapter.Connect(foundDevice.Address, bluetooth.ConnectionParams{})
	if e != nil {
		log.Printf("[FailedConnect] %v", e)
		failCnt++
		if d != nil {
			log.Printf("[FailedConnect] %v", d.Disconnect())
		}
		return
	}

	log.Printf("Good ConnectedTarget (%v) %s ", d.IsConnected(), d.DevPath)

	device = d

	// Connected. Look up the Nordic UART Service.
	log.Printf("Discovering service...")
	services, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		println("Failed to discover the Nordic UART Service:", err.Error())
		log.Printf("===========END=========(%d)\r\n", failCnt)
		failCnt = 0
		return
	}
	service := services[0]

	// Get the two characteristics present in this service.
	chars, err := service.DiscoverCharacteristics([]bluetooth.UUID{rxUUID, txUUID})
	if err != nil {
		println("Failed to discover RX and TX characteristics:", err.Error())
		log.Printf("===========END=========(%d)\r\n", failCnt)
		failCnt = 0
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
	log.Printf("rx.UUID() %v\r\n", rx.UUID())
	log.Printf("TX %v\r\n", tx)
	//log.Printf("DiscoverCharacteristics:%+v\r\n", chars)
	/*
	   &等待从机断开
	*/

	_, err = tx.EnableNotifications(func(value []byte) {
		log.Printf("PI recv %d bytes: %X\r\n", len(value), value)
		rxdatacount++
		rxdatanumber = rxdatanumber + int64(len(value))
	})
	if err != nil {
		log.Printf("Failed EnableNotifications %+v\r\n", err.Error())
		log.Printf("===========END=========(%d)\r\n", failCnt)
		failCnt = 0
		return
	}
	log.Printf("Connected.When NODE disconnect.This pid while Exit\r\n")
	/*等待从机断开 PI从不发消息*/
	for {
		if !device.IsConnected() {
			yesCnt++
			log.Printf("TotalCount=%d,yes=%d,fail=%d,rxdatacount=%d,rxdatanumber=%d", runCnt, yesCnt, failCnt, rxdatacount, rxdatanumber)
			log.Printf("===========END=========device GoodBye\r\n")
			rxdatacount = 0
			rxdatanumber = 0
			return
		}
		time.Sleep(time.Microsecond * 100)
	}

	/*
	   & PI主动断开

	*/
	/*
	   	//Enable notifications to receive incoming data.
	   	count := 0
	   LOOP:
	   	cccd, err := tx.EnableNotifications(func(value []byte) {
	   		//log.Printf("PI recv %d bytes: %X\r\n", len(value), value)
	   		log.Printf("PI recv %d \r\n", len(value))
	   		rxdatacount++
	   		rxdatanumber = rxdatanumber + int64(len(value))
	   	})
	   	if err != nil {
	   		log.Printf("Failed EnableNotifications %+v\r\n", err.Error())
	   		log.Printf("===========END=========(%d)\r\n", failCnt)
	   		failCnt = 0
	   		return
	   	} else {
	   		log.Printf("EnableNotifications OK\r\n")
	   		time.Sleep(time.Second)
	   		log.Printf("DisableNotifications %v\r\n", tx.DisableNotifications(cccd))
	   		time.Sleep(time.Second)
	   		count++
	   		if (count) == 4 {

	   			goto NEXT
	   		}
	   		goto LOOP
	   	}
	   NEXT:
	   	log.Printf("++++++++++++++++++++++++\r\n")
	   	//主动断开
	   	log.Printf("device.Disconnect() [%v]\r\n", device.Disconnect())
	   	//MUKA代码可以不冲洗 TingGo必须冲洗
	   	//如果不要下面这行 可以表演特技!程序会卡在这里 新建一个ssh执行./flush 1 这个程序就活了！
	   	//./flush的来源的单独编译那个冲洗树的程序
	   	//log.Printf("adapter.Flush() [%v]\r\n", adapter.Flush())
	   	//adapter.FlushOne(foundDevice.Address.String())
	   	time.Sleep(time.Second)
	*/
}

func main() {
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)
	if !hciinit() {
		return
	}

	for KK := 0; KK < 5; KK++ {
		oneloop()
	}
}
