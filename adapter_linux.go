//go:build !baremetal
// +build !baremetal

// Some documentation for the BlueZ D-Bus interface:
// https://git.kernel.org/pub/scm/bluetooth/bluez.git/tree/doc

package bluetooth

import (
	"errors"
	"fmt"

	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/hw/linux"
)

type Adapter struct {
	adapter              *adapter.Adapter1
	id                   string
	Mac                  string
	TargetName           string
	cancelChan           chan struct{}
	defaultAdvertisement *Advertisement

	connectHandler func(device Addresser, connected bool)
}

// DefaultAdapter is the default adapter on the system. On Linux, it is the
// first adapter available.
//
// Make sure to call Enable() before using it to initialize the adapter.
var DefaultAdapter = &Adapter{
	TargetName: "NO",
	connectHandler: func(device Addresser, connected bool) {
		return
	},
}
var HCI1Adapter = &Adapter{
	id:         "hci1",
	TargetName: "NO",
	connectHandler: func(device Addresser, connected bool) {
		return
	},
}

// Enable configures the BLE stack. It must be called before any
// Bluetooth-related calls (unless otherwise indicated).
func (a *Adapter) Enable() (err error) {
	if a.id == "" {
		a.adapter, err = api.GetDefaultAdapter()
		if err != nil {
			return err
		}
		a.id, err = a.adapter.GetAdapterID()
		a.Mac = a.adapter.Properties.Address
	} else { //说明a.id已经前置赋值
		a.adapter, err = api.GetAdapter(a.id)
		if err != nil {
			return err
		}
		a.Mac = a.adapter.Properties.Address
	}
	return nil
}

func (a *Adapter) SetHciId(id string) {
	a.id = id
}

func (a *Adapter) SetTargetName(name string) {
	a.TargetName = name
}
func (a *Adapter) Hello() {
	fmt.Println("HELLO")
}

func (a *Adapter) Address() (MACAddress, error) {
	if a.adapter == nil {
		return MACAddress{}, errors.New("adapter not enabled")
	}
	fmt.Println("a.adapter.Properties.Address", a.adapter.Properties.Address)
	mac, err := ParseMAC(a.adapter.Properties.Address)
	if err != nil {
		return MACAddress{}, err
	}
	fmt.Println("Address", mac)
	return MACAddress{MAC: mac}, nil
}

func (a *Adapter) Enable2(hcix string) (err error) {
	adapter.SetDefaultAdapterID(hcix) //仅仅增加一句话
	if a.id == "" {
		a.adapter, err = api.GetDefaultAdapter()
		if err != nil {
			return
		}
		a.id, err = a.adapter.GetAdapterID()
	}
	return nil
}

func (a *Adapter) Enable3(hcix string) (err error) {
	//adapter.SetDefaultAdapterID(hcix)
	if a.id == "" {
		a.adapter, err = api.GetAdapter(hcix) //GetDefaultAdapter()
		if err != nil {
			return
		}
		a.id, err = a.adapter.GetAdapterID()
	}
	return nil
}

//调用大哥的方法 优雅复位HCI
func (a *Adapter) Reset() (err error) {
	return linux.Reset(a.id)
}

//全部冲洗 树干净 所以的连接的 都冲洗走
func (a *Adapter) Flush() (err error) {
	devices, err := a.adapter.GetDevices()
	if err != nil {
		return err
	}
	fmt.Println(len(devices))
	fmt.Println(devices[0])

	for i, dev := range devices {
		fmt.Println(i, dev.Path())
		err = a.adapter.RemoveDevice(dev.Path())
		if err != nil {
			return fmt.Errorf("FlushDevices.RemoveDevice %s: %s", dev.Path(), err)
		}
	}
	fmt.Println("Flushed")
	return nil
}

//传入MAC地址 AA:BB:BB:BB:BB:BB 将其冲洗掉
func (a *Adapter) FlushOne(address string) (err error) {
	device, err := a.adapter.GetDeviceByAddress(address)
	if err != nil {
		return err
	}

	err = a.adapter.RemoveDevice(device.Path())
	if err != nil {
		return fmt.Errorf("FlushDevices.RemoveDevice %s: %s", device.Path(), err)
	}

	fmt.Println("FlushOne ", device.Path())
	return nil
}
