//go:build windows

package cmd

import (
	"github.com/lxn/walk"

	"github.com/imgk/runcat-go"
)

func Run() error {
	// New main window
	mw, err := walk.NewMainWindow()
	if err != nil {
		return err
	}

	// New NotifyIcon
	ni, err := walk.NewNotifyIcon(mw)
	if err != nil {
		return err
	}
	defer ni.Dispose()

	// Set tooltip
	if err := ni.SetToolTip("RunCat for Windows"); err != nil {
		return err
	}

	errCh := make(chan error, 2)

	// Add "Exit" action
	{
		action := walk.NewAction()
		if err := action.SetText("E&xit"); err != nil {
			return err
		}
		action.Triggered().Attach(func() {
			errCh <- nil
			walk.App().Exit(0)
		})
		if err := ni.ContextMenu().Actions().Add(action); err != nil {
			return err
		}
	}

	// Set the tray visible
	if err := ni.SetVisible(true); err != nil {
		return err
	}

	// Get icon and update it
	go func(ni *walk.NotifyIcon, errCh chan error) {
		for {
			icon := runcat.GetNextIcon()
			if err := ni.SetIcon(icon); err != nil {
				errCh <- err
				walk.App().Exit(0)
				break
			}
		}
	}(ni, errCh)

	// Run the message loog
	mw.Run()

	select {
	case err := <-errCh:
		return err
	default:
	}

	return nil
}
