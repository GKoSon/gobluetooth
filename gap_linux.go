//go:build !baremetal
// +build !baremetal

package bluetooth

import (
	"log"
	"strings"
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

var setOnce bool = true

func (a *Adapter) ScanPlus(filter map[string]interface{}, callback func(*Adapter, ScanResult)) error {
	ch_koson_connect := make(chan bool)
	defer close(ch_koson_connect)
	if a.cancelChan != nil {
		return errScanning
	}

	cancelChan := make(chan struct{})
	a.cancelChan = cancelChan

	if setOnce {
		err := a.adapter.SetDiscoveryFilter(filter)
		if err != nil {
			return err
		}
		setOnce = false
		go a.KosonFlush()//定时清空
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

	thisdevice := make(map[dbus.ObjectPath]*device.Device1Properties)
	//start
	deviceList, err := a.adapter.GetDevices()
	if err != nil {
		return err
	}
	for _, dev := range deviceList {
		if dev.Properties.Connected {
			thisdevice[dev.Path()] = dev.Properties
			log.Printf("Node %s is TXRXING do not touch  %s", dev.Properties.Name, dev.Properties.Address)
			select {
			case <-cancelChan:
				return nil
			default:
			}
		} else {
			a.FlushOne(dev.Properties.Address)
		}

	}

	log.Printf("TingGo thisdevice %#v\r\n", thisdevice)

	//前置--解决连不上问题
	err = a.adapter.StartDiscovery()
	if err != nil {
		return err
	}
	seconddisconnect := false
	var NEW_CHG_SAME_PATH dbus.ObjectPath

	for {
		select {
		case <-cancelChan:
			a.adapter.StopDiscovery()
			return nil
		default:
		}

		select {
		case sig := <-signal:
			switch sig.Name {
			case "org.freedesktop.DBus.ObjectManager.InterfacesRemoved": //DEL
				//objectPath := sig.Body[0].(dbus.ObjectPath)
				//log.Printf("TingGo DEL Node[%s]\r\n", objectPath)

			case "org.freedesktop.DBus.ObjectManager.InterfacesAdded":
				log.Printf("TingGo NEW Node[%#v]\r\n", sig)
				objectPath := sig.Body[0].(dbus.ObjectPath)
				interfaces := sig.Body[1].(map[string]map[string]dbus.Variant)
				rawprops, ok := interfaces["org.bluez.Device1"]
				if !ok {
					continue
				}
				var props *device.Device1Properties //这就是巨大的结构体 有可能第一次就连接好吗?
				props, _ = props.FromDBusMap(rawprops)
				thisdevice[objectPath] = props

				log.Printf("TingGo NEW Node props.Name [%s]\r\n", props.Name)
				log.Printf("TingGo NEW Node props.Address [%s]\r\n", props.Address)
				a.adapter.StopDiscovery()
				NEW_CHG_SAME_PATH = objectPath
				//connecteddev := a.MUKAConnect(props.Address)
				//log.Printf("TingGo cmd MUKAConnect [%v]\r\n", connecteddev)
				go a.KosonConnect(thisdevice, objectPath, ch_koson_connect, callback)
				break
			case "org.freedesktop.DBus.Properties.PropertiesChanged":
				interfaceName := sig.Body[0].(string)
				if interfaceName != "org.bluez.Device1" {
					continue
				}
				if NEW_CHG_SAME_PATH != sig.Path { //关键
					log.Printf("TingGo car [%s][%s]\r\n", NEW_CHG_SAME_PATH, sig.Path)
					a.adapter.StartDiscovery()
					continue
				}
				changes := sig.Body[1].(map[string]dbus.Variant)
				props := thisdevice[sig.Path]
				if props == nil {
					break
				}
				for field, val := range changes {
					switch field {
					case "RSSI":
						props.RSSI = val.Value().(int16)
					case "Name":
						props.Name = val.Value().(string)
					case "UUIDs":
						props.UUIDs = val.Value().([]string)
					case "Connected":
						props.Connected = val.Value().(bool)
						if props.Connected == true {
							log.Printf("TingGo CHG Connected [%s]\r\n", props.Address)
							seconddisconnect = true
						} else if props.Connected == false {
							log.Printf("TingGo CHG DisConnected[%v] [%s]\r\n", seconddisconnect, props.Address)
							if seconddisconnect { //help 1--会过来 但是过了两次 秒断
								a.adapter.StartDiscovery()
								seconddisconnect = false
								a.FlushOne(props.Address)
								ch_koson_connect <- false
							}
						}
					case "ServicesResolved":
						props.ServicesResolved = val.Value().(bool)
						if props.ServicesResolved == true {
							log.Printf("TingGo CHG ServicesResolved [%s]\r\n", props.Address)
							ch_koson_connect <- true
						} else if props.ServicesResolved == false {
							log.Printf("TingGo CHG DisServicesResolved [%s]\r\n", props.Address)
						}
					}
				}
				/*
				   2022/06/06 14:17:45.232437 gap_linux.go:193: TingGo CHG Connected
				   2022/06/06 14:17:45.232485 gap_linux.go:210: TingGo CHG DisServicesResolved
				   2022/06/06 14:17:45.232520 gap_linux.go:196: TingGo CHG DisConnected

				*/
				break

			}
		case <-cancelChan:
			continue
		}
	}

	// unreachable
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

// Device is a connection to a remote peripheral.
type Device struct {
	device  *device.Device1
	DevPath string //debug
}

func (a *Adapter) MUKAConnect(address string) *device.Device1 {

	dev, err := a.adapter.GetDeviceByAddress(address)
	if err != nil {
		return nil
	}

	err = dev.Connect()
	if err != nil {
		return nil
	}
	return dev
}

/*
退出条件
return
1---尝试N次 最后放弃
2---发生秒断 监听到CHG信号 直接退出 消亡自己进程
3---连接成功 监听到CHG信号 回调用户空间函数 消亡自己

*/

const CONN_RETRY int = 3

func (a *Adapter) KosonConnect(devmap map[dbus.ObjectPath]*device.Device1Properties,
	onePath dbus.ObjectPath,
	dbusack chan bool,
	callback func(*Adapter, ScanResult)) {

	ticker_period := 3 * time.Second
	t := time.NewTicker(ticker_period)
	defer t.Stop()

	props := devmap[onePath]
	a.MUKAConnect(props.Address)

	retry := 0
	for {
		select {
		case yes := <-dbusack:
			if yes {
				callback(a, makeScanResult(props))
			}
			return
		case <-t.C:
			if retry > CONN_RETRY {
				log.Printf("KosonConnect giveup%d\r\n", retry)
				a.adapter.StartDiscovery()
				return //彻底放弃
			}
			a.MUKAConnect(props.Address)
			retry++
			log.Printf("KosonConnect retry %d\r\n", retry)
		}
	}
}

func (a *Adapter) KosonFlush() {
	ticker_period := 1 * time.Minute
	t := time.NewTicker(ticker_period)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			log.Printf("FK-KosonFlush========= [%v]\r\n", a.Flush())
		}
	}
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

	log.Printf("==>dev.Properties.Connected=%v\r\n", dev.Properties.Connected)
	if !dev.Properties.Connected {
		// Not yet connected, so do it now.
		// The properties have just been read so this is fresh data.

		err := dev.Connect()
		log.Printf("==>dev.Properties.Connected==>dev.Connect()=%v\r\n", err)
		if err != nil {
			return nil, err
		}
	} else {
		log.Printf("==>dev.Properties.Connected==>do nothing\r\n")
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
	return d.device.Disconnect()
}

func (d *Device) IsConnected() bool {
	b, e := d.device.GetConnected()
	if e == nil {
		return b
	}
	return false
}
