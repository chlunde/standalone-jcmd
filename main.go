/*
standalone-jcmd - hack emulating jcmd without any java-devel depedencies.

Black box / reversed by using strace on jcmd.

Usage: ./jcmd <pid> <command>

Example: ./jcmd 123 GC.class_stats

TODO: jcmd 123 JFR.start duration=30s settings=profile filename=path/filename.jfr
*/

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"syscall"
	"time"
)

func activateAttachAPI(pid int) error {
	// It's not, lets do a quick ceremony of touching a file and
	// sending SIGQUIT to activate this feature

	attachpath := fmt.Sprintf("/proc/%v/cwd/.attach_pid%v", pid, pid)
	if err := os.WriteFile(attachpath, nil, 0o777); err != nil {

		// backup directory in case of permission issues
		altpath := fmt.Sprintf("/proc/%v/root/tmp/.attach_pid%v", pid, pid)
		if err := os.WriteFile(altpath, nil, 0o777); err != nil {
			return fmt.Errorf("could not touch file to activate attach api: %w", err)
		}
	}

	proc, err := os.FindProcess(pid)
	if err != nil { // can't happen on unix
		return fmt.Errorf("could not find process: %w", err)
	}

	if err := proc.Signal(syscall.SIGQUIT); err != nil {
		return fmt.Errorf("could not send signal 3 to activate attach API: %w", err)
	}

	// Check if the UNIX socket is active
	sock := socketPath(pid)
	for i := 1; i < 10; i++ {
		if _, err := os.Stat(sock); err != nil && !os.IsNotExist(err) {
			break
		}

		// exponential backoff
		time.Sleep(time.Duration(1<<uint(i)) * time.Millisecond)
	}

	return nil
}

func socketPath(pid int) string {
	return fmt.Sprintf("/proc/%v/root/tmp/.java_pid%v", pid, pid)
}

func connect(pid int) (*net.UnixConn, error) {
	sock := socketPath(pid)

	// Check if the UNIX socket is active
	if _, err := os.Stat(sock); err != nil && os.IsNotExist(err) {
		if err := activateAttachAPI(pid); err != nil {
			return nil, err
		}
	}

	addr, err := net.ResolveUnixAddr("unix", sock)
	if err != nil {
		return nil, err // can't happen (on linux)
	}

	return net.DialUnix("unix", nil, addr)
}

func main() {
	pidstr := os.Args[1]
	pid, err := strconv.Atoi(pidstr)
	if err != nil {
		fmt.Printf("%v is not a valid integer/pid: %v", pidstr, err)
		os.Exit(1)
	}

	conn, err := connect(pid)
	if err != nil {
		fmt.Printf("connect using attach api failed: %v", err)
		os.Exit(1)
	}

	var cmd string
	if len(os.Args) < 3 {
		cmd = "help"
	} else {
		cmd = os.Args[2]
	}

	// TODO: Double check if one write per step is required
	// TODO: What is the meaning here? What if there are arguments?
	for _, s := range []string{
		"1", "\x00", "jcmd", "\x00", cmd, "\x00", "\x00", "\x00",
	} {
		_, err := conn.Write([]byte(s))
		if err != nil {
			fmt.Printf("unable to send command to Java process: %v", err)
			os.Exit(1)
		}
	}

	_, err = io.Copy(os.Stdout, conn)
	if err != nil {
		fmt.Printf("unable to read jcmd response from Java process: %v", err)
		os.Exit(1)
	}
}
