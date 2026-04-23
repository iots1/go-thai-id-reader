//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/getlantern/systray"
)

const (
	swHide    = 0
	swRestore = 9
)

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	user32                  = syscall.NewLazyDLL("user32.dll")
	procGetConsoleWindow    = kernel32.NewProc("GetConsoleWindow")
	procGetWindowLongPtrW   = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW   = user32.NewProc("SetWindowLongPtrW")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procShowWindow          = user32.NewProc("ShowWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")

	trayErrCh     chan error
	trayReadyCh   chan struct{}
	trayOnce      sync.Once
	trayReadyOnce sync.Once
)

const (
	gwlExStyle int32 = -20

	wsExToolWindow = 0x00000080
	wsExAppWindow  = 0x00040000

	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020
)

func int32ToUintptr(v int32) uintptr {
	return uintptr(int(v))
}

func runWithTray(startFn func() error) {
	trayErrCh = make(chan error, 1)
	trayReadyCh = make(chan struct{})

	// Hide the console as early as possible for background behavior.
	hideConsoleWindow()

	go keepConsoleHiddenUntilTrayReady()

	go func() {
		trayErrCh <- startFn()
	}()

	systray.Run(onTrayReady, onTrayExit)
}

func onTrayReady() {
	if iconData, err := loadTrayIcon(); err == nil {
		systray.SetIcon(iconData)
	} else {
		// Fallback: keep Windows built-in default icon when no custom icon is available.
		log.Printf("[main] tray icon fallback: %v", err)
	}

	systray.SetTitle("Meditech ID Reader")
	systray.SetTooltip("Meditech ID Reader is running")
	trayReadyOnce.Do(func() {
		close(trayReadyCh)
	})

	openItem := systray.AddMenuItem("Open App", "Show the running command window")
	closeItem := systray.AddMenuItem("Close", "Stop this process")

	go func() {
		for {
			select {
			case <-openItem.ClickedCh:
				showConsoleWindow()
			case <-closeItem.ClickedCh:
				trayOnce.Do(func() {
					systray.Quit()
				})
				return
			case err := <-trayErrCh:
				if err != nil {
					showConsoleWindow()
					log.Printf("[main] app stopped: %v", err)
				}
				trayOnce.Do(func() {
					systray.Quit()
				})
				return
			}
		}
	}()
}

func keepConsoleHiddenUntilTrayReady() {
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ticker.C:
			hideConsoleWindow()
		case <-trayReadyCh:
			hideConsoleWindow()
			return
		case <-timeout.C:
			return
		}
	}
}

func onTrayExit() {
	os.Exit(0)
}

func getConsoleWindow() uintptr {
	hwnd, _, _ := procGetConsoleWindow.Call()
	return hwnd
}

func hideConsoleWindow() {
	hwnd := getConsoleWindow()
	if hwnd == 0 {
		return
	}
	setConsoleTaskbarVisible(hwnd, false)
	procShowWindow.Call(hwnd, swHide)
}

func showConsoleWindow() {
	hwnd := getConsoleWindow()
	if hwnd == 0 {
		return
	}
	setConsoleTaskbarVisible(hwnd, true)
	procShowWindow.Call(hwnd, swRestore)
	procSetForegroundWindow.Call(hwnd)
}

func setConsoleTaskbarVisible(hwnd uintptr, visible bool) {
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, int32ToUintptr(gwlExStyle))
	newStyle := style

	if visible {
		newStyle &^= wsExToolWindow
		newStyle |= wsExAppWindow
	} else {
		newStyle &^= wsExAppWindow
		newStyle |= wsExToolWindow
	}

	if newStyle != style {
		procSetWindowLongPtrW.Call(hwnd, int32ToUintptr(gwlExStyle), newStyle)
		procSetWindowPos.Call(
			hwnd,
			0,
			0,
			0,
			0,
			0,
			swpNoSize|swpNoMove|swpNoZOrder|swpNoActivate|swpFrameChanged,
		)
	}
}

func loadTrayIcon() ([]byte, error) {
	iconPath := os.Getenv("TRAY_ICON_PATH")
	if iconPath == "" {
		exePath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("resolve executable path: %w", err)
		}
		iconPath = filepath.Join(filepath.Dir(exePath), "tray.ico")
	}

	iconData, err := os.ReadFile(iconPath)
	if err != nil {
		return nil, fmt.Errorf("read icon file (%s): %w", iconPath, err)
	}
	if len(iconData) == 0 {
		return nil, fmt.Errorf("icon file is empty (%s)", iconPath)
	}

	return iconData, nil
}
