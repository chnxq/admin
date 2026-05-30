package bootstrap

// Add GeneratedData/GeneratedServices bootstrap hooks here.
// This file is created once and is never overwritten by xkit.

func (data *GeneratedData) afterInit() {
	if data == nil {
		return
	}
	data.WrapAuditLogRepos()
}

func (services *GeneratedServices) afterInit(data *GeneratedData) {
	_, _ = services, data
}
