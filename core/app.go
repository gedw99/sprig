package core

import (
	"fmt"
)

// App bundles core application services into a single convenience type.
type App interface {
	Notifications() NotificationService
	Arbor() ArborService
	Settings() SettingsService
	Sprout() SproutService
}

// app bundles services together.
type app struct {
	NotificationService
	SettingsService
	ArborService
	SproutService
}

var _ App = &app{}

// NewApp constructs an App or fails with an error. This process will fail
// if any of the application services fail to initialize correctly.
func NewApp(stateDir string) (application App, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed constructing app: %w", err)
		}
	}()
	a := &app{}

	// Instantiate all of the services.
	// Settings must be initialized first, as other services rely on derived
	// values from it
	if a.SettingsService, err = newSettingsService(stateDir); err != nil {
		return nil, err
	}
	if a.ArborService, err = newArborService(a.SettingsService); err != nil {
		return nil, err
	}
	if a.NotificationService, err = newNotificationService(a.SettingsService, a.ArborService); err != nil {
		return nil, err
	}
	if a.SproutService, err = newSproutService(a.ArborService); err != nil {
		return nil, err
	}

	// Connect services together
	if addr := a.Settings().Address(); addr != "" {
		a.Sprout().ConnectTo(addr)
	}
	a.Notifications().Register(a.Arbor().Store())

	return a, nil
}

// Settings returns the app's settings service implementation.
func (a *app) Settings() SettingsService {
	return a.SettingsService
}

// Arbor returns the app's arbor service implementation.
func (a *app) Arbor() ArborService {
	return a.ArborService
}

// Notifications returns the app's notification service implementation.
func (a *app) Notifications() NotificationService {
	return a.NotificationService
}

// Sprout returns the app's sprout service implementation.
func (a *app) Sprout() SproutService {
	return a.SproutService
}
