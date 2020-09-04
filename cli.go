package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/ecletus/plug"
	"github.com/moisespsena-go/default-logger"
	"github.com/moisespsena-go/task"
	"github.com/spf13/cobra"

	"github.com/ecletus/ecletus"
)

var (
	E_INIT     = PREFIX + ":init"
	E_REGISTER = PREFIX + ":register"
	log        = defaultlogger.GetOrCreateLogger(PREFIX)
)

type Plugin struct {
	CliKey string
	Cli *Cli
}

func (this Plugin) ProvideOptions() []string {
	return []string{this.CliKey}
}

func (this *Plugin) Provides(options *plug.Options) {
	options.Set(this.CliKey, this.Cli)
}

type Event struct {
	plug.PluginEventInterface
	CLI *Cli
}

type RegisterEvent struct {
	plug.PluginEventInterface
	RootCmd *cobra.Command
	CLI     *Cli
}

func OnRegisterE(dis plug.EventDispatcherInterface, cb func(e *RegisterEvent) error) error {
	return dis.OnE(E_REGISTER, func(e plug.PluginEventInterface) error {
		return cb(e.(*RegisterEvent))
	})
}

func OnRegister(dis plug.EventDispatcherInterface, cb func(e *RegisterEvent)) {
	dis.On(E_REGISTER, func(e plug.PluginEventInterface) {
		cb(e.(*RegisterEvent))
	})
}

type CliOptions struct {
	RootCmd *cobra.Command
	Stderr io.Writer
	Args []string
}

type Cli struct {
	Ecletus                    *ecletus.Ecletus
	RootCmd                    *cobra.Command
	initCalled, 
	registerCalled bool
	args                       []string
	cmdName                    string
	doneCalled                 bool
	InitCallbacks, 
	DoneCallbacks,
	RegisterCallbacks []func(cli *Cli) error
	Stderr                     io.Writer
}

func New(ecl *ecletus.Ecletus, opt ...CliOptions) *Cli {
	var opts CliOptions
	for _, opts = range opt{}
	if opts.RootCmd == nil {
		opts.RootCmd = &cobra.Command{}
	}
	if opts.Args == nil {
		opts.Args = os.Args
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.RootCmd.Use == "" {
		opts.RootCmd.Use = opts.Args[0]
		opts.Args = opts.Args[1:]
	}
	opts.RootCmd.SetArgs(opts.Args)

	return &Cli{Ecletus: ecl, RootCmd: opts.RootCmd, Stderr: opts.Stderr, args: opts.Args}
}

func (c *Cli) OnDone(cb ...func(cli *Cli) error) {
	c.DoneCallbacks = append(c.DoneCallbacks, cb...)
}

func (c *Cli) OnRegister(cb ...func(cli *Cli) error) {
	c.RegisterCallbacks = append(c.RegisterCallbacks, cb...)
}

func (c *Cli) OnInit(cb ...func(cli *Cli) error) {
	c.InitCallbacks = append(c.InitCallbacks, cb...)
}

func (c *Cli) Done() {
	if c.doneCalled {
		return
	}
	c.doneCalled = true
	if c.DoneCallbacks != nil {
		log.Debug("calling done callbacks")
		for i, cb := range c.DoneCallbacks {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Error("done callback #%d panics: %v", i, r)
					}
				}()
				if err := cb(c); err != nil {
					log.Error("done callback #%d failed: %v", i, err)
				}
			}()
		}
	}
}

func (c *Cli) TriggerInit() (err error) {
	for _, f := range c.InitCallbacks {
		if err = f(c); err != nil {
			return
		}
	}
	return c.Ecletus.Container.Plugins.TriggerPlugins(&Event{plug.NewPluginEvent(E_INIT), c})
}

func (c *Cli) Init() (error) {
	if !c.initCalled {
		if err := c.Ecletus.Plugins().Add(&Plugin{}); err != nil {
			return err
		}
		c.initCalled = true
		return c.TriggerInit()
	}
	return nil
}

func (c *Cli) TriggerRegister() (err error) {
	for _, f := range c.RegisterCallbacks {
		if err = f(c); err != nil {
			return
		}
	}
	return c.Ecletus.Container.Plugins.TriggerPlugins(&RegisterEvent{plug.NewPluginEvent(E_REGISTER), c.RootCmd, c})
}

func (c *Cli) Register() (error) {
	if !c.registerCalled {
		c.registerCalled = true
		return c.TriggerRegister()
	}
	return nil
}

func (c *Cli) Execute() (err error) {
	defer c.Done()

	if !c.initCalled {
		err = c.Init()
		if err != nil {
			return
		}
	}
	if !c.registerCalled {
		err = c.Register()
		if err != nil {
			return
		}
	}
	if err = c.RootCmd.Execute(); err != nil {
		if c.Stderr != nil {
			fmt.Fprintln(c.Stderr, err)
			return nil
		}
	}
	return 
}

type CliTask struct {
	Cli *Cli
}

func NewTask(cli *Cli) *CliTask {
	return &CliTask{Cli: cli}
}

func (this *CliTask) Setup(ta task.Appender) (err error) {
	if err = this.Cli.Execute(); err != nil {
		return 
	}
	return this.Cli.Ecletus.Setup(ta)
}

func (this *CliTask) Start(done func()) (stop task.Stoper, err error) {
	return this.Cli.Ecletus.Start(done)
}

func (this *CliTask) Run() (err error) {
	return this.Cli.Ecletus.Run()
}