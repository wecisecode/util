package merrs

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/spacemonkeygo/errors"
	"github.com/spf13/cast"
)

type ErrDataKey errors.DataKey

var ErrorDataKeyModule = ErrDataKey(errors.GenSym())
var ErrorDataKeyInform = ErrDataKey(errors.GenSym())
var ErrorDataKeyStacks = ErrDataKey(errors.GenSym())
var ErrorDataKeyCause = ErrDataKey(errors.GenSym())
var ErrorDataKeyNoType = ErrDataKey(errors.GenSym())

type ErrorClass struct {
	gec *errors.ErrorClass
}

func (e *ErrorClass) NewPlain(args ...interface{}) error {
	return e.NewWith("", []error{fmt.Errorf("%s", fmt.Sprint(args...))}, nil, 1)
}

func (e *ErrorClass) NewError(infos ...any) error {
	return e.New(append(infos, 1)...)
}

type SSTuples [][2]string
type SSTuple [2]string
type SSMap map[string]string
type SSMaps []map[string]string
type Map map[string]any
type Maps []map[string]any
type Module string

// 参数类型可以是 Module, msg string, cause error, SSMaps, SSMap, map[string]string, Map, map[string]any, SSTuples, SSTuple, [2]string
// depth int
func (e *ErrorClass) New(infos ...any) error {
	if len(infos) > 0 {
		if format, ok := infos[0].(string); ok {
			fcount := strings.Count(strings.ReplaceAll(format, "%%", ""), "%")
			if fcount > 0 && len(infos) > fcount {
				msg := fmt.Sprintf(format, infos[1:fcount+1]...)
				infos = append([]any{msg}, infos[fcount+1:]...)
			}
		}
	}
	module := ""
	depth := 1
	cause := []error{}
	infossms := SSMaps{}
	for i, info := range infos {
		if info == nil {
			continue
		}
		switch info := info.(type) {
		case Module:
			module = string(info)
		case int:
			depth += info
		case string:
			if info != "" {
				cause = append(cause, fmt.Errorf("%s", info))
			}
		case error:
			if info == nil {
				continue
			}
			cause = append(cause, info)
		case []error:
			for _, info := range info {
				if info == nil {
					continue
				}
				cause = append(cause, info)
			}
		case SSMaps:
			if len(info) == 0 {
				continue
			}
			infossms = append(infossms, info...)
		case SSMap:
			if len(info) == 0 {
				continue
			}
			infossms = append(infossms, info)
		case map[string]string:
			if len(info) == 0 {
				continue
			}
			infossms = append(infossms, info)
		case SSTuple:
			infossms = append(infossms, SSMap{info[0]: info[1]})
		case [2]string:
			infossms = append(infossms, SSMap{info[0]: info[1]})
		case []string:
			for i := 0; i < len(info); i += 2 {
				if i+1 < len(info) {
					infossms = append(infossms, SSMap{info[i]: info[i+1]})
				} else {
					infossms = append(infossms, SSMap{info[i]: ""})
				}
			}
		case SSTuples:
			for _, info := range info {
				infossms = append(infossms, SSMap{info[0]: info[1]})
			}
		case [][2]string:
			for _, info := range info {
				infossms = append(infossms, SSMap{info[0]: info[1]})
			}
		case Maps:
			if len(info) == 0 {
				continue
			}
			for _, info := range info {
				ssm := SSMap{}
				for k, v := range info {
					ssm[k] = cast.ToString(v)
				}
				infossms = append(infossms, ssm)
			}
		case []map[string]any:
			if len(info) == 0 {
				continue
			}
			for _, info := range info {
				ssm := SSMap{}
				for k, v := range info {
					ssm[k] = cast.ToString(v)
				}
				infossms = append(infossms, ssm)
			}
		case Map:
			if len(info) == 0 {
				continue
			}
			ssm := SSMap{}
			for k, v := range info {
				ssm[k] = cast.ToString(v)
			}
			infossms = append(infossms, ssm)
		case map[string]any:
			if len(info) == 0 {
				continue
			}
			ssm := SSMap{}
			for k, v := range info {
				ssm[k] = cast.ToString(v)
			}
			infossms = append(infossms, ssm)
		default:
			infossms = append(infossms, SSMap{fmt.Sprint("info", i): cast.ToString(info)})
		}
	}
	return e.NewWith(module, cause, infossms, depth)
}

func (e *ErrorClass) NewCause(cause ...error) error {
	return e.NewWith("", cause, nil, 1)
}

func (e *ErrorClass) Parent() *ErrorClass {
	pec := e.gec.Parent()
	if pec == nil {
		return nil
	}
	return getErrorClass(pec.String())
}

func (e *ErrorClass) String() string {
	return e.gec.String()
}

// inform map数组，用于显示一些有序的key-value信息，martix.SSMaps{{"k1","v1"},{"k2":"v2"}...}，
// stacks_depth >=0 自动生成调用栈信息，stacks_depth < 0 不打印调用栈，
func (e *ErrorClass) NewWith(module string, causes []error, inform SSMaps, stacks_depth int) error {
	sstacks := ""
	if stacks_depth >= 0 {
		stacks := getStack(2 + stacks_depth)
		if len(stacks) > 0 && module == "" {
			module = stacks[0].FuncName()
		}
		sstacks = stacks.String()
	}
	emsg := ""
	mcauses := []*Error{}
	for _, cause := range causes {
		mcause, isMError := mError(cause)
		if mcause != nil {
			if isMError {
				// 只有 MError 类型的错误才可以作为 诱因错误，可以用于深度判断错误类型
				mcauses = append(mcauses, mcause)
			}
			if emsg == "" {
				// 第一个非空诱因错误信息作为当前错误信息
				emsg = mcause.ErrorMsg
			} else if !isMError {
				// 非 MError 类型的错误只作为普通关联错误信息
				inform = append(inform, SSMap{"related error": mcause.Error()})
			}
		}
	}
	return e.newWithOptions(emsg, []errors.ErrorOption{
		errors.SetData(errors.DataKey(ErrorDataKeyModule), module),
		errors.SetData(errors.DataKey(ErrorDataKeyInform), inform),
		errors.SetData(errors.DataKey(ErrorDataKeyStacks), sstacks),
		errors.SetData(errors.DataKey(ErrorDataKeyCause), mcauses),
	})
}

func (e *ErrorClass) newWithOptions(message string, opts []errors.ErrorOption) error {
	return MError(e.gec.NewWith(message, opts...))
}

func (e *ErrorClass) Is(ec *ErrorClass) bool {
	return e.gec.Is(ec.gec)
}

func (e *ErrorClass) Contains(err error) bool {
	if err == nil {
		return false
	}
	gerr := gerror(err)
	if gerr == nil {
		return false
	}
	if e.gec.Contains(gerr) {
		return true
	}
	cause := gerr.GetData(errors.DataKey(ErrorDataKeyCause))
	if cause != nil {
		if mcauses, ok := cause.([]*Error); ok && mcauses != nil && len(mcauses) > 0 {
			for _, cause := range mcauses {
				if e.Contains(cause) {
					return true
				}
			}
			return false
		}
	}
	return false
}

// 可序列化Error，用于传输
type Error struct {
	ErrorType   string
	ErrorMsg    string
	ErrorModule string
	ErrorInform SSMaps
	ErrorStacks string
	ErrorCause  []*Error
	ErrorNoType bool
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	etype := e.ErrorType
	if e.ErrorNoType {
		etype = "[" + e.ErrorModule + "]"
	}
	message := strings.TrimRight(e.ErrorMsg, "\t\r\n ")
	if message != "" {
		if strings.Contains(message, "\n") {
			msg := strings.Replace(message, "\n", "\n  ", -1)
			message = fmt.Sprintf("%s:\n  %s", etype, msg)
		} else {
			message = fmt.Sprintf("%s: %s", etype, message)
		}
	}
	if module := e.ErrorModule; module != "" && !e.ErrorNoType {
		if message != "" {
			message += "\n"
		}
		message += fmt.Sprintf(
			"%s %-10s %s", etype, "module:", module)
	}
	if inform := e.ErrorInform; inform != nil && len(inform) > 0 {
		for _, kv := range inform {
			for k, v := range kv {
				if message != "" {
					message += "\n"
				}
				v = strings.TrimRight(v, "\t\r\n ")
				if strings.Contains(v, "\n") {
					message += fmt.Sprintf(
						"%s %s:\n  %s", etype, k,
						strings.Replace(v, "\n", "\n  ", -1))
				} else {
					message += fmt.Sprintf(
						"%s %-10s %s", etype, k+":", v)
				}
			}
		}
	}
	if stacks := e.ErrorStacks; stacks != "" {
		if message != "" {
			message += "\n"
		}
		message += fmt.Sprintf(
			"%s backtrace:\n  %s", etype,
			strings.Replace(stacks, "\n", "\n  ", -1))
	}
	if causes := e.ErrorCause; causes != nil {
		if message != "" {
			message += "\n"
		}
		for i, cause := range causes {
			causeKey := "cause"
			if len(causes) > 1 {
				causeKey += " " + strconv.Itoa(i)
			}
			message += fmt.Sprintf(
				"%s %s:\n  %s", etype, causeKey,
				strings.Replace(cause.Error(), "\n", "\n  ", -1))
		}
	}
	return message
}

var errorclassesmutex sync.RWMutex
var errorclasses = map[string]*ErrorClass{}

func NewErrorClass(name string, parent *ErrorClass, options ...errors.ErrorOption) (ec *ErrorClass) {
	options = append([]errors.ErrorOption{errors.NoCaptureStack()}, options...)
	if parent == nil {
		ec = pushErrorClass(errors.NewClass(name, options...))
	} else {
		ec = pushErrorClass(parent.gec.NewClass(name, options...))
	}
	return
}

func pushErrorClass(sec *errors.ErrorClass) (ec *ErrorClass) {
	errorclassesmutex.Lock()
	defer errorclassesmutex.Unlock()
	ec = &ErrorClass{gec: sec}
	errorclasses[ec.String()] = ec
	return
}

func getErrorClass(name string) (ec *ErrorClass) {
	errorclassesmutex.RLock()
	defer errorclassesmutex.RUnlock()
	return errorclasses[name]
}

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

var MErr = NewErrorClass("ModuleReplaceType", nil, errors.SetData(errors.DataKey(ErrorDataKeyNoType), true))

var NormalError = NewErrorClass("Error", nil)
var ErrProgram = NewErrorClass("Program", nil)
var ErrParams = NewErrorClass("Params", nil)
var ServiceCallError = NewErrorClass("ServiceCall", nil)
var ServiceChangedError = NewErrorClass("ServiceChanged", nil)
var ServiceTimeoutError = NewErrorClass("[Timeout]", nil)
var ServicePartError = NewErrorClass("ServicePart", nil)
var ServiceProcError = NewErrorClass("ServiceProc", nil)
var ServiceOutputError = NewErrorClass("ServiceOutput", nil)
var UnsupportedError = NewErrorClass("Unsupported", nil)
var CollisionError = NewErrorClass("Collision", nil)

var UninitedError = NewErrorClass("UninitedError", nil)
var ClosedError = NewErrorClass("CloseError", nil)
var ExistError = NewErrorClass("ExistError", nil)
var NotExistError = NewErrorClass("NotExistError", nil)

// 转为errors.Error，用于分类判断
func gerror(err error) *errors.Error {
	if err == nil {
		return nil
	}
	if se, ok := err.(*errors.Error); ok {
		return se
	}
	if me, ok := err.(*Error); ok {
		if me == nil {
			return nil
		}
		ec := getErrorClass(me.ErrorType)
		if ec == nil {
			ec = ErrProgram
		}

		se := ec.gec.NewWith(
			me.ErrorMsg,
			errors.SetData(errors.DataKey(ErrorDataKeyModule), me.ErrorModule),
			errors.SetData(errors.DataKey(ErrorDataKeyInform), me.ErrorInform),
			errors.SetData(errors.DataKey(ErrorDataKeyStacks), me.ErrorStacks),
			errors.SetData(errors.DataKey(ErrorDataKeyCause), me.ErrorCause),
			errors.SetData(errors.DataKey(ErrorDataKeyNoType), me.ErrorNoType),
		).(*errors.Error)

		return se
	}
	return errors.GetClass(err).New(err.Error()).(*errors.Error)
}

// 可序列化Error，用于传输
func MError(err error) *Error {
	e, _ := mError(err)
	return e
}

func mError(err error) (e *Error, isMError bool) {
	if err == nil {
		return nil, false
	}
	if me, ok := err.(*Error); ok {
		return me, true
	}
	if se, ok := err.(*errors.Error); ok {
		typ := se.Class().String()
		msg := se.WrappedErr().Error()
		return &Error{
			ErrorType:   typ,
			ErrorMsg:    msg,
			ErrorModule: cast.ToString(se.GetData(errors.DataKey(ErrorDataKeyModule))),
			ErrorInform: func() SSMaps {
				if edkv := se.GetData(errors.DataKey(ErrorDataKeyInform)); edkv != nil && edkv != "" {
					if maps, ok := edkv.(SSMaps); ok {
						return maps
					}
				}
				return nil
			}(),
			ErrorStacks: func() string {
				if stks := se.GetData(errors.DataKey(ErrorDataKeyStacks)); stks != nil && stks != "" {
					return cast.ToString(stks)
				}
				return se.Stack()
			}(),
			ErrorCause: func() []*Error {
				if cause := se.GetData(errors.DataKey(ErrorDataKeyCause)); cause != nil {
					if ecause, ok := cause.([]*Error); ok {
						return ecause
					}
				}
				return nil
			}(),
			ErrorNoType: cast.ToBool(se.GetData(errors.DataKey(ErrorDataKeyNoType))),
		}, true
	}
	return &Error{ErrorMsg: err.Error()}, false
}

func ErrorType(err error) string {
	switch e := err.(type) {
	case *Error:
		return e.ErrorType
	case *errors.Error:
		return e.Class().String()
	}
	return ErrProgram.String()
}

// 参数类型可以是 msg string, cause error, SSMaps, SSMap, map[string]any, map[string]any, SSTuples, SSTuple, [2]string
func New(info ...any) error {
	return MErr.New(append(info, 1)...)
}

func NewError(info ...any) error {
	return MErr.New(append(info, 1)...)
}

func NewCause(cause ...error) error {
	return MErr.NewWith("", cause, nil, 1)
}

func NewWith(module string, err error, inform SSMaps, stacks_depth int) *Error {
	if module == "" {
		module = "Error"
	}
	if stacks_depth >= 0 {
		stacks_depth += 1
	}
	return MErr.NewWith(module, []error{err}, inform, stacks_depth).(*Error)
}

type stack []frame

func (me stack) String() string {
	var frames []string
	for _, stk := range me {
		frames = append(frames, stk.String())
	}
	return strings.Join(frames, "\n")
}

func getStack(depth int) (stack stack) {
	var pcs [256]uintptr
	amount := runtime.Callers(depth+1, pcs[:])
	stack = make([]frame, amount)
	for i := 0; i < amount; i++ {
		stack[i] = frame{pcs[i]}
	}
	return stack
}

// frame logs the pc at some point during execution.
type frame struct {
	pc uintptr
}

// String returns a human readable form of the frame.
func (e frame) FuncName() string {
	if e.pc == 0 {
		return ""
	}
	f := runtime.FuncForPC(e.pc)
	if f == nil {
		return ""
	}
	fns := strings.Split(f.Name(), ".")
	if len(fns) == 0 {
		return ""
	}
	return fns[len(fns)-1]
}

// String returns a human readable form of the frame.
func (e frame) String() string {
	if e.pc == 0 {
		return "unknown.unknown:0"
	}
	f := runtime.FuncForPC(e.pc)
	if f == nil {
		return "unknown.unknown:0"
	}
	file, line := f.FileLine(e.pc)
	return fmt.Sprintf("%s:%s:%d", f.Name(), filepath.Base(file), line)
}
