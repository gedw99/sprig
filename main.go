package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"gioui.org/app"
	gioapp "gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/x/profiling"
	status "git.sr.ht/~athorp96/forest-ex/active-status"
	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/sprig/core"
	sprigTheme "git.sr.ht/~whereswaldon/sprig/widget/theme"
	"github.com/pkg/profile"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	go func() {
		w := app.NewWindow(app.Title("Sprig"))
		if err := eventLoop(w); err != nil {
			log.Fatalf("exiting due to error: %v", err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func eventLoop(w *app.Window) error {
	var (
		dataDir    string
		invalidate bool
		profileOpt string
	)

	dataDir, err := getDataDir("sprig")
	if err != nil {
		log.Printf("finding application data dir: %v", err)
	}

	flag.StringVar(&profileOpt, "profile", "none", "create the provided kind of profile. Use one of [none, cpu, mem, block, goroutine, mutex, trace, gio]")
	flag.BoolVar(&invalidate, "invalidate", false, "invalidate every single frame, only useful for profiling")
	flag.StringVar(&dataDir, "data-dir", dataDir, "application state directory")
	flag.Parse()

	profiler := ProfileOpt(profileOpt).NewProfiler()
	profiler.Start()
	defer profiler.Stop()

	app, err := core.NewApp(w, dataDir)
	if err != nil {
		log.Fatalf("Failed initializing application: %v", err)
	}

	go heartbeat(app)

	// handle ctrl+c to shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	vm := NewViewManager(w, app)
	vm.ApplySettings(app.Settings())
	vm.RegisterView(ReplyViewID, NewReplyListView(app))
	vm.RegisterView(ConnectFormID, NewConnectFormView(app))
	vm.RegisterView(SettingsID, NewCommunityMenuView(app))
	vm.RegisterView(IdentityFormID, NewIdentityFormView(app))
	vm.RegisterView(ConsentViewID, NewConsentView(app))

	if app.Settings().AcknowledgedNoticeVersion() < NoticeVersion {
		vm.RequestViewSwitch(ConsentViewID)
	} else if app.Settings().Address() == "" {
		vm.RequestViewSwitch(ConnectFormID)
	} else if app.Settings().ActiveArborIdentityID() == nil {
		vm.RequestViewSwitch(IdentityFormID)
	} else {
		vm.RequestViewSwitch(ReplyViewID)
	}

	var ops op.Ops
	for {
		select {
		case <-sigs:
			app.Shutdown()
			return nil
		case event := (<-w.Events()):
			switch event := event.(type) {
			case system.DestroyEvent:
				app.Shutdown()
				return event.Err
			case *system.CommandEvent:
				if event.Type == system.CommandBack {
					vm.HandleBackNavigation(event)
				}
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, event)
				if profiler.Recorder != nil {
					profiler.Record(gtx)
				}
				if invalidate {
					op.InvalidateOp{}.Add(gtx.Ops)
				}
				th := app.Theme().Current()
				layout.Stack{}.Layout(gtx,
					layout.Expanded(func(gtx C) D {
						return sprigTheme.Rect{
							Color: th.Background.Dark.Bg,
							Size: f32.Point{
								X: float32(gtx.Constraints.Max.X),
								Y: float32(gtx.Constraints.Max.Y),
							},
						}.Layout(gtx)
					}),
					layout.Stacked(func(gtx C) D {
						return layout.Inset{
							Bottom: event.Insets.Bottom,
							Left:   event.Insets.Left,
							Right:  event.Insets.Right,
							Top:    event.Insets.Top,
						}.Layout(gtx, func(gtx C) D {
							return layout.Stack{}.Layout(gtx,
								layout.Expanded(func(gtx C) D {
									return sprigTheme.Rect{
										Color: th.Background.Default.Bg,
										Size: f32.Point{
											X: float32(gtx.Constraints.Max.X),
											Y: float32(gtx.Constraints.Max.Y),
										},
									}.Layout(gtx)
								}),
								layout.Stacked(vm.Layout),
							)
						})
					}),
				)
				event.Frame(gtx.Ops)
			default:
				ProcessPlatformEvent(app, event)
			}
		}
	}
}

type ViewID int

const (
	ConnectFormID ViewID = iota
	IdentityFormID
	SettingsID
	ReplyViewID
	ConsentViewID
)

// getDataDir returns application specific file directory to use for storage.
// Suffix is joined to the path for convenience.
func getDataDir(suffix string) (string, error) {
	d, err := app.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, suffix), nil
}

// Profiler unifies the profiling api between Gio profiler and pkg/profile.
type Profiler struct {
	Starter  func(p *profile.Profile)
	Stopper  func()
	Recorder func(gtx C)
}

// Start profiling.
func (pfn *Profiler) Start() {
	if pfn.Starter != nil {
		pfn.Stopper = profile.Start(pfn.Starter).Stop
	}
}

// Stop profiling.
func (pfn *Profiler) Stop() {
	if pfn.Stopper != nil {
		pfn.Stopper()
	}
}

// Record GUI stats per frame.
func (pfn Profiler) Record(gtx C) {
	if pfn.Recorder != nil {
		pfn.Recorder(gtx)
	}
}

// ProfileOpt specifies the various profiling options.
type ProfileOpt string

const (
	None      ProfileOpt = "none"
	CPU       ProfileOpt = "cpu"
	Memory    ProfileOpt = "mem"
	Block     ProfileOpt = "block"
	Goroutine ProfileOpt = "goroutine"
	Mutex     ProfileOpt = "mutex"
	Trace     ProfileOpt = "trace"
	Gio       ProfileOpt = "gio"
)

// NewProfiler creates a profiler based on the selected option.
func (p ProfileOpt) NewProfiler() Profiler {
	switch p {
	case "", None:
		return Profiler{}
	case CPU:
		return Profiler{Starter: profile.CPUProfile}
	case Memory:
		return Profiler{Starter: profile.MemProfile}
	case Block:
		return Profiler{Starter: profile.BlockProfile}
	case Goroutine:
		return Profiler{Starter: profile.GoroutineProfile}
	case Mutex:
		return Profiler{Starter: profile.MutexProfile}
	case Trace:
		return Profiler{Starter: profile.TraceProfile}
	case Gio:
		var (
			recorder *profiling.CSVTimingRecorder
			err      error
		)
		return Profiler{
			Starter: func(*profile.Profile) {
				recorder, err = profiling.NewRecorder(nil)
				if err != nil {
					log.Printf("starting profiler: %v", err)
				}
			},
			Stopper: func() {
				if recorder == nil {
					return
				}
				if err := recorder.Stop(); err != nil {
					log.Printf("stopping profiler: %v", err)
				}
			},
			Recorder: func(gtx C) {
				if recorder == nil {
					return
				}
				recorder.Profile(gtx)
			},
		}
	}
	return Profiler{}
}

// heartbeat starts the active status heartbeat.
func heartbeat(app core.App) {
	app.Arbor().Communities().WithCommunities(func(c []*forest.Community) {
		if app.Settings().ActiveArborIdentityID() != nil {
			builder, err := app.Settings().Builder()
			if err == nil {
				log.Printf("Begining active-status heartbeat")
				go status.StartActivityHeartBeat(app.Arbor().Store(), c, builder, time.Minute*5)
			} else {
				log.Printf("Could not acquire builder: %v", err)
			}
		}
	})
	if windower, ok := app.(interface{ Window() *gioapp.Window }); ok {
		app.Arbor().Store().SubscribeToNewMessages(func(n forest.Node) {
			windower.Window().Invalidate()
		})
	}
}
