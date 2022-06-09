//go:build !baremetal
// +build !baremetal

package bluetooth

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/advertising"
	"github.com/muka/go-bluetooth/bluez/profile/device"
)

// Address contains a Bluetooth MAC address.
type Address struct {
	MACAddress
}

// Advertisement encapsulates a single advertisement instance.
type Advertisement struct {
	adapter       *Adapter
	advertisement *api.Advertisement
	properties    *advertising.LEAdvertisement1Properties
}

// DefaultAdvertisement returns the default advertisement instance but does not
// configure it.
func (a *Adapter) DefaultAdvertisement() *Advertisement {
	if a.defaultAdvertisement == nil {
		a.defaultAdvertisement = &Advertisement{
			adapter: a,
		}
	}
	return a.defaultAdvertisement
}

// Configure this advertisement.
//
// On Linux with BlueZ, it is not possible to set the advertisement interval.
func (a *Advertisement) Configure(options AdvertisementOptions) error {
	if a.advertisement != nil {
		panic("todo: configure advertisement a second time")
	}

	a.properties = &advertising.LEAdvertisement1Properties{
		Type:      advertising.AdvertisementTypeBroadcast,
		Timeout:   1<<16 - 1,
		LocalName: options.LocalName,
	}
	for _, uuid := range options.ServiceUUIDs {
		a.properties.ServiceUUIDs = append(a.properties.ServiceUUIDs, uuid.String())
	}

	return nil
}

// Start advertisement. May only be called after it has been configured.
func (a *Advertisement) Start() error {
	if a.advertisement != nil {
		panic("todo: start advertisement a second time")
	}
	_, err := api.ExposeAdvertisement(a.adapter.id, a.properties, uint32(a.properties.Timeout))
	if err != nil {
		return err
	}
	return nil
}

// Scan starts a BLE scan. It is stopped by a call to StopScan. A common pattern
// is to cancel the scan when a particular device has been found.
//
// On Linux with BlueZ, incoming packets cannot be observed directly. Instead,
// existing devices are watched for property changes. This closely simulates the
// behavior as if the actual packets were observed, but it has flaws: it is
// possible some events are missed and perhaps even possible that some events
// are duplicated.
func (a *Adapter) Scan(filter map[string]interface{}, callback func(*Adapter, ScanResult)) error {
	if a.cancelChan != nil {
		return errScanning
	}

	// Channel that will be closed when the scan is stopped.
	// Detecting whether the scan is stopped can be done by doing a non-blocking
	// read from it. If it succeeds, the scan is stopped.
	cancelChan := make(chan struct{})
	a.cancelChan = cancelChan

	// This appears to be necessary to receive any BLE discovery results at all.
	if filter != nil {
		defer a.adapter.SetDiscoveryFilter(nil)
		err := a.adapter.SetDiscoveryFilter(filter)
		if err != nil {
			return err
		}
	}

	bus, err := dbus.SystemBus()
	if err != nil {
		return err
	}

	signal := make(chan *dbus.Signal)
	bus.Signal(signal)
	defer bus.RemoveSignal(signal)

	propertiesChangedMatchOptions := []dbus.MatchOption{dbus.WithMatchInterface("org.freedesktop.DBus.Properties")}
	bus.AddMatchSignal(propertiesChangedMatchOptions...)
	defer bus.RemoveMatchSignal(propertiesChangedMatchOptions...)

	newObjectMatchOptions := []dbus.MatchOption{dbus.WithMatchInterface("org.freedesktop.DBus.ObjectManager")}
	bus.AddMatchSignal(newObjectMatchOptions...)
	defer bus.RemoveMatchSignal(newObjectMatchOptions...)

	// Go through all connected devices and present the connected devices as
	// scan results. Also save the properties so that the full list of
	// properties is known on a PropertiesChanged signal. We can't present the
	// list of cached devices as scan results as devices may be cached for a
	// long time, long after they have moved out of range.
	deviceList, err := a.adapter.GetDevices()
	if err != nil {
		return err
	}
	devices := make(map[dbus.ObjectPath]*device.Device1Properties)
	for _, dev := range deviceList {
		if dev.Properties.Connected {
			callback(a, makeScanResult(dev.Properties))
			select {
			case <-cancelChan:
				return nil
			default:
			}
		}
		devices[dev.Path()] = dev.Properties
	}

	// Instruct BlueZ to start discovering.
	err = a.adapter.StartDiscovery()
	if err != nil {
		return err
	}

	for {
		// Check whether the scan is stopped. This is necessary to avoid a race
		// condition between the signal channel and the cancelScan channel when
		// the callback calls StopScan() (no new callbacks may be called after
		// StopScan is called).
		select {
		case <-cancelChan:
			a.adapter.StopDiscovery()
			return nil
		default:
		}

		select {
		case sig := <-signal:
			// This channel receives anything that we watch for, so we'll have
			// to check for signals that are relevant to us.
			switch sig.Name {
			case "org.freedesktop.DBus.ObjectManager.InterfacesAdded":
				objectPath := sig.Body[0].(dbus.ObjectPath)
				interfaces := sig.Body[1].(map[string]map[string]dbus.Variant)
				rawprops, ok := interfaces["org.bluez.Device1"]
				if !ok {
					continue
				}
				var props *device.Device1Properties
				props, _ = props.FromDBusMap(rawprops)
				devices[objectPath] = props
				//log.Printf("Scan InterfacesAdded : %v\r\n", makeScanResult(props).Address)
				callback(a, makeScanResult(props))
			case "org.freedesktop.DBus.Properties.PropertiesChanged":
				interfaceName := sig.Body[0].(string)
				if interfaceName != "org.bluez.Device1" {
					continue
				}
				changes := sig.Body[1].(map[string]dbus.Variant)
				props := devices[sig.Path]
				for field, val := range changes {
					switch field {
					case "RSSI":
						props.RSSI = val.Value().(int16)
					case "Name":
						props.Name = val.Value().(string)
					case "UUIDs":
						props.UUIDs = val.Value().([]string)
					}
				}
				//log.Printf("Scan PropertiesChanged : %v\r\n", makeScanResult(props).Address)
				callback(a, makeScanResult(props))

			}
		case <-cancelChan:
			continue
		}
	}

	// unreachable
}

var discoverying bool = false
var Chdiscovery = make(chan bool)

func (a *Adapter) resetdiscoverying() {
	log.Printf("TingGo discoverying set false\r\n")
	discoverying = false
}

//永远在后台执行 永不退出
//往这个通道发消息 6S以后就会开始发现
func (a *Adapter) DelayDiscovery(start chan bool) {
	DurationOfTime := time.Duration(6) * time.Second
	f := func() {
		if !discoverying {
			log.Printf("TinyGo DelayDiscovery\r\n")
			a.onDiscovery()
		}
	}
	for {
		select {
		case <-start:
			time.AfterFunc(DurationOfTime, f)
		}
	}
}

//放弃使用 过于暴力
func (a *Adapter) delayDiscovery() {
	for {
		time.Sleep(time.Second * 9)
		if !discoverying {
			log.Printf("TinyGo delayDiscovery\r\n")
			a.onDiscovery()
		}
	}
}

func (a *Adapter) onDiscovery() error {
	if discoverying {
		log.Println("TinyGo StartDiscovery NULL")
		return nil
	}

	err := a.adapter.StartDiscovery()
	if err != nil {
		log.Println("TinyGo StartDiscovery", err)
		return err
	}
	discoverying = true
	return nil
}

func (a *Adapter) offDiscovery() error {
	if !discoverying {
		log.Println("TinyGo StopDiscovery NULL")
		return nil
	}
	err := a.adapter.StopDiscovery()
	if err != nil {
		log.Println("TinyGo StopDiscovery", err)
		return err
	}
	discoverying = false
	return nil
}

var setOnce bool = true

func (a *Adapter) ScanPlus(filter map[string]interface{}, callback func(*Adapter, ScanResult)) error {

	if a.cancelChan != nil {
		return errScanning
	}

	cancelChan := make(chan struct{})
	a.cancelChan = cancelChan

	thisdevice := make(map[dbus.ObjectPath]*device.Device1Properties)

	if setOnce {
		err := a.adapter.SetDiscoveryFilter(filter)
		if err != nil {
			return err
		}
		setOnce = false
		//go a.delayDiscovery()
		go a.DelayDiscovery(Chdiscovery)
	}

	bus, err := dbus.SystemBus()
	if err != nil {
		return err
	}

	signal := make(chan *dbus.Signal)
	bus.Signal(signal)
	defer bus.RemoveSignal(signal)

	propertiesChangedMatchOptions := []dbus.MatchOption{dbus.WithMatchInterface("org.freedesktop.DBus.Properties")}
	bus.AddMatchSignal(propertiesChangedMatchOptions...)
	defer bus.RemoveMatchSignal(propertiesChangedMatchOptions...)

	newObjectMatchOptions := []dbus.MatchOption{dbus.WithMatchInterface("org.freedesktop.DBus.ObjectManager")}
	bus.AddMatchSignal(newObjectMatchOptions...)
	defer bus.RemoveMatchSignal(newObjectMatchOptions...)

	///////////////////start
	deviceList, err := a.adapter.GetDevices()
	if err != nil {
		log.Printf("TingGo income die\r\n")
		return err
	}

	for k, dev := range deviceList {
		if dev.Properties.Connected {
			log.Printf("TingGo income %d-%s\r\n", k, dev.Properties.Address)
		} else {
			a.adapter.RemoveDevice(dev.Path()) //func (a *Adapter1) FlushDevices()
			log.Printf("TingGo remove %d-%s\r\n", k, dev.Properties.Address)
		}
		thisdevice[dev.Path()] = dev.Properties
	}

	///////////////////end

	err = a.onDiscovery()
	if err != nil {
		return err
	}

	for {

		select {
		case <-cancelChan:
			a.offDiscovery()
			log.Printf("TingGo goodbye\r\n")
			return nil

		case sig := <-signal:
			switch sig.Name {
			case "org.freedesktop.DBus.ObjectManager.InterfacesRemoved": //DEL
				//objectPath := sig.Body[0].(dbus.ObjectPath)
				//log.Printf("TingGo DEL Node[%s]\r\n", objectPath)

			case "org.freedesktop.DBus.ObjectManager.InterfacesAdded":
				//log.Printf("TingGo NEW Node[%#v]\r\n", sig)
				objectPath := sig.Body[0].(dbus.ObjectPath)
				interfaces := sig.Body[1].(map[string]map[string]dbus.Variant)
				rawprops, ok := interfaces["org.bluez.Device1"]
				if !ok {
					continue
				}
				var props *device.Device1Properties //这就是巨大的结构体 有可能第一次就连接好吗?
				props, _ = props.FromDBusMap(rawprops)
				thisdevice[objectPath] = props

				//log.Printf("TingGo NEW Node props.Name [%s]\r\n", props.Name)
				log.Printf("TingGo NEW Node props.Address [%s]\r\n", props.Address)

				//go
				a.MUKAConnect(props.Address)

				break
			case "org.freedesktop.DBus.Properties.PropertiesChanged":
				interfaceName := sig.Body[0].(string)
				if interfaceName != "org.bluez.Device1" {
					continue
				}

				changes := sig.Body[1].(map[string]dbus.Variant)
				props := thisdevice[sig.Path]
				if props == nil {
					log.Printf("TingGo CHG FUCK\r\n")
					break
				}
				for field, val := range changes {
					switch field {
					case "RSSI":
						props.RSSI = val.Value().(int16)
						log.Printf("TingGo CHG props.RSSI [%v]\r\n", props.RSSI)
						if !props.Connected {
							log.Printf("TingGo CHG this gay need connect [%s]\r\n", props.Address)
							//go
							a.MUKAConnect(props.Address)
						}
						break
					case "Name":
						props.Name = val.Value().(string)
						log.Printf("TingGo CHG props.Name [%v]\r\n", props.Name)
						break
					case "UUIDs":
						props.UUIDs = val.Value().([]string)
						log.Printf("TingGo CHG props.UUIDs [%v]\r\n", props.UUIDs)
						break
					case "Connected":
						props.Connected = val.Value().(bool)
						if props.Connected == true {
							log.Printf("TingGo CHG Connected [%s]\r\n", props.Address)
						} else if props.Connected == false {
							log.Printf("TingGo CHG DisConnected [%s]\r\n", props.Address)
						}
						break
					case "ServicesResolved":
						props.ServicesResolved = val.Value().(bool)
						if props.ServicesResolved == true {
							log.Printf("TingGo CHG ServicesResolved [%s]\r\n", props.Address)
							callback(a, makeScanResult(props))
						} else if props.ServicesResolved == false {
							log.Printf("TingGo CHG DisServicesResolved [%s]\r\n", props.Address)
						}
						break
					}
				}
				break
			} //switch
		} //select
	} //for
}

// StopScan stops any in-progress scan. It can be called from within a Scan
// callback to stop the current scan. If no scan is in progress, an error will
// be returned.
func (a *Adapter) StopScan() error {
	if a.cancelChan == nil {
		return errNotScanning
	}
	close(a.cancelChan)

	a.cancelChan = nil
	return nil
}

// makeScanResult creates a ScanResult from a Device1 object.
func makeScanResult(props *device.Device1Properties) ScanResult {
	// Assume the Address property is well-formed.
	addr, _ := ParseMAC(props.Address)

	// Create a list of UUIDs.
	var serviceUUIDs []UUID
	for _, uuid := range props.UUIDs {
		// Assume the UUID is well-formed.
		parsedUUID, _ := ParseUUID(uuid)
		serviceUUIDs = append(serviceUUIDs, parsedUUID)
	}

	a := Address{MACAddress{MAC: addr}}
	a.SetRandom(props.AddressType == "random")

	return ScanResult{
		RSSI:        props.RSSI,
		Address:     a,
		MUKAAddress: props.Address,
		AdvertisementPayload: &advertisementFields{
			AdvertisementFields{
				LocalName:    props.Name,
				ServiceUUIDs: serviceUUIDs,
			},
		},
	}
}

//全部冲洗 树干净 所以的连接的 都冲洗走
func (a *Adapter) Flush() (err error) {
	defer a.resetdiscoverying()
	devices, err := a.adapter.GetDevices()
	if err != nil {
		return err
	}

	if 0 == len(devices) {
		log.Printf("TingGo Tree is zero ---do nothing--- busctl tree org.bluez\r\n")
		return nil
	}
	for i, dev := range devices {
		log.Println("TingGo FlushDevices", i, dev.Path(), dev.Properties.Connected)
		//if !dev.Properties.Connected {
		err = a.adapter.RemoveDevice(dev.Path())
		//fmt.Println("REMOVE", dev.Path())
		//}
		//err = a.adapter.RemoveDevice(dev.Path())
		if err != nil {
			log.Println("TingGo FlushDevices Fail", i, dev.Path())
			return err
		}
	}
	log.Printf("TingGo Flushed done -busctl tree org.bluez\r\n")

	return nil

}

//传入MAC地址 AA:BB:BB:BB:BB:BB 将其冲洗掉
func (a *Adapter) FlushOne(address string) (err error) {
	device, err := a.adapter.GetDeviceByAddress(address)
	if err != nil {
		return err
	}
	if device == nil {
		log.Printf("TingGo FlushOne Null\r\n")
		return nil
	}

	err = a.adapter.RemoveDevice(device.Path())
	if err != nil {
		log.Printf("TingGo FlushOne %s fail %v\r\n", device.Path(), err)
		return err
	}
	device.Close() //测试一下
	log.Printf("TingGo FlushOne OK %s\r\n", device.Path())
	return nil
}

// Device is a connection to a remote peripheral.
type Device struct {
	device  *device.Device1
	DevPath string
}

//做一个表 每次进去的时候就登记 出来的时候就移除 下次进去的时候看看表里面在做啥 如果在工作就不要进去了
var devMap map[string]bool = make(map[string]bool)
var lock sync.Mutex

func mapdel(mac string) {
	lock.Lock()
	devMap[mac] = false
	lock.Unlock()
	log.Printf("TingGo MUKAConnect mapdel[%s]\r\n", mac)
}

func mapadd(mac string) {
	lock.Lock()
	devMap[mac] = true
	lock.Unlock()
	log.Printf("TingGo MUKAConnect mapadd[%s]\r\n", mac)
}

//MUKAConnect Connect ERR:Operation already in progress
func (a *Adapter) MUKAConnect(address string) *device.Device1 {

	value, ok := devMap[address] //ok为true则存在，ok为false则map的key不存在
	if ok && value {
		log.Printf("TingGo MUKACon this gay is working %s\r\n", address)
		log.Printf("TingGo devMap[%d] %#v\r\n", len(devMap), devMap)
		return nil
	}
	mapadd(address)
	defer mapdel(address)

	log.Printf("TingGo ==>Connect==>start %s\r\n", address)
	a.offDiscovery()
	//defer a.onDiscovery()
	Chdiscovery <- true
	dev, err := a.adapter.GetDeviceByAddress(address)
	if err != nil {
		log.Printf("TingGo MUKAConnect GetDeviceByAddress ERR1 %v\r\n", err)
		log.Printf("TingGo ==>Connect==>end1 %s\r\n", address)
		return nil
	}
	if dev == nil {
		log.Printf("TingGo MUKAConnect GetDeviceByAddress ERR2 %s\r\n", address)
		log.Printf("TingGo ==>Connect==>end2 %s\r\n", address)
		return nil
	}

	err = dev.SetTrusted(true)
	if err != nil {
		log.Printf("TingGo MUKAConnect SetTrusted ERR:%v\r\n", err)
		log.Printf("TingGo ==>Connect==>end3 %s\r\n", address)
		return nil
	}
	retry := 0
LOOP:
	err = dev.Connect()
	if err != nil {
		log.Printf("TingGo MUKAConnect Connect ERR:%v\r\n", err)
		log.Printf("TingGo==>Connect==>end4 %s\r\n", address)
		retry++
		if retry > 2 {
			return nil
		} else {
			goto LOOP
		}
	}
	log.Printf("TingGo==>Connect==>end5 %s\r\n", address)
	log.Printf("TingGo MUKAConnect Connect OK %s\r\n", address)
	return dev
}

func String_rm_char(a string, b string) string {
	mac := ""
	str := strings.Split(a, b)
	for _, s := range str {
		mac += s
	}
	return mac
}

// Connect starts a connection attempt to the given peripheral device address.
//
// On Linux and Windows, the IsRandom part of the address is ignored.
func (a *Adapter) Connect(address Addresser, params ConnectionParams) (*Device, error) {
	adr := address.(Address)
	path := string(a.adapter.Path()) + "/dev_" + strings.Replace(adr.MAC.String(), ":", "_", -1)
	devicePath := dbus.ObjectPath(path)
	dev, err := device.NewDevice1(devicePath) //device来自MUKA包//MUKA自己也封装了一个类似函数
	if err != nil {
		return nil, err
	}

	log.Printf("TingGo==>dev.Properties.Connected=%v\r\n", dev.Properties.Connected)
	if !dev.Properties.Connected {
		// Not yet connected, so do it now.
		// The properties have just been read so this is fresh data.

		err := dev.Connect()
		log.Printf("TingGo==>dev.Properties.Connected==>dev.Connect()=%v\r\n", err)
		if err != nil {
			return nil, err
		}
	} else {
		log.Printf("TingGo==>dev.Properties.Connected==>do nothing\r\n")
	}
	// TODO: a proper async callback.
	a.connectHandler(nil, true)
	return &Device{
		device:  dev,
		DevPath: path,
	}, nil
}

func (a *Adapter) MUKAGetDeviceByAddress(address string) (*Device, error) {

	dev, err := a.adapter.GetDeviceByAddress(address)
	if err != nil {
		return nil, err
	}

	return &Device{
		device:  dev,
		DevPath: address,
	}, nil
}

// Disconnect from the BLE device. This method is non-blocking and does not
// wait until the connection is fully gone.
func (d *Device) Disconnect() error {
	//源码
	return d.device.Disconnect()
	//方式2 定位永不返回
	log.Printf("TingGo==>Disconnect==>start\r\n")
	d.device.Disconnect()
	log.Printf("TingGo==>Disconnect==>end\r\n")
	time.Sleep(time.Microsecond * 50)
	return nil
	//方式3
	//d.device.Close()
	//return nil
}

func (d *Device) IsConnected() bool {
	b, e := d.device.GetConnected()
	if e == nil {
		return b
	}
	return false
}
