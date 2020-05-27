package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/sqp/pulseaudio"
)

var timeout = flag.Duration("t", 500*time.Millisecond, "mute timeout after last keypress")
var verbose = flag.Bool("v", false, "give more detailed output")

type microphoneController struct {
	pulse           *pulseaudio.Client
	initialMicState []*pulseaudio.Object
}

func newMicCtrl() microphoneController {
	// load PA-dbus-module as it is not loaded by default
	exec.Command("pacmd", "load-module", "module-dbus-protocol").Run()

	// establish a connection to pulseaudio
	var err error
	pulse, err := pulseaudio.New()
	if err != nil {
		log.Fatal(err)
	}
	mc := microphoneController{pulse, nil}
	mc.initialMicState = mc.getUnmutedMics()
	return mc
}

func (micCtrl microphoneController) getUnmutedMics() []*pulseaudio.Object {
	// obtain all microphones
	sources, err := micCtrl.pulse.Core().ListPath("Sources")
	if err != nil {
		return []*pulseaudio.Object{}
	}

	mics := make([]*pulseaudio.Object, len(sources))
	i := 0
	// map object paths to source-objects
	for _, src := range sources {
		dev := micCtrl.pulse.Device(src)
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

func (micCtrl microphoneController) mute(keypressDump <-chan bool) []*pulseaudio.Object {
	devices2mute := micCtrl.getUnmutedMics()

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
		case <-time.After(*timeout):
			return devices2mute
		}
	}
}

func (micCtrl microphoneController) unmute(devices2unmute []*pulseaudio.Object) {
	for _, dev := range devices2unmute {
		if dev != nil {
			dev.Set("Mute", false)
		}
	}
}

func (micCtrl microphoneController) restoreInitialState() {
	micCtrl.unmute(micCtrl.initialMicState)
}

func monitorKeypresses(scanner *bufio.Scanner, keypressDump chan<- bool) {
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

func main() {
	flag.Parse()

	logger := log.New(ioutil.Discard, "  ", 0)
	if *verbose {
		logger.SetOutput(os.Stdout)
	}

	micCtrl := newMicCtrl()

	// restore unmuted mic state on SIGTERM
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			micCtrl.restoreInitialState()
			os.Exit(0)
		}
	}()

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

		logger.Println("muting")

		mics := micCtrl.mute(keypressDump)

		logger.Println("unmuting")

		micCtrl.unmute(mics)
	}
}
