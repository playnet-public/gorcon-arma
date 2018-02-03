package common

//ExtFunc is the definition of an externally exposed function
type ExtFunc struct {
	Key  string
	Func ScheduleFunc
}

//NewExtFunc generates an ExtFunc object
func NewExtFunc(key string, f ScheduleFunc) ExtFunc {
	return ExtFunc{
		Key:  key,
		Func: f,
	}
}

//ExtFuncs is an array if ExtFunc Objects
type ExtFuncs []ExtFunc

//NewExtFuncs generates an ExtFuncs object
func NewExtFuncs(fa ...ExtFunc) ExtFuncs {
	return fa
}

//ScheduleFunc is a function type used by events
type ScheduleFunc func(cmd string)

//ScheduleFuncs is a map of functions that may be used by events
type ScheduleFuncs map[string]ScheduleFunc
