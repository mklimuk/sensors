package console

import (
	"fmt"
	"io"
	"os"
)

const PictoTicket = "🎟 "
const PictoFinish = "🏁"
const PictoNotebook = "📒"
const PictoBottle = "🍾"
const PictoCoffee = "☕"
const PictoTree = "🌲"
const PictoPolice = "👮"
const PictoStop = "🚫"
const PictoGhost = "👻"
const PictoCalendar = "📅"
const PictoPin = "📌"
const PictoThermometer = "🌡"
const PictoHumidity = "💧"
const PictoWoman = "👩"
const PictoKey = "🔑"
const PictoCert = "📜"

var writer io.Writer
var errWriter io.Writer

var Trace bool

func init() {
	writer = os.Stdout
	errWriter = os.Stderr
}

func SetOutput(w, errw io.Writer) {
	writer = w
	errWriter = errw
}

func Format(err error) string {
	return fmt.Sprintf("%s: %s\n", Red("ERROR"), err.Error())
}

func Error(msg string) {
	_, _ = fmt.Fprintf(errWriter, "%s: %s\n", Red("ERROR"), msg)
}

func Errorf(msg string, args ...interface{}) {
	_, _ = fmt.Fprintf(errWriter, "%s: %s\n", Red("ERROR"), fmt.Sprintf(msg, args...))
}

func Warn(msg string) {
	_, _ = fmt.Fprintf(errWriter, "%s: %s\n", Yellow("WARN"), msg)
}

func Warnf(msg string, args ...interface{}) {
	_, _ = fmt.Fprintf(errWriter, "%s: %s\n", Yellow("WARN"), fmt.Sprintf(msg, args...))
}

func Info(msg string) {
	_, _ = fmt.Fprintf(writer, "%s %s\n", White("..."), msg)
}

func Infof(msg string, args ...interface{}) {
	_, _ = fmt.Fprintf(writer, "%s %s\n", White("..."), fmt.Sprintf(msg, args...))
}

func Debug(msg string) {
	if Trace {
		_, _ = fmt.Fprintf(writer, "%s %s\n", White("[DEBUG]"), msg)
	}
}

func Debugf(msg string, args ...interface{}) {
	if Trace {
		_, _ = fmt.Fprintf(writer, "%s %s\n", White("[DEBUG]"), fmt.Sprintf(msg, args...))
	}
}

func PInfof(picto, msg string, args ...interface{}) {
	_, _ = fmt.Fprintf(writer, "%s %s\n", picto, fmt.Sprintf(msg, args...))
}

func Print(msg string) {
	_, _ = fmt.Fprintln(writer, msg)
}

func Printf(msg string, args ...interface{}) {
	_, _ = fmt.Fprintf(writer, msg, args...)
}
