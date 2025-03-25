package merrs

import "github.com/spacemonkeygo/errors"

// 系统错误
var (
	NotImplementedError = pushErrorClass(errors.NotImplementedError)
	ProgrammerError     = pushErrorClass(errors.ProgrammerError)
	PanicError          = pushErrorClass(errors.PanicError)

	SyscallError        = pushErrorClass(errors.SyscallError)
	ErrnoError          = pushErrorClass(errors.ErrnoError)
	NetworkError        = pushErrorClass(errors.NetworkError)
	UnknownNetworkError = pushErrorClass(errors.UnknownNetworkError)
	AddrError           = pushErrorClass(errors.AddrError)
	InvalidAddrError    = pushErrorClass(errors.InvalidAddrError)
	NetOpError          = pushErrorClass(errors.NetOpError)
	NetParseError       = pushErrorClass(errors.NetParseError)
	DNSError            = pushErrorClass(errors.DNSError)
	DNSConfigError      = pushErrorClass(errors.DNSConfigError)
	IOError             = pushErrorClass(errors.IOError)
	EOF                 = pushErrorClass(errors.EOF)
	ClosedPipeError     = pushErrorClass(errors.ClosedPipeError)
	NoProgressError     = pushErrorClass(errors.NoProgressError)
	ShortBufferError    = pushErrorClass(errors.ShortBufferError)
	ShortWriteError     = pushErrorClass(errors.ShortWriteError)
	UnexpectedEOFError  = pushErrorClass(errors.UnexpectedEOFError)
	ContextError        = pushErrorClass(errors.ContextError)
	ContextCanceled     = pushErrorClass(errors.ContextCanceled)
	ContextTimeout      = pushErrorClass(errors.ContextTimeout)

	HierarchicalError = pushErrorClass(errors.HierarchicalError)
	SystemError       = pushErrorClass(errors.SystemError)
)

// 默认错误类型，错误类型显示为相关模块名
var MErr = NewErrorClass("ModuleReplaceType", nil, errors.SetData(errors.DataKey(ErrorDataKeyNoType), true))

// 已经存在
var ExistError = NewErrorClass("ExistError", nil)

// 不存在
var NotExistError = NewErrorClass("NotExistError", nil)

// 地址冲突
var CollisionError = NewErrorClass("Collision", nil)

// 未初始化
var UninitedError = NewErrorClass("UninitedError", nil)

// 已关闭
var ClosedError = NewErrorClass("CloseError", nil)

// 内部使用，随时可能改变
var (
	NormalError = NewErrorClass("Error", nil)

	UnsupportedError = NewErrorClass("Unsupported", nil)

	ServiceCallError    = NewErrorClass("ServiceCall", nil)
	ServiceChangedError = NewErrorClass("ServiceChanged", nil)
	ServiceTimeoutError = NewErrorClass("[Timeout]", nil)
	ServicePartError    = NewErrorClass("ServicePart", nil)
	ServiceProcError    = NewErrorClass("ServiceProc", nil)
	ServiceOutputError  = NewErrorClass("ServiceOutput", nil)

	FileNotFoundError      = NewErrorClass("FileNotFound", NotExistError)
	ClassNotFoundError     = NewErrorClass("ClassNotFound", NotExistError)
	FieldNotFoundError     = NewErrorClass("FieldNotFound", NotExistError)
	DatadNotFoundError     = NewErrorClass("DataNotFound", NotExistError)
	NamespaceNotFoundError = NewErrorClass("NamespaceNotFound", NotExistError)
	KeyspaceNotFoundError  = NewErrorClass("KeyspaceNotFound", NotExistError)

	ErrProgram = NewErrorClass("Program", nil)
	ErrParams  = NewErrorClass("Params", nil)

	ErrFormat = NewErrorClass("[Format]", nil)
	ErrParam  = NewErrorClass("[Param]", ErrFormat)

	ErrParser    = NewErrorClass("[Parser]", nil)
	ErrValid     = NewErrorClass("[Valid]", nil)
	ErrJson      = NewErrorClass("[Json]", nil)
	ErrNoRight   = NewErrorClass("[NoRight]", nil)
	ErrExec      = NewErrorClass("[Exec]", nil)
	ErrNoSupport = NewErrorClass("[NoSupport]", nil)

	ErrCQLExec = NewErrorClass("[CQL.Exec]", nil)

	ErrUnKnown = NewErrorClass("[UnKnown]", nil)

	ErrDebug = NewErrorClass("[debug]", nil)

	ErrRedis = NewErrorClass("[Redis]", nil)

	BreakError = NewErrorClass("Break", nil)
)
