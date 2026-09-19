//go:build linux

package indicator

type Indicator struct {
	listener *DBusListener
}

func New() *Indicator {
	return &Indicator{}
}

func (ind *Indicator) Start() error {
	ind.listener = NewDBusListener(func(state State) {
		NotifyState(state)
	})
	return ind.listener.Start()
}

func (ind *Indicator) Stop() {
	if ind.listener != nil {
		ind.listener.Stop()
	}
}
