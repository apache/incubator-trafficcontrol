// Inspired by https://www.goinggo.net/2013/11/using-log-package-in-go.html
package log

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

var (
	Debug       *log.Logger
	Info        *log.Logger
	Warning     *log.Logger
	Error       *log.Logger
	Event       *log.Logger
	debugCloser io.Closer
	infoCloser  io.Closer
	warnCloser  io.Closer
	errCloser   io.Closer
	eventCloser io.Closer
)

func initLogger(logger **log.Logger, oldLogCloser *io.Closer, newLogWriter io.WriteCloser, logPrefix string, logFlags int) {
	if newLogWriter == nil {
		*logger = nil
		if *oldLogCloser != nil {
			(*oldLogCloser).Close()
			*oldLogCloser = nil
		}
		return
	}

	if *logger != nil {
		(*logger).SetOutput(newLogWriter)
	} else {
		*logger = log.New(newLogWriter, logPrefix, logFlags)
	}

	if *oldLogCloser != nil {
		(*oldLogCloser).Close()
	}
	*oldLogCloser = newLogWriter
}

// Init initailizes the logs with the given io.WriteClosers. If `Init` was previously called, existing loggers are Closed. If you have loggers which are not Closers or which must not be Closed, wrap them with `log.NopCloser`.
func Init(eventW, errW, warnW, infoW, debugW io.WriteCloser) {
	initLogger(&Debug, &debugCloser, debugW, "DEBUG: ", log.Lshortfile)
	initLogger(&Info, &infoCloser, infoW, "INFO: ", log.Lshortfile)
	initLogger(&Warning, &warnCloser, warnW, "WARNING: ", log.Lshortfile)
	initLogger(&Error, &errCloser, errW, "ERROR: ", log.Lshortfile)
	initLogger(&Event, &eventCloser, eventW, "", 0)
}

func logf(logger *log.Logger, format string, v ...interface{}) {
	if logger == nil {
		return
	}
	logger.Output(stackFrame, time.Now().UTC().Format(timeFormat)+": "+strings.Replace(strings.TrimSpace(fmt.Sprintf(format, v...)), "\n", `\n`, -1)+"\n")
}

func logln(logger *log.Logger, v ...interface{}) {
	if logger == nil {
		return
	}
	logger.Output(stackFrame, time.Now().UTC().Format(timeFormat)+": "+strings.Replace(strings.TrimSpace(fmt.Sprintln(v...)), "\n", `\n`, -1)+"\n")
}

const timeFormat = time.RFC3339Nano
const stackFrame = 3

func Errorf(format string, v ...interface{}) { logf(Error, format, v...) }
func Errorln(v ...interface{})               { logln(Error, v...) }
func Warnf(format string, v ...interface{})  { logf(Warning, format, v...) }
func Warnln(v ...interface{})                { logln(Warning, v...) }
func Infof(format string, v ...interface{})  { logf(Info, format, v...) }
func Infoln(v ...interface{})                { logln(Info, v...) }
func Debugf(format string, v ...interface{}) { logf(Debug, format, v...) }
func Debugln(v ...interface{})               { logln(Debug, v...) }

const eventFormat = "%.3f %s"

func eventTime(t time.Time) float64 {
	return float64(t.Unix()) + (float64(t.Nanosecond()) / 1e9)
}

// event log entries (TM event.log, TR access.log, etc)
func Eventf(t time.Time, format string, v ...interface{}) {
	if Event == nil {
		return
	}
	// 1484001185.287 ...
	Event.Printf(eventFormat, eventTime(t), fmt.Sprintf(format, v...))
}

// EventfRaw writes to the event log with no prefix.
func EventfRaw(format string, v ...interface{}) {
	if Event == nil {
		return
	}
	Event.Printf(format, v...)
}

// EventRaw writes to the event log with no prefix, and no newline. Go's Printf is slow, using this with string concatenation is by far the fastest way to log, and should be used for frequent logs.
func EventRaw(s string) {
	if Event == nil {
		return
	}
	Event.Output(stackFrame, s)
}

// Close calls `Close()` on the given Closer, and logs any error. On error, the context is logged, followed by a colon, the error message, and a newline. This is primarily designed to be used in `defer`, for example, `defer log.Close(resp.Body, "readData fetching /foo/bar")`.
func Close(c io.Closer, context string) {
	err := c.Close()
	if err != nil {
		Errorf("%v: %v", context, err)
	}
}

// Closef acts like Close, with a given format string and values, followed by a colon, the error message, and a newline. The given values are not coerced, concatenated, or printed unless an error occurs, so this is more efficient than `Close()`.
func Closef(c io.Closer, contextFormat string, v ...interface{}) {
	err := c.Close()
	if err != nil {
		Errorf(contextFormat, v...)
		Errorf(": %v", err)
	}
}

// Write calls `Write()` on the given Writer, and logs any error. On error, the context is logged, followed by a colon, the error message, and a newline.
func Write(w io.Writer, b []byte, context string) {
	_, err := w.Write(b)
	if err != nil {
		Errorf("%v: %v", context, err)
	}
}

// Writef acts like Write, with a given format string and values, followed by a colon, the error message, and a newline. The given values are not coerced, concatenated, or printed unless an error occurs, so this is more efficient than `Write()`.
func Writef(w io.Writer, b []byte, contextFormat string, v ...interface{}) {
	_, err := w.Write(b)
	if err != nil {
		Errorf(contextFormat, v...)
		Errorf(": %v", err)
	}
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

func NopCloser(w io.Writer) io.WriteCloser {
	return nopCloser{w}
}

// LogLocation is a location to log to. This may be stdout, stderr, null (/dev/null), or a valid file path.
type LogLocation string

const (
	// LogLocationStdout indicates the stdout IO stream
	LogLocationStdout = "stdout"
	// LogLocationStderr indicates the stderr IO stream
	LogLocationStderr = "stderr"
	// LogLocationNull indicates the null IO stream (/dev/null)
	LogLocationNull = "null"
	//StaticFileDir is the directory that contains static html and js files.
	StaticFileDir = "/opt/traffic_monitor/static/"
)

func GetLogWriter(location LogLocation) (io.WriteCloser, error) {
	switch location {
	case LogLocationStdout:
		return NopCloser(os.Stdout), nil
	case LogLocationStderr:
		return NopCloser(os.Stderr), nil
	case LogLocationNull:
		fallthrough
	case "":
		return nil, nil
	default:
		return os.OpenFile(string(location), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	}
}

type Config interface {
	ErrorLog() LogLocation
	WarningLog() LogLocation
	InfoLog() LogLocation
	DebugLog() LogLocation
	EventLog() LogLocation
}

func GetLogWriters(cfg Config) (io.WriteCloser, io.WriteCloser, io.WriteCloser, io.WriteCloser, io.WriteCloser, error) {
	eventLoc := cfg.EventLog()
	errLoc := cfg.ErrorLog()
	warnLoc := cfg.WarningLog()
	infoLoc := cfg.InfoLog()
	debugLoc := cfg.DebugLog()

	eventW, err := GetLogWriter(eventLoc)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("getting log event writer %v: %v", eventLoc, err)
	}
	errW, err := GetLogWriter(errLoc)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("getting log error writer %v: %v", errLoc, err)
	}
	warnW, err := GetLogWriter(warnLoc)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("getting log warning writer %v: %v", warnLoc, err)
	}
	infoW, err := GetLogWriter(infoLoc)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("getting log info writer %v: %v", infoLoc, err)
	}
	debugW, err := GetLogWriter(debugLoc)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("getting log debug writer %v: %v", debugLoc, err)
	}
	return eventW, errW, warnW, infoW, debugW, nil
}

func InitCfg(cfg Config) error {
	eventW, errW, warnW, infoW, debugW, err := GetLogWriters(cfg)
	if err != nil {
		return err
	}
	Init(eventW, errW, warnW, infoW, debugW)
	return nil
}
