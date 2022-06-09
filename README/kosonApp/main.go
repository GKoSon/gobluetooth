package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	bluetooth "github.com/GKoSon/gobluetooth"
)

var (
	serviceUUID = bluetooth.ServiceUUIDNordicUART
	rxUUID      = bluetooth.CharacteristicUUIDUARTRX
	txUUID      = bluetooth.CharacteristicUUIDUARTTX
)

var adapter = bluetooth.DefaultAdapter

var appDevMap map[string]bool
var lock sync.Mutex

const target_name = "M_SHANGHAI" //"M_IZAR_ESP_TEST"

//const target_name = "M_IZAR_TEST"

func String_rm_char(a string, b string) string {
	mac := ""
	str := strings.Split(a, b)
	for _, s := range str {
		mac += s
	}
	return mac
}

//https://blog.csdn.net/weixin_44908159/article/details/123609779
func mapdel(mac string) {
	lock.Lock()
	delete(appDevMap, mac)
	lock.Unlock()
	log.Printf("mapdel[%s]\r\n", mac)
}

func mapadd(mac string) {
	lock.Lock()
	appDevMap[mac] = true
	lock.Unlock()
	log.Printf("mapadd[%s]\r\n", mac)
}
func app1(dev *bluetooth.Device) {
	mac := String_rm_char(dev.DevPath, ":")
	//defer delete(appDevMap, mac)
	defer mapdel(mac)
	mapadd(mac)

	log.Printf("[%s]Discovering service...\r\n", mac)
	services, err := dev.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		log.Println(mac, "Failed to discover the Nordic UART Service:", err.Error())
		return
	}

	log.Printf("[%s]Discovering Characteristics...\r\n", mac)
	service := services[0]
	chars, err := service.DiscoverCharacteristics([]bluetooth.UUID{rxUUID, txUUID})
	if err != nil {
		log.Println(mac, "Failed to discover RX and TX characteristics:", err.Error())
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
	log.Printf("[%s]RX %v\r\n", mac, rx)
	//log.Printf("rx.UUID() %v\r\n", rx.UUID())

	count := 0
LOOP:
	cccd, err := tx.EnableNotifications(func(value []byte) {
		//log.Printf("PI recv %d bytes: %X\r\n", len(value), value)
		log.Printf("[%s]PI recv %d \r\n", mac, len(value))
	})

	if err != nil {
		log.Printf("[%s]EnableNotifications Failed %+v\r\n", mac, err.Error())
		return
	} else {
		log.Printf("[%s]EnableNotifications OK\r\n", mac)
		time.Sleep(time.Second)
		log.Printf("[%s]DisableNotifications %v\r\n", mac, tx.DisableNotifications(cccd))
		time.Sleep(time.Second)
		count++
		if (count) == 8 {
			goto NEXT
		}
		goto LOOP
	}
NEXT:

	//主动断开
	log.Printf("[%s]Disconnected device...\r\n", mac)
	go dev.Disconnect()
	//err = dev.Disconnect()
	//if err != nil {
	//	log.Printf("[%s]Disconnected Failed %+v\r\n", mac, err.Error())
	//	return
	//}
	//time.Sleep(time.Second)

	//log.Printf("[%s][%v]main remove device...\r\n", mac, dev.IsConnected()) //100%false
	//adapter.FlushOne(dev.DevPath)

	log.Printf("[%s]done...\r\n", mac)
	return
}

func app2(dev *bluetooth.Device) {
	mac := String_rm_char(dev.DevPath, ":")
	defer mapdel(mac)
	mapadd(mac)

	log.Printf("[%s]Discovering service...\r\n", mac)
	services, err := dev.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		log.Println(mac, "Failed to discover the Nordic UART Service:", err.Error())
		return
	}

	log.Printf("[%s]Discovering Characteristics...\r\n", mac)
	service := services[0]
	chars, err := service.DiscoverCharacteristics([]bluetooth.UUID{rxUUID, txUUID})
	if err != nil {
		log.Println(mac, "Failed to discover RX and TX characteristics:", err.Error())
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
	log.Printf("[%s]RX %v\r\n", mac, rx)

	_, err = tx.EnableNotifications(func(value []byte) {
		//log.Printf("[%s]PI recv %d \r\n", mac, len(value))
	})

	if err != nil {
		log.Printf("[%s]EnableNotifications Failed %+v\r\n", mac, err.Error())
		return
	}

	for {
		time.Sleep(time.Microsecond * 10)
		if !dev.IsConnected() {
			log.Printf("[%s]Disconnected device...\r\n", mac)
			return
		}
	}

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
	//log.Printf("useing[%v]", M.isRandom)//小写无法打印 用61行办法
	for i := 0; i < 6; i++ {
		log.Printf("0X%02X ", M.MAC[i])
	}

	return true
}
func oneloop() {
	var device *bluetooth.Device
	err := adapter.ScanPlus(
		map[string]interface{}{
			"Transport": "le",
			"UUIDs":     []string{serviceUUID.String()},
			"Pattern":   target_name,
		},

		func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			log.Printf("result.Address.String()--MUKA--%s\r\n", result.Address.String())
			device, _ = adapter.MUKAGetDeviceByAddress(result.Address.String()) //反向查找能力
			log.Printf("ScanPlus will break dev:%#v\r\n", device)
			adapter.StopScan()
			//appDevMap[String_rm_char(result.Address.String(), ":")] = true
			//mapadd(String_rm_char(result.Address.String(), ":"))
			//不能放在这里 需要完全对应del/add
		})

	if err != nil {
		log.Printf("Failed ScanPlus %v", err.Error())
		//adapter.Reset()
		//log.Printf("Failed ScanPlus Help [%v]\r\n", adapter.Reset())//没效果
		adapter.StopScan()
		return
	}
	/*******************************************************/
	if device == nil {
		log.Printf("Strange device is nil\r\n")
		return
	}
	go app1(device)
}

func isCanceled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func main() {
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)
	if !hciinit() {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	ResetBle()
	log.Printf("HELLO APP->:adapter.Reset() [%v]\r\n", adapter.Reset())
	log.Printf("HELLO APP->:adapter.Flush() [%v]\r\n", adapter.Flush())

	appDevMap = make(map[string]bool)

	go func() {
		diecount := 0
		for {
			time.Sleep(time.Second * 20)
			lock.Lock()
			alivedev := len(appDevMap)
			log.Printf("check appDevMap[%d] %#v\r\n", alivedev, appDevMap)
			lock.Unlock()
			if alivedev == 100 {
				diecount++
				log.Printf("check APP->help cmd\r\n")
				log.Printf("check APP->:adapter.Reset() [%v]\r\n", adapter.Reset()) //MUST前面 后面可能冲洗卡住
				log.Printf("check APP->:adapter.Flush() [%v]\r\n", adapter.Flush())
				ResetBle()
				cancel()
				ctx, cancel = context.WithCancel(context.Background())
				go func(ctx context.Context) {
					for {
						if isCanceled(ctx) {
							break
						}
						log.Printf("check MAIN APP[%d]->:oneloop\r\n", diecount)
						oneloop()
					}

				}(ctx)
			}
		}
	}()

	go func(ctx context.Context) {
		for {
			if isCanceled(ctx) {
				break
			}
			log.Printf("MAIN APP->:oneloop")
			oneloop()
		}

	}(ctx)

	for {
	}

}

func ResetBle() {

	cmd := exec.Command("/etc/init.d/bluetooth", "restart")
	stdout, err := cmd.Output()
	if err != nil {
		log.Printf("[ResetBle]exec.Command fail %v\r\n", err)
	} else {
		log.Printf("[ResetBle]exec.Command ok %s\r\n", stdout)
	}

}
