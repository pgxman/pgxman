package osx

import "github.com/zcalusic/sysinfo"

func Vendor() string {
	var si sysinfo.SysInfo
	si.GetSysInfo()

	return si.OS.Vendor
}
