//go:build !noutmp

package src

const tagUtmp = ""

// Adds UTMP entry as user process
func addUtmpEntry(username string, pid int, ttyNo string, xdisplay string) bool {
	// Not implemented yet
	return false
}

// End UTMP entry by marking as dead process
func endUtmpEntry(value bool) {
	// Not implemented yet
}

// Adds BTMP entry to log unsuccessful login attempt.
func addBtmpEntry(username string, pid int, ttyNo string) {
	// Not implemented yet
}
