package errorconverter

import (
	"runtime"

	raven "github.com/getsentry/raven-go"
	"github.com/pkg/errors"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func PkgErrorToRavenPacket(err error) *raven.Packet {
	return raven.NewPacket(err.Error(), PkgErrorToRavenException(err))
}

func PkgErrorToRavenException(err error) *raven.Exception {
	return raven.NewException(err, PkgErrorToRavenStack(err))
}

// PkgErrorToRavenStack converts a github.com/pkg/errors error to a raven compatible stacktrace
func PkgErrorToRavenStack(err error) *raven.Stacktrace {
	var frames []*raven.StacktraceFrame

	tracer, ok := err.(stackTracer)
	if !ok {
		return &raven.Stacktrace{
			Frames: frames,
		}
	}

	for _, frame := range []errors.Frame(tracer.StackTrace()) {
		pc := uintptr(frame) - 1
		fn := runtime.FuncForPC(pc)
		line := 0
		file := ""
		if fn != nil {
			file, line = fn.FileLine(pc)
		}
		frames = append(frames, raven.NewStacktraceFrame(pc, file, line, 3, raven.DefaultClient.IncludePaths()))
	}
	// reverse frames, raven wants them in reverse order
	for i, j := 0, len(frames)-1; i < j; i, j = i+1, j-1 {
		frames[i], frames[j] = frames[j], frames[i]
	}
	return &raven.Stacktrace{
		Frames: frames,
	}
}
