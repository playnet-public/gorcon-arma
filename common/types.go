package common

//ScheduleFunc is a function type used by events
type ScheduleFunc func(cmd string)

//ScheduleFuncs is a map of functions that may be used by events
type ScheduleFuncs map[string]ScheduleFunc
