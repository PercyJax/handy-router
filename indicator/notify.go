//go:build linux

package indicator

import (
	"fmt"
	"log"
	"sync"

	"github.com/godbus/dbus/v5"
)

const (
	dbusNotificationsName  = "org.freedesktop.Notifications"
	dbusNotificationsPath  = "/org/freedesktop/Notifications"
	dbusNotificationsIface = "org.freedesktop.Notifications"
)

var (
	defaultNotifier *Notifier
	notifierOnce    sync.Once
)

type Notifier struct {
	conn      *dbus.Conn
	mu        sync.Mutex
	currentID uint32
}

func getNotifier() *Notifier {
	notifierOnce.Do(func() {
		conn, err := dbus.SessionBus()
		if err != nil {
			log.Printf("Failed to connect to session bus: %v", err)
			return
		}
		defaultNotifier = &Notifier{conn: conn}
	})
	return defaultNotifier
}

// Notify sends or updates a persistent notification. Replaces the previous one.
func (n *Notifier) Notify(summary, body, iconPath string) (uint32, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	obj := n.conn.Object(dbusNotificationsName, dbus.ObjectPath(dbusNotificationsPath))

	call := obj.Call(
		dbusNotificationsIface+".Notify", 0,
		"Handy",              // app_name
		n.currentID,          // replaces_id (0 = new, >0 = update existing)
		iconPath,             // app_icon (file path or icon name)
		summary,              // summary
		body,                 // body
		[]string{},           // actions
		map[string]dbus.Variant{
			"urgency": dbus.MakeVariant(uint8(2)), // critical urgency
		},
		int32(-1), // expire_timeout: -1 = server default
	)
	if call.Err != nil {
		return 0, fmt.Errorf("notification failed: %w", call.Err)
	}

	var id uint32
	if err := call.Store(&id); err != nil {
		return 0, fmt.Errorf("failed to get notification id: %w", err)
	}

	n.currentID = id
	return id, nil
}

func (n *Notifier) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.currentID == 0 {
		return nil
	}

	obj := n.conn.Object(dbusNotificationsName, dbus.ObjectPath(dbusNotificationsPath))
	call := obj.Call(
		dbusNotificationsIface+".CloseNotification", 0,
		n.currentID,
	)
	id := n.currentID
	n.currentID = 0

	if call.Err != nil {
		return fmt.Errorf("close notification failed: %w", call.Err)
	}
	log.Printf("Closed notification id=%d", id)
	return nil
}

func NotifyState(state State) {
	n := getNotifier()
	if n == nil {
		return
	}

	switch state {
	case StateRecording:
		if _, err := n.Notify("Handy", "Recording...", "/usr/lib/Handy/resources/recording.png"); err != nil {
			log.Printf("Notification failed: %v", err)
		} else {
			log.Printf("Notification: Recording...")
		}
	case StateProcessing:
		if _, err := n.Notify("Handy", "Processing...", ""); err != nil {
			log.Printf("Notification failed: %v", err)
		} else {
			log.Printf("Notification: Processing...")
		}
	case StateIdle:
		if err := n.Close(); err != nil {
			log.Printf("Close notification failed: %v", err)
		} else {
			log.Printf("Notification: dismissed")
		}
	}
}
