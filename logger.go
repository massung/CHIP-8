/* Copyright (c) 2017 Jeffrey Massung
 *
 * This software is provided 'as-is', without any express or implied
 * warranty.  In no event will the authors be held liable for any damages
 * arising from the use of this software.
 *
 * Permission is granted to anyone to use this software for any purpose,
 * including commercial applications, and to alter it and redistribute it
 * freely, subject to the following restrictions:
 *
 * 1. The origin of this software must not be misrepresented; you must not
 *    claim that you wrote the original software. If you use this software
 *    in a product, an acknowledgment in the product documentation would be
 *    appreciated but is not required.
 *
 * 2. Altered source versions must be plainly marked as such, and must not be
 *    misrepresented as being the original software.
 *
 * 3. This notice may not be removed or altered from any source distribution.
 */

package main

import (
	"strings"
)

// Logger creates a new output log that can be viewed and scrolled.
type Logger struct {
	// buf contains each line of logged text.
	buf []string

	// pos is the current user read position within the log.
	pos int
}

// NewLog creates a new Logger.
func NewLog() *Logger {
	return &Logger{
		buf: make([]string, 0, 100),
		pos: 0,
	}
}

// Log outputs a new line to the log.
func (log *Logger) Log(s ...string) {
	scroll := log.pos == len(log.buf)

	// add the new line
	log.buf = append(log.buf, strings.Join(s, " "))

	if scroll {
		log.pos = len(log.buf)
	}
}

// Logln outline a new line to the log, with an empty line prefixed.
func (log *Logger) Logln(s ...string) {
	scroll := log.pos == len(log.buf)

	// append the lines
	log.buf = append(log.buf, "", strings.Join(s, " "))

	if scroll {
		log.pos = len(log.buf)
	}
}

// Window returns a slice of strings logged.
func (log *Logger) Window(n int) []string {
	start := log.pos - n

	// don't scroll past the beginning
	if start < 0 {
		start = 0
	}

	if start+n >= len(log.buf) {
		return log.buf[start:]
	}

	return log.buf[start : start+n]
}

// Scroll the log to the beginning.
func (log *Logger) Home() {
	log.pos = 0
}

// Scroll the log to the end.
func (log *Logger) End() {
	log.pos = len(log.buf)
}

// ScrollUp scrolls the log back one position.
func (log *Logger) ScrollUp() {
	log.pos -= 1

	// clamp to home
	if log.pos < 0 {
		log.Home()
	}
}

// ScrollDown scrolls the log forward one position.
func (log *Logger) ScrollDown(windowSize int) {
	log.pos += 1

	// if less than the window size, drop to it
	if log.pos <= windowSize {
		log.pos = windowSize + 1
	}

	// clamp to home
	if log.pos >= len(log.buf) {
		log.End()
	}
}
