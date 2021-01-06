package ch912x

import "errors"

var (
	ErrInvalidNetworkInterface = errors.New("ch912x: invalid network interface")
	ErrCH9120InvalidJSON       = errors.New("ch912x: the JSON not is CH9120 configuration")
	ErrCH9121InvalidJSON       = errors.New("ch912x: the JSON not is CH9121 configuration")
	ErrCH9126InvalidJSON       = errors.New("ch912x: the JSON not is CH9126 configuration")
	ErrModuleKindWrong         = errors.New("ch912x: the module kind wrong")
	ErrModuleMustMAC           = errors.New("ch912x: the need to provide `ModuleMAC`")
	ErrTaskRunning             = errors.New("ch912x: the previous task was not completed")
	ErrUnknownModuleType       = errors.New("ch912x: unknown module type")
)
