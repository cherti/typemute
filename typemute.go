package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/sqp/pulseaudio"
)

var timeoutSeconds = flag.Float64("t", 1.0, "mute timeout after last keypress in seconds")
var verbose = flag.Bool("v", false, "give more detailed output")
var initialMicState []*pulseaudio.Object = getUnmutedMics()

func monitorKeypresses(scanner *bufio.Scanner, keypressDump chan bool) {
	fmt.Println("Monitoring keyboard. Press Ctrl+C to exit.")
	for scanner.Scan() {
		slc := strings.Split(scanner.Text(), " ")
		if slc[len(slc)-1] == "pressed" {
			keypressDump <- true
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}
}

func getUnmutedMics() []*pulseaudio.Object {
	// get a connection to pulseaudio
	pulse, err := pulseaudio.New()
	if err != nil {
		fmt.Println(err)
	}

	// obtain all microphones
	sources, err := pulse.Core().ListPath("Sources")
	mics := make([]*pulseaudio.Object, len(sources))
	i := 0
	// map object paths to source-objects
	for _, src := range sources {
		dev := pulse.Device(src)
		var muted bool
		dev.Get("Mute", &muted)
		props, _ := dev.MapString("PropertyList")
		devtype := props["device.class"]
		// filter out monitor-devices, because we only want actual microphones
		if devtype == "sound" && !muted {
			mics[i] = dev
			i++
		}
	}
	return mics
}

func mute(keypressDump chan bool) []*pulseaudio.Object {
	devices2mute := getUnmutedMics()

	// actually mute devices
	for _, dev := range devices2mute {
		if dev != nil {
			dev.Set("Mute", true)
		}
	}

	// wait until 1s passes without a keypress
	for {
		select {
		case <-keypressDump:
		case <-time.After(time.Duration(*timeoutSeconds) * time.Second):
			return devices2mute
		}
	}
}

func unmute(devices2unmute []*pulseaudio.Object) {
	for _, dev := range devices2unmute {
		if dev != nil {
			dev.Set("Mute", false)
		}
	}
}

func main() {

	flag.Parse()

	// restore unmuted mic state on SIGTERM
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			unmute(initialMicState)
			os.Exit(0)
		}
	}()

	cmd := exec.Command("pactl", "load-module", "module-dbus-protocol")
	cmd.Start()
	cmd.Wait()
	cmd := exec.Command("sudo", "unbuffer", "libinput", "debug-events")
	out, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}

	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
	}

	keypressDump := make(chan bool)
	s := bufio.NewScanner(out)
	go monitorKeypresses(s, keypressDump)

	for {
		<-keypressDump
		if *verbose {
			fmt.Println("muting")
		}
		mics := mute(keypressDump)
		if *verbose {
			fmt.Println("unmuting")
		}
		unmute(mics)
	}
}
