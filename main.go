// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/exp/slices"
	"golang.org/x/sys/unix"
)

var (
	minAvg = flag.Float64("min-avg", 1, "")
	delay  = flag.Duration("d", 1*time.Minute, "")
)

func isatty() bool {
	_, err := unix.IoctlGetWinsize(int(os.Stdin.Fd()), unix.TIOCGWINSZ)
	return err == nil
}

func readOutput(r io.Reader, watchEstimated bool, stopFunc func()) {
	rSecs := regexp.MustCompile(`\d+ secs?`)
	rMins := regexp.MustCompile(`\d+ mins?`)
	rHours := regexp.MustCompile(`\d+ hours?`)
	rDays := regexp.MustCompile(`\d+ days?`)
	rRec := regexp.MustCompile(`, (\d+)/\d+`)

	getDuration := func(r *regexp.Regexp, s string, multiplier time.Duration) time.Duration {
		a, _, _ := strings.Cut(r.FindString(s), " ")
		n, _ := strconv.Atoi(a)
		return time.Duration(n) * multiplier
	}

	parseTime := func(t string) time.Duration {
		return getDuration(rDays, t, 24*time.Hour) +
			getDuration(rHours, t, time.Hour) +
			getDuration(rMins, t, time.Minute) +
			getDuration(rSecs, t, time.Second)
	}

	var timeStarted, timeEstimated time.Duration
	var recoveredBefore int
	var justFinished bool
	s := bufio.NewScanner(r)
	for s.Scan() {
		t := s.Text()
		switch {
		case strings.HasPrefix(t, "Status"):
			justFinished = strings.Contains(t, ": Exhausted") || strings.Contains(t, ": Bypass")
		case strings.HasPrefix(t, "Time.Started"):
			timeStarted = parseTime(t)
		case strings.HasPrefix(t, "Time.Estimated"):
			timeEstimated = parseTime(t)
		case strings.HasPrefix(t, "Recovered."):
			m := rRec.FindStringSubmatch(t)
			if m == nil {
				panic("no match for Recovered!")
			}
			recovered, _ := strconv.Atoi(m[1])
			var avg float64
			if timeStarted > 0 {
				avg = float64(recovered-recoveredBefore) * float64(time.Minute) / float64(timeStarted)
				t += fmt.Sprintf(" avg/min:%.2f", avg)
			}
			if justFinished {
				recoveredBefore = recovered
				break
			}
			if timeStarted <= *delay {
				break
			}
			if timeEstimated <= *delay && watchEstimated {
				break
			}
			if avg < *minAvg {
				stopFunc()
				timeStarted, timeEstimated = 0, 0
			}
		}
		fmt.Println(t)
	}
}

func runWithPty(c *exec.Cmd) error {
	pr, pw := io.Pipe()
	defer pw.Close()

	c.Stdout = pw
	c.Stderr = os.Stderr

	f, err := pty.Start(c)
	if err != nil {
		return fmt.Errorf("Start: %w", err)
	}

	go func() {
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			_, _ = f.Write([]byte(s.Text()))
		}
		if err := s.Err(); err != nil {
			log.Println("Scan:", err)
		}
	}()

	go readOutput(pr, true, func() { _, _ = f.Write([]byte{'b'}) })

	if err := c.Wait(); err != nil {
		return fmt.Errorf("Wait: %w", err)
	}

	return nil
}

func runWithStdin(c *exec.Cmd) error {
	pr, pw := io.Pipe()
	defer pw.Close()

	c.Stdin = os.Stdin
	c.Stdout = pw
	c.Stderr = os.Stderr

	if err := c.Start(); err != nil {
		return fmt.Errorf("Start: %w", err)
	}

	go readOutput(pr, false, func() { _ = c.Process.Signal(os.Interrupt) })

	if err := c.Wait(); err != nil {
		return fmt.Errorf("Wait: %w", err)
	}

	return nil
}

func main() {
	var hcArgs []string
	if i := slices.Index(os.Args, "--"); i == -1 {
		hcArgs = os.Args[1:]
		os.Args = os.Args[:1]
	} else {
		hcArgs = os.Args[i+1:]
		os.Args = os.Args[:i]
	}
	flag.Parse()

	if !slices.ContainsFunc(hcArgs, func(s string) bool { return strings.HasPrefix(s, "--status-timer") }) {
		hcArgs = slices.Insert(hcArgs, 1, "--status-timer=15")
	}
	if !slices.Contains(hcArgs, "--status") {
		hcArgs = slices.Insert(hcArgs, 1, "--status")
	}
	c := exec.Command(hcArgs[0], hcArgs[1:]...)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		_ = c.Process.Kill()
		os.Exit(255)
	}()

	var err error
	if isatty() {
		err = runWithPty(c)
	} else {
		err = runWithStdin(c)
	}
	if err != nil {
		log.Println(err)
	}
}
