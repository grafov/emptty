package src

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

// xorgSession defines structure for xorg
type xorgSession struct {
	*commonSession
	xorg *exec.Cmd
}

// Starts Xorg as carrier for Xorg Session.
func (x *xorgSession) startCarrier() {
	if !x.conf.DefaultXauthority {
		x.auth.usr().setenv(envXauthority, x.auth.usr().getenv(envXdgRuntimeDir)+"/.emptty-xauth")
		os.Remove(x.auth.usr().getenv(envXauthority))
	}

	x.auth.usr().setenv(envDisplay, ":"+x.getFreeXDisplay())

	// generate mcookie
	cmd := cmdAsUser(x.auth.usr(), lookPath("mcookie", "/usr/bin/mcookie"))
	mcookie, err := cmd.Output()
	handleErr(err)
	logPrint("Generated mcookie")

	// generate xauth
	cmd = cmdAsUser(x.auth.usr(), lookPath("xauth", "/usr/bin/xauth"), "add", x.auth.usr().getenv(envDisplay), ".", string(mcookie))
	_, err = cmd.Output()
	handleErr(err)
	logPrint("Generated xauthority")

	// start X
	logPrint("Starting Xorg")

	xorgArgs := []string{"vt" + x.conf.strTTY(), x.auth.usr().getenv(envDisplay)}
	if x.allowRootlessX() {
		xorgArgs = append(xorgArgs, "-keeptty")
	}

	if x.conf.XorgArgs != "" {
		arrXorgArgs := parseExec(x.conf.XorgArgs)
		xorgArgs = append(xorgArgs, arrXorgArgs...)
	}

	if x.allowRootlessX() {
		x.xorg = cmdAsUser(x.auth.usr(), lookPath("Xorg", "/usr/bin/Xorg"), xorgArgs...)
		x.xorg.Env = x.auth.usr().environ()
		if err := x.setTTYOwnership(x.conf, x.auth.usr().uid); err != nil {
			logPrint(err)
		}
	} else {
		x.xorg = exec.Command(lookPath("Xorg", "/usr/bin/Xorg"), xorgArgs...)
		os.Setenv(envDisplay, x.auth.usr().getenv(envDisplay))
		os.Setenv(envXauthority, x.auth.usr().getenv(envXauthority))
		x.xorg.Env = os.Environ()
	}

	x.xorg.Start()
	if x.xorg.Process == nil {
		handleStrErr("Xorg is not running")
	}
	logPrint("Started Xorg")

	if err := openXDisplay(x.auth.usr().getenv(envDisplay)); err != nil {
		handleStrErr("Could not open X Display.")
	}
}

// Gets Xorg Pid as int
func (x *xorgSession) getCarrierPid() int {
	if x.xorg == nil {
		handleStrErr("Xorg is not running")
	}
	return x.xorg.Process.Pid
}

// Finishes Xorg as carrier for Xorg Session
func (x *xorgSession) finishCarrier() error {
	// Stop Xorg
	x.xorg.Process.Signal(os.Interrupt)
	err := x.xorg.Wait()
	logPrint("Interrupted Xorg")

	// Remove auth
	os.Remove(x.auth.usr().getenv(envXauthority))
	logPrint("Cleaned up xauthority")

	// Revert rootless TTY ownership
	if x.allowRootlessX() {
		if err := x.setTTYOwnership(x.conf, os.Getuid()); err != nil {
			logPrint(err)
		}
	}

	return err
}

// Sets TTY ownership to defined uid, but keeps the original gid.
func (x *xorgSession) setTTYOwnership(conf *config, uid int) error {
	info, err := os.Stat(conf.ttyPath())
	if err != nil {
		return err
	}
	stat := info.Sys().(*syscall.Stat_t)

	err = os.Chown(conf.ttyPath(), uid, int(stat.Gid))
	if err != nil {
		return err
	}
	err = os.Chmod(conf.ttyPath(), 0620)
	return err
}

// Finds free display for spawning Xorg instance.
func (x *xorgSession) getFreeXDisplay() string {
	for i := 0; i < 32; i++ {
		filename := fmt.Sprintf("/tmp/.X%d-lock", i)
		if !fileExists(filename) {
			return strconv.Itoa(i)
		}
	}
	return "0"
}

// Checks is rootless Xorg is allowed to be used
func (x *xorgSession) allowRootlessX() bool {
	return x.conf.RootlessXorg && (x.conf.DaemonMode || x.conf.ttyPath() == getCurrentTTYName("", true))
}
