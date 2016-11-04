package errorconverter

import (
	"runtime"

	raven "github.com/getsentry/raven-go"
	"github.com/pkg/errors"
)

type causer interface {
	Cause() error
}

type stackTracer interface {
	causer
	StackTrace() errors.StackTrace
}

func rootStackTracer(err error) error {
	lastStackTracer := err
	for err != nil {
		// current err is not a stackTracer, return plain error
		if _, ok := err.(stackTracer); ok {
			lastStackTracer = err
		}

		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}
	return lastStackTracer
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

	rootErr := rootStackTracer(err)
	tracer, ok := rootErr.(stackTracer)
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
		frame := raven.NewStacktraceFrame(pc, file, line, 3, raven.DefaultClient.IncludePaths())
		if frame != nil {
			frames = append(frames, frame)
		}
	}
	// reverse frames, raven wants them in reverse order
	for i, j := 0, len(frames)-1; i < j; i, j = i+1, j-1 {
		frames[i], frames[j] = frames[j], frames[i]
	}
	return &raven.Stacktrace{
		Frames: frames,
	}
}
