package osx

import (
	"runtime"

	"github.com/zcalusic/sysinfo"
)

func Vendor() string {
	var si sysinfo.SysInfo
	si.GetSysInfo()

	os := si.OS.Vendor
	if os == "" {
		os = runtime.GOOS
	}

	return os
}
