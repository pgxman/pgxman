package osx

import (
	"github.com/zcalusic/sysinfo"
)

func Sysinfo() sysinfo.SysInfo {
	var si sysinfo.SysInfo
	si.GetSysInfo()

	return si
}
