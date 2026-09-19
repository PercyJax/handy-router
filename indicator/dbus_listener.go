//go:build linux

package indicator

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

type State string

const (
	StateIdle       State = "idle"
	StateRecording  State = "recording"
	StateProcessing State = "processing"
)

type iconSignature [16]byte

var knownIcons map[iconSignature]State

func computeSignature(data []byte) iconSignature {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return md5.Sum(data)
	}
	bounds := img.Bounds()
	buf := make([]byte, 0, bounds.Dx()*bounds.Dy()*4)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			buf = append(buf, byte(r>>8), byte(g>>8), byte(b>>8), byte(a>>8))
		}
	}
	return md5.Sum(buf)
}

func init() {
	knownIcons = make(map[iconSignature]State)
	paths := map[string]State{
		"/usr/lib/Handy/resources/tray_idle.png":         StateIdle,
		"/usr/lib/Handy/resources/tray_recording.png":    StateRecording,
		"/usr/lib/Handy/resources/tray_transcribing.png": StateProcessing,
		"/usr/lib/Handy/resources/handy.png":             StateIdle,
		"/usr/lib/Handy/resources/recording.png":         StateRecording,
		"/usr/lib/Handy/resources/transcribing.png":      StateProcessing,
	}
	for path, state := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		sig := computeSignature(data)
		knownIcons[sig] = state
	}
	log.Printf("Loaded %d known icon signatures", len(knownIcons))
}

type DBusListener struct {
	conn      *dbus.Conn
	handyName string
	handyPath string
	mu        sync.Mutex
	state     State
	onChange  func(State)
}

func NewDBusListener(onChange func(State)) *DBusListener {
	return &DBusListener{
		state:    StateIdle,
		onChange: onChange,
	}
}

func (l *DBusListener) findHandyService() (string, string, error) {
	obj := l.conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher")
	call := obj.Call("org.freedesktop.DBus.Properties.GetAll", 0, "org.kde.StatusNotifierWatcher")
	if call.Err != nil {
		return "", "", fmt.Errorf("failed to get watcher properties: %w", call.Err)
	}

	var props map[string]dbus.Variant
	if err := call.Store(&props); err != nil {
		return "", "", fmt.Errorf("failed to store properties: %w", err)
	}

	if variant, ok := props["RegisteredStatusNotifierItems"]; ok {
		var items []string
		if err := variant.Store(&items); err != nil {
			return "", "", fmt.Errorf("failed to store items: %w", err)
		}

		for _, item := range items {
			parts := strings.SplitN(item, "/", 2)
			if len(parts) == 2 {
				serviceName := parts[0]
				objectPath := "/" + parts[1]

				titleObj := l.conn.Object(serviceName, dbus.ObjectPath(objectPath))
				titleCall := titleObj.Call("org.freedesktop.DBus.Properties.Get", 0, "org.kde.StatusNotifierItem", "Title")
				if titleCall.Err == nil {
					var title string
					if err := titleCall.Store(&title); err == nil {
						if strings.ToLower(title) == "handy" {
							return serviceName, objectPath, nil
						}
					}
				}
			}
		}
	}

	return "", "", fmt.Errorf("Handy not found on D-Bus")
}

func (l *DBusListener) ensureHandy() bool {
	if l.handyName != "" && l.handyPath != "" {
		return true
	}

	name, path, err := l.findHandyService()
	if err != nil {
		return false
	}

	l.handyName, l.handyPath = name, path
	log.Printf("Found Handy at %s%s", name, path)
	return true
}

func (l *DBusListener) queryAndDetermineState() (State, bool) {
	if l.handyName == "" || l.handyPath == "" {
		return StateIdle, false
	}

	obj := l.conn.Object(l.handyName, dbus.ObjectPath(l.handyPath))
	call := obj.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.kde.StatusNotifierItem", "IconName")
	if call.Err != nil {
		return StateIdle, false
	}

	var iconPath string
	if err := call.Store(&iconPath); err != nil {
		return StateIdle, false
	}

	data, err := os.ReadFile(iconPath)
	if err != nil {
		return StateIdle, false
	}

	sig := computeSignature(data)
	if state, ok := knownIcons[sig]; ok {
		return state, true
	}

	return StateIdle, true
}

func (l *DBusListener) handleStateChange() {
	if !l.ensureHandy() {
		return
	}

	newState, ok := l.queryAndDetermineState()
	if !ok {
		l.handyName = ""
		l.handyPath = ""
		return
	}

	l.mu.Lock()
	oldState := l.state
	l.state = newState
	l.mu.Unlock()

	if oldState != newState {
		log.Printf("Handy state: %s -> %s", oldState, newState)
		if l.onChange != nil {
			l.onChange(newState)
		}
	}
}

func (l *DBusListener) Start() error {
	var err error
	l.conn, err = dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("failed to connect to session bus: %w", err)
	}

	l.handyName, l.handyPath, err = l.findHandyService()
	if err != nil {
		log.Printf("Handy not found: %v", err)
	} else {
		log.Printf("Found Handy at %s%s", l.handyName, l.handyPath)
	}

	// Subscribe to NewIcon signals
	err = l.conn.AddMatchSignal(
		dbus.WithMatchInterface("org.kde.StatusNotifierItem"),
		dbus.WithMatchMember("NewIcon"),
	)
	if err != nil {
		return fmt.Errorf("failed to add match: %w", err)
	}

	err = l.conn.AddMatchSignal(
		dbus.WithMatchInterface("org.kde.StatusNotifierWatcher"),
		dbus.WithMatchMember("StatusNotifierItemRegistered"),
	)
	if err != nil {
		return fmt.Errorf("failed to add match: %w", err)
	}

	err = l.conn.AddMatchSignal(
		dbus.WithMatchInterface("org.kde.StatusNotifierWatcher"),
		dbus.WithMatchMember("StatusNotifierItemUnregistered"),
	)
	if err != nil {
		return fmt.Errorf("failed to add match: %w", err)
	}

	// Large buffer so signals don't get dropped
	signalCh := make(chan *dbus.Signal, 100)
	l.conn.Signal(signalCh)

	// Dedicated consumer goroutine — drains channel fast, debounces state checks
	go func() {
		var debounce *time.Timer
		for sig := range signalCh {
			_ = sig // signal is just a wake-up trigger
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(50*time.Millisecond, func() {
				l.handleStateChange()
			})
		}
	}()

	log.Printf("Indicator watching Handy tray icon")
	return nil
}

func (l *DBusListener) Stop() {
	if l.conn != nil {
		l.conn.Close()
	}
}
