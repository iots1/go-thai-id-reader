//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

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
	procMessageBoxW         = user32.NewProc("MessageBoxW")

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

	mbOk              = 0x00000000
	mbYesNo           = 0x00000004
	mbIconInformation = 0x00000040
	mbIconWarning     = 0x00000030
	mbIconQuestion    = 0x00000020
	mbSystemModal     = 0x00001000
	mbSetForeground   = 0x00010000

	idYes = 6

	createNewProcessGroup = 0x00000200
	detachedProcess       = 0x00000008
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
	deviceMenu := systray.AddMenuItem("Device Info", "Inspect device identifiers")
	smbiosItem := deviceMenu.AddSubMenuItem("Check SMBIOS GUID", "Show this machine's SMBIOS GUID")
	copySmbiosItem := deviceMenu.AddSubMenuItem("Copy SMBIOS GUID", "Copy this machine's SMBIOS GUID to clipboard")
	editEnvItem := systray.AddMenuItem("Edit .env", "Edit the .env configuration file")
	systray.AddSeparator()
	closeItem := systray.AddMenuItem("Close", "Stop this process")

	go func() {
		for {
			select {
			case <-openItem.ClickedCh:
				showConsoleWindow()
			case <-smbiosItem.ClickedCh:
				go showSMBIOSGUIDDialog()
			case <-copySmbiosItem.ClickedCh:
				go copySMBIOSGUIDToClipboard()
			case <-editEnvItem.ClickedCh:
				go editEnvFile()
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

func showSMBIOSGUIDDialog() {
	guid := getSMBIOSGUID()
	if guid == "" {
		showMessageBox("SMBIOS GUID", "Unable to read SMBIOS GUID on this machine.", mbIconWarning)
		return
	}
	showMessageBox("SMBIOS GUID", guid, mbIconInformation)
}

func copySMBIOSGUIDToClipboard() {
	guid := getSMBIOSGUID()
	if guid == "" {
		showMessageBox("SMBIOS GUID", "Unable to read SMBIOS GUID on this machine.", mbIconWarning)
		return
	}
	if err := copyToClipboard(guid); err != nil {
		log.Printf("[tray] copy SMBIOS GUID: %v", err)
		showMessageBox("SMBIOS GUID", "Failed to copy SMBIOS GUID to clipboard.", mbIconWarning)
		return
	}
	showMessageBox("SMBIOS GUID", "Copied to clipboard:\n"+guid, mbIconInformation)
}

func copyToClipboard(s string) error {
	cmd := exec.Command("clip")
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run()
}

func editEnvFile() {
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		if err := os.WriteFile(envFilePath, []byte{}, 0644); err != nil {
			log.Printf("[tray] create .env: %v", err)
			showMessageBox(".env Editor",
				fmt.Sprintf("Cannot create .env at:\n%s\n\n%v", envFilePath, err),
				mbIconWarning)
			return
		}
	}

	cmd := exec.Command("notepad.exe", envFilePath)
	if err := cmd.Run(); err != nil {
		log.Printf("[tray] notepad: %v", err)
		showMessageBox(".env Editor",
			fmt.Sprintf("Failed to open .env in Notepad:\n%v", err),
			mbIconWarning)
		return
	}

	if messageBoxYesNo("Restart required",
		"The .env file may have changed.\nRestart the program now to apply changes?") == idYes {
		restartApp()
	}
}

func messageBoxYesNo(title, text string) int {
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		log.Printf("[tray] message box title encode: %v", err)
		return 0
	}
	textPtr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		log.Printf("[tray] message box text encode: %v", err)
		return 0
	}
	ret, _, _ := procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		mbYesNo|mbIconQuestion|mbSystemModal|mbSetForeground,
	)
	return int(ret)
}

func restartApp() {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("[tray] resolve executable: %v", err)
		showMessageBox(".env Editor", "Failed to locate executable for restart.", mbIconWarning)
		return
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: detachedProcess | createNewProcessGroup,
	}
	if err := cmd.Start(); err != nil {
		log.Printf("[tray] restart: %v", err)
		showMessageBox(".env Editor",
			fmt.Sprintf("Failed to start new instance:\n%v", err),
			mbIconWarning)
		return
	}

	trayOnce.Do(func() {
		systray.Quit()
	})
}

func showMessageBox(title, text string, icon uintptr) {
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		log.Printf("[tray] message box title encode: %v", err)
		return
	}
	textPtr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		log.Printf("[tray] message box text encode: %v", err)
		return
	}
	procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		mbOk|icon|mbSystemModal|mbSetForeground,
	)
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
		iconPath = filepath.Join(filepath.Dir(envFilePath), "tray.ico")
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
