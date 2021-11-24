//go:build windows

package runcat

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/lxn/walk"
)

var channel = make(chan *walk.Icon)

func init() {
	go loop()
}

func readTheme() string {
	k, err := registry.OpenKey(registry.CURRENT_USER, `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return "light_cat"
	}
	defer k.Close()

	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return "light_cat"
	}

	fmt.Println(v)

	if v != 0 {
		return "light_cat"
	}

	return "dark_cat"
}

func loadIcons(theme string) (icons [5]*walk.Icon, err error) {
	for i, v := range []string{
		"_0.ico",
		"_1.ico",
		"_2.ico",
		"_3.ico",
		"_4.ico",
	} {
		icons[i], err = walk.NewIconFromResource("$" + theme + v)
		if err != nil {
			return
		}
	}
	return
}

func loop() {
	// read theme setting
	theme := readTheme()

	// load icons
	icons, err := loadIcons(theme)
	if err != nil {
		log.Panic(err)
	}

	// get CPU time
	type TimeSet struct {
		Idle   windows.Filetime
		Kernel windows.Filetime
		User   windows.Filetime
	}
	var one, two TimeSet
	if err := GetsystemTimes(&one.Idle, &one.Kernel, &one.User); err != nil {
		log.Panic(err)
	}

	// send first icon
	channel <- icons[0]
	index := 1

	for {
		// get next CPU time
		if err := GetsystemTimes(&two.Idle, &two.Kernel, &two.User); err != nil {
			log.Panic(err)
		}

		if t := readTheme(); theme != t {
			theme = t

			var err error
			icons, err = loadIcons(theme)
			if err != nil {
				log.Printf("change theme error: %s\n", err)
			}
		}

		// sleep due to CPU usage
		time.Sleep(func() time.Duration {
			const Min = 10
			const Max = 500
			const Range = Max - Min

			idle := int64(two.Idle.LowDateTime) | int64(two.Idle.HighDateTime)<<32 - int64(one.Idle.LowDateTime) | int64(one.Idle.HighDateTime)<<32
			kernel := int64(two.Kernel.LowDateTime) | int64(two.Kernel.HighDateTime)<<32 - int64(one.Kernel.LowDateTime) | int64(one.Kernel.HighDateTime)<<32
			user := int64(two.User.LowDateTime) | int64(two.User.HighDateTime)<<32 - int64(one.User.LowDateTime) | int64(one.User.HighDateTime)<<32
			percentage := float64(idle) / float64(kernel+user)

			return time.Duration(Min+int64(Range*percentage*percentage)) * time.Millisecond
		}())

		// send icon
		channel <- icons[index]

		// update setting
		one = two
		if index == 4 {
			index = 0
		} else {
			index += 1
		}
	}
}

// GetNextIcon is ...
func GetNextIcon() *walk.Icon {
	return <-channel
}

// Get CPU usage ...
var (
	kernel32 = windows.MustLoadDLL("kernel32.dll")

	getSystemTimes = kernel32.MustFindProc("GetSystemTimes")
)

// GetSystemTimes is ...
func GetsystemTimes(idle, kernel, user *windows.Filetime) error {
	ret, _, err := getSystemTimes.Call(uintptr(unsafe.Pointer(idle)), uintptr(unsafe.Pointer(kernel)), uintptr(unsafe.Pointer(user)))
	if ret == 0 {
		return err
	}
	return nil
}
