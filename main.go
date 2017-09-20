package main

import (
	"fmt"
	"time"

	"github.com/gonutz/di8"
	"github.com/gonutz/w32"
	"github.com/gonutz/win"
)

func main() {
	var keyboard *di8.Device
	var eventBuf [32]di8.DEVICEOBJECTDATA
	var lCtrlDown, rCtrlDown, lShiftDown, rShiftDown bool
	toggle := func() {}
	update := func() {}
	reset := func() {}
	ctrlDown := func() bool { return lCtrlDown || rCtrlDown }
	shiftDown := func() bool { return lShiftDown || rShiftDown }

	window, err := win.NewWindow(
		0,
		0,
		300,
		150,
		"timerwindow",
		func(window w32.HWND, msg uint32, w, l uintptr) uintptr {
			switch msg {
			case w32.WM_TIMER:
				if keyboard != nil {
					n, err := keyboard.GetDeviceData(eventBuf[:], 0)
					// buffer overflows are fine, the unread keys can be handled
					// in the next timer tick
					if err != nil && err.Code() != di8.BUFFEROVERFLOW {
						check(err)
					}
					events := eventBuf[:n]
					for _, e := range events {
						switch e.Ofs {
						case di8.K_LCONTROL:
							lCtrlDown = di8.KeyDown(uint8(e.Data))
						case di8.K_RCONTROL:
							rCtrlDown = di8.KeyDown(uint8(e.Data))
						case di8.K_LSHIFT:
							lShiftDown = di8.KeyDown(uint8(e.Data))
						case di8.K_RSHIFT:
							rShiftDown = di8.KeyDown(uint8(e.Data))
						case di8.K_F12:
							if ctrlDown() && di8.KeyDown(uint8(e.Data)) {
								if shiftDown() {
									reset()
								} else {
									toggle()
								}
							}
						}
					}
				}
				update()
				return 0
			case w32.WM_PAINT:
				var ps w32.PAINTSTRUCT
				hdc := w32.BeginPaint(window, &ps)
				w32.TextOut(hdc, 0, 0, "Ctrl+F12 to toggle start")
				w32.TextOut(hdc, 0, 20, "Ctrl+Shift+F12 to reset")
				w32.EndPaint(window, &ps)
			case w32.WM_DESTROY:
				w32.PostQuitMessage(0)
				return 0
			}
			return w32.DefWindowProc(window, msg, w, l)
		},
	)
	check(err)
	w32.SetWindowText(window, "Timer")
	w32.SetTimer(window, 1, 250, 0)

	di, err := di8.Create(di8.HINSTANCE(w32.GetModuleHandle("")))
	check(err)
	defer di.Release()

	keyboard, err = di.CreateDevice(di8.GUID_SysKeyboard)
	check(err)
	defer keyboard.Release()

	check(keyboard.SetCooperativeLevel(
		di8.HWND(window),
		di8.SCL_BACKGROUND|di8.SCL_NONEXCLUSIVE,
	))

	check(keyboard.SetDataFormat(&di8.Keyboard))

	keyboard.SetProperty(
		di8.PROP_BUFFERSIZE,
		di8.NewPropDWord(0, di8.PH_DEVICE, uint32(len(eventBuf))),
	)

	check(keyboard.Acquire())
	defer keyboard.Unacquire()

	var timing bool
	var start time.Time
	var seconds float64
	toggle = func() {
		timing = !timing
		if timing {
			start = time.Now()
		}
	}
	update = func() {
		if timing {
			now := time.Now()
			dt := now.Sub(start)
			start = now
			seconds += dt.Seconds()
		}
		// print the time
		sec := int(seconds + 0.5)
		min := sec / 60
		h := min / 60
		sec %= 60
		min %= 60
		timeStr := fmt.Sprintf("%d:%2.2d:%2.2d", h, min, sec)
		caption := timeStr
		if !timing {
			caption = "Timer (" + timeStr + ")"
		}
		w32.SetWindowText(window, caption)
	}
	reset = func() {
		timing = false
		seconds = 0
	}

	win.RunMainLoop()
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
