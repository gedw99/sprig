package main

import (
	"log"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	materials "gioui.org/x/component"
	"git.sr.ht/~whereswaldon/sprig/core"
	"git.sr.ht/~whereswaldon/sprig/icons"
	sprigWidget "git.sr.ht/~whereswaldon/sprig/widget"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
)

type SettingsView struct {
	manager ViewManager

	core.App

	widget.List
	ConnectionForm          sprigWidget.TextForm
	IdentityButton          widget.Clickable
	CommunityList           layout.List
	CommunityBoxes          []widget.Bool
	ProfilingSwitch         widget.Bool
	ThemeingSwitch          widget.Bool
	NotificationsSwitch     widget.Bool
	TestNotificationsButton widget.Clickable
	TestResults             string
	BottomBarSwitch         widget.Bool
	DockNavSwitch           widget.Bool
	DarkModeSwitch          widget.Bool
	UseOrchardStoreSwitch   widget.Bool
}

type Section struct {
	*material.Theme
	Heading string
	Items   []layout.Widget
}

var sectionItemInset = layout.UniformInset(unit.Dp(8))
var itemInset = layout.Inset{
	Left:   unit.Dp(8),
	Right:  unit.Dp(8),
	Top:    unit.Dp(2),
	Bottom: unit.Dp(2),
}

func (s Section) Layout(gtx C) D {
	items := make([]layout.FlexChild, len(s.Items)+1)
	items[0] = layout.Rigid(component.SubheadingDivider(s.Theme, s.Heading).Layout)
	for i := range s.Items {
		items[i+1] = layout.Rigid(s.Items[i])
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, items...)
}

type SimpleSectionItem struct {
	*material.Theme
	Control layout.Widget
	Context string
}

func (s SimpleSectionItem) Layout(gtx C) D {
	return layout.Inset{
		Top:    unit.Dp(4),
		Bottom: unit.Dp(4),
	}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return s.Control(gtx)
			}),
			layout.Rigid(func(gtx C) D {
				if s.Context == "" {
					return D{}
				}
				return itemInset.Layout(gtx, material.Body2(s.Theme, s.Context).Layout)
			}),
		)
	})
}

var _ View = &SettingsView{}

func NewCommunityMenuView(app core.App) View {
	c := &SettingsView{
		App: app,
	}
	c.List.Axis = layout.Vertical
	c.ConnectionForm.TextField.SetText(c.Settings().Address())
	c.ConnectionForm.TextField.SingleLine = true
	c.ConnectionForm.TextField.Submit = true
	return c
}

func (c *SettingsView) HandleIntent(intent Intent) {}

func (c *SettingsView) AppBarData() (bool, string, []materials.AppBarAction, []materials.OverflowAction) {
	return true, "Settings", []materials.AppBarAction{}, []materials.OverflowAction{}
}

func (c *SettingsView) NavItem() *materials.NavItem {
	return &materials.NavItem{
		Name: "Settings",
		Icon: icons.SettingsIcon,
	}
}

func (c *SettingsView) Update(gtx layout.Context) {
	settingsChanged := false
	for i := range c.CommunityBoxes {
		box := &c.CommunityBoxes[i]
		if box.Changed() {
			log.Println("updated")
		}
	}
	if c.IdentityButton.Clicked() {
		c.manager.RequestViewSwitch(IdentityFormID)
	}
	if c.ProfilingSwitch.Changed() {
		c.manager.SetProfiling(c.ProfilingSwitch.Value)
	}
	if c.ThemeingSwitch.Changed() {
		c.manager.SetThemeing(c.ThemeingSwitch.Value)
	}
	if c.ConnectionForm.Submitted() {
		c.Settings().SetAddress(c.ConnectionForm.TextField.Text())
		settingsChanged = true
		c.Sprout().ConnectTo(c.Settings().Address())
	}
	if c.NotificationsSwitch.Changed() {
		c.Settings().SetNotificationsGloballyAllowed(c.NotificationsSwitch.Value)
		settingsChanged = true
	}
	if c.TestNotificationsButton.Clicked() {
		err := c.Notifications().Notify("Testing!", "This is a test notification from sprig.")
		if err == nil {
			c.TestResults = "Sent without errors"
		} else {
			c.TestResults = "Failed: " + err.Error()
		}
	}
	if c.BottomBarSwitch.Changed() {
		c.Settings().SetBottomAppBar(c.BottomBarSwitch.Value)
		settingsChanged = true
	}
	if c.DockNavSwitch.Changed() {
		c.Settings().SetDockNavDrawer(c.DockNavSwitch.Value)
		settingsChanged = true
	}
	if c.DarkModeSwitch.Changed() {
		c.Settings().SetDarkMode(c.DarkModeSwitch.Value)
		settingsChanged = true
	}
	if c.UseOrchardStoreSwitch.Changed() {
		c.Settings().SetUseOrchardStore(c.UseOrchardStoreSwitch.Value)
		settingsChanged = true
	}
	if settingsChanged {
		c.manager.ApplySettings(c.Settings())
		go c.Settings().Persist()
	}
}

func (c *SettingsView) BecomeVisible() {
	c.ConnectionForm.TextField.SetText(c.Settings().Address())
	c.NotificationsSwitch.Value = c.Settings().NotificationsGloballyAllowed()
	c.BottomBarSwitch.Value = c.Settings().BottomAppBar()
	c.DockNavSwitch.Value = c.Settings().DockNavDrawer()
	c.DarkModeSwitch.Value = c.Settings().DarkMode()
	c.UseOrchardStoreSwitch.Value = c.Settings().UseOrchardStore()
}

func (c *SettingsView) Layout(gtx layout.Context) layout.Dimensions {
	sTheme := c.Theme().Current()
	theme := sTheme.Theme
	sections := []Section{
		{
			Heading: "Identity",
			Items: []layout.Widget{
				func(gtx C) D {
					if c.Settings().ActiveArborIdentityID() != nil {
						id, _ := c.Settings().Identity()
						return itemInset.Layout(gtx, sprigTheme.AuthorName(sTheme, string(id.Name.Blob), id.ID(), true).Layout)
					}
					return itemInset.Layout(gtx, material.Button(theme, &c.IdentityButton, "Create new Identity").Layout)
				},
			},
		},
		{
			Heading: "Connection",
			Items: []layout.Widget{
				SimpleSectionItem{
					Theme: theme,
					Control: func(gtx C) D {
						return itemInset.Layout(gtx, func(gtx C) D {
							form := sprigTheme.TextForm(sTheme, &c.ConnectionForm, "Connect", "HOST:PORT")
							return form.Layout(gtx)
						})
					},
					Context: "You can restart your connection to a relay by hitting the Connect button above without changing the address.",
				}.Layout,
			},
		},
		{
			Heading: "Notifications",
			Items: []layout.Widget{
				SimpleSectionItem{
					Theme: theme,
					Control: func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Switch(theme, &c.NotificationsSwitch).Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Body1(theme, "Enable notifications").Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Button(theme, &c.TestNotificationsButton, "Test").Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Body2(theme, c.TestResults).Layout)
							}),
						)
					},
					Context: "Currently supported on Android and Linux/BSD. macOS support coming soon.",
				}.Layout,
			},
		},
		{
			Heading: "Store",
			Items: []layout.Widget{
				SimpleSectionItem{
					Theme: theme,
					Control: func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Switch(theme, &c.UseOrchardStoreSwitch).Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Body1(theme, "Use Orchard store").Layout)
							}),
						)
					},
					Context: "Orchard is a single-file read-oriented database for storing nodes.",
				}.Layout,
			},
		},
		{
			Heading: "User Interface",
			Items: []layout.Widget{
				SimpleSectionItem{
					Theme: theme,
					Control: func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Switch(theme, &c.BottomBarSwitch).Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Body1(theme, "Use bottom app bar").Layout)
							}),
						)
					},
					Context: "Only recommended on mobile devices.",
				}.Layout,
				SimpleSectionItem{
					Theme: theme,
					Control: func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Switch(theme, &c.DockNavSwitch).Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Body1(theme, "Dock navigation to the left edge of the UI").Layout)
							}),
						)
					},
					Context: "Only recommended on desktop devices.",
				}.Layout,
				SimpleSectionItem{
					Theme: theme,
					Control: func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Switch(theme, &c.DarkModeSwitch).Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Body1(theme, "Dark Mode").Layout)
							}),
						)
					},
				}.Layout,
			},
		},
		{
			Heading: "Developer",
			Items: []layout.Widget{
				SimpleSectionItem{
					Theme: theme,
					Control: func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Switch(theme, &c.ProfilingSwitch).Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Body1(theme, "Display graphics profiling").Layout)
							}),
						)
					},
				}.Layout,
				SimpleSectionItem{
					Theme: theme,
					Control: func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Switch(theme, &c.ThemeingSwitch).Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return itemInset.Layout(gtx, material.Body1(theme, "Display theme editor").Layout)
							}),
						)
					},
				}.Layout,
				func(gtx C) D {
					return itemInset.Layout(gtx, material.Body1(theme, VersionString).Layout)
				},
			},
		},
	}
	return material.List(theme, &c.List).Layout(gtx, len(sections), func(gtx C, index int) D {
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx C) D {
			return component.Surface(theme).Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return itemInset.Layout(gtx, func(gtx C) D {
					sections[index].Theme = theme
					return sections[index].Layout(gtx)
				})
			})
		})
	})
}

func (c *SettingsView) SetManager(mgr ViewManager) {
	c.manager = mgr
}
