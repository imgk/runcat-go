//go:build windows

package runcat

import (
	"log"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

var channel = make(chan *walk.Icon)

func init() {
	go loop()
}

func readTheme() string {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		log.Printf("OpenKey error: %v\n", err)
		return "light_cat"
	}
	defer k.Close()

	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		log.Printf("GetIntegerValue error: %v\n", err)
		return "light_cat"
	}

	if v != 0 {
		return "light_cat"
	}

	return "dark_cat"
}

func loadIcons(theme string) (icons [5]*walk.Icon, err error) {
	for i, v := range []string{
		"_0_ico",
		"_1_ico",
		"_2_ico",
		"_3_ico",
		"_4_ico",
	} {
		icons[i], err = walk.NewIconFromResource(theme + v)
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

	// send first icon
	channel <- icons[0]
	index := 1

	handle := win.PDH_HQUERY(0)
	if errCode := win.PdhOpenQuery(0, 0, &handle); errCode != win.ERROR_SUCCESS {
		log.Printf("PdhOpenQuery error: %v\n", windows.Errno(errCode))
	}
	defer win.PdhCloseQuery(handle)

	counter := win.PDH_HCOUNTER(0)
	if errCode := win.PdhAddCounter(handle, `\Processor Information(_Total)\% Processor Time`, 0, &counter); errCode != win.ERROR_SUCCESS {
		log.Printf("PdhAddCounter error: %v\n", windows.Errno(errCode))
	}

	// if errCode := win.PdhCollectQueryData(handle); errCode != win.PDH_CSTATUS_INVALID_DATA {
	if errCode := win.PdhCollectQueryData(handle); errCode != win.ERROR_SUCCESS {
		log.Printf("PdhCollectQueryData error: %v\n", errCode)
	}

	time.Sleep(time.Second)

	if errCode := win.PdhCollectQueryData(handle); errCode != win.ERROR_SUCCESS {
		log.Printf("PdhCollectQueryData error: %v\n", errCode)
	}

	value := win.PDH_FMT_COUNTERVALUE_DOUBLE{}

	for {
		// sleep due to CPU usage
		time.Sleep(func() time.Duration {
			const Min = 10
			const Max = 500
			const Range = Max - Min

			if errCode := win.PdhGetFormattedCounterValueDouble(counter, nil, &value); errCode != win.ERROR_SUCCESS {
				log.Printf("PdhCollectQueryData error: %v\n", errCode)
			}

			per := (100 - value.DoubleValue) / 100
			log.Printf("CPU usage: %v\n", per)
			return time.Duration(Min+int64(Range*per*per*per)) * time.Millisecond
		}())

		if errCode := win.PdhCollectQueryData(handle); errCode != win.ERROR_SUCCESS {
			log.Printf("PdhCollectQueryData error: %v\n", errCode)
		}

		if t := readTheme(); theme != t {
			theme = t

			var err error
			icons, err = loadIcons(theme)
			if err != nil {
				log.Printf("change theme error: %v\n", err)
			}
		}

		// send icon
		channel <- icons[index]

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

