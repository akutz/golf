package logrus

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/akutz/golf"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	nocolor = 0
	red     = 31
	green   = 32
	yellow  = 33
	blue    = 34
	gray    = 37
)

var (
	baseTimestamp time.Time
	isTerminal    bool
)

func init() {
	baseTimestamp = time.Now()
}

func miniTS() int {
	return int(time.Since(baseTimestamp) / time.Second)
}

// TextFormatter generates golf-enabled logs in plain-text formt.
//
// The Formatter interface is used to implement a custom Formatter. It takes an
// `Entry`. It exposes all the fields, including the default ones:
//
// * `entry.Data["msg"]`. The message passed from Info, Warn, Error ..
// * `entry.Data["time"]`. The timestamp.
// * `entry.Data["level"]. The level the entry was logged at.
//
// Any additional fields added with `WithField` or `WithFields` are also in
// `entry.Data`. Format is expected to return an array of bytes which are then
// logged to `logger.Out`.
//
// More information at https://github.com/akutz/logrus/blob/master/formatter.go
type TextFormatter struct {
	log.TextFormatter
	sync.Once
	isTerminal bool
}

func (f *TextFormatter) init(entry *log.Entry) {
	if entry.Logger != nil {
		f.isTerminal = f.checkIfTerminal(entry.Logger.Out)
	}
}

func (f *TextFormatter) checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}

func (f *TextFormatter) Format(entry *log.Entry) ([]byte, error) {
	var keys []string = make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}

	if !f.DisableSorting {
		sort.Strings(keys)
	}

	b := &bytes.Buffer{}

	prefixFieldClashes(entry.Data)

	f.Do(func() { f.init(entry) })

	isColorTerminal := f.isTerminal && (runtime.GOOS != "windows")
	isColored := (f.ForceColors || isColorTerminal) && !f.DisableColors

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = DefaultTimestampFormat
	}
	if isColored {
		f.printColored(b, entry, keys, timestampFormat)
	} else {
		if !f.DisableTimestamp {
			f.appendKeyValue(b, "time", entry.Time.Format(timestampFormat))
		}
		f.appendKeyValue(b, "level", entry.Level.String())
		if entry.Message != "" {
			f.appendKeyValue(b, "msg", entry.Message)
		}
		for _, key := range keys {
			f.appendKeyValue(b, key, entry.Data[key])
		}
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func (f *TextFormatter) printColored(
	b *bytes.Buffer,
	entry *log.Entry,
	keys []string,
	timestampFormat string) {

	var levelColor int
	switch entry.Level {
	case log.DebugLevel:
		levelColor = gray
	case log.WarnLevel:
		levelColor = yellow
	case log.ErrorLevel, log.FatalLevel, log.PanicLevel:
		levelColor = red
	default:
		levelColor = blue
	}

	levelText := strings.ToUpper(entry.Level.String())[0:4]

	if !f.FullTimestamp {
		fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%04d] %-44s ",
			levelColor, levelText, miniTS(), entry.Message)
	} else {
		fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%s] %-44s ",
			levelColor,
			levelText,
			entry.Time.Format(timestampFormat),
			entry.Message)
	}
	for _, k := range keys {
		v := entry.Data[k]

		switch v.(type) {
		case golf.Golfs:
			fields := golf.Fore(k, v)
			for kk, vv := range fields {
				switch vvt := vv.(type) {
				case string:
					if needsQuoting(vvt) {
						fmt.Fprintf(b,
							" \x1b[%dm%s\x1b[0m=%s",
							levelColor, kk, vvt)
					} else {
						fmt.Fprintf(b,
							" \x1b[%dm%s\x1b[0m=%q",
							levelColor, kk, vvt)
					}
				default:
					fmt.Fprintf(b,
						" \x1b[%dm%s\x1b[0m=%+v",
						levelColor, kk, vvt)
				}

			}
		default:
			fmt.Fprintf(b, " \x1b[%dm%s\x1b[0m=%+v", levelColor, k, v)
		}
	}
}

func needsQuoting(text string) bool {
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.') {
			return false
		}
	}
	return true
}

func (f *TextFormatter) appendKeyValue(
	b *bytes.Buffer,
	key string, value interface{}) {

	switch value.(type) {
	case golf.Golfs:
		for ck, cv := range golf.Fore(key, value) {
			f.appendKeyValue(b, ck, cv)
		}
		return
	}

	b.WriteString(key)
	b.WriteByte('=')

	switch value := value.(type) {
	case string:
		if needsQuoting(value) {
			b.WriteString(value)
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	case error:
		errmsg := value.Error()
		if needsQuoting(errmsg) {
			b.WriteString(errmsg)
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	default:
		fmt.Fprint(b, value)
	}

	b.WriteByte(' ')
}
