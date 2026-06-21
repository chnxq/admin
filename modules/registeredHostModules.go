package modules

import "admin/shared/modulehost"

var modules = []modulehost.Module{
	//&xdev.Module{},
	// New Module palace here
}

func RegisteredHostModules() []modulehost.Module {
	return modules
}
