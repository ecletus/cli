package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/ecletus/plug"
	"github.com/moisespsena-go/default-logger"
	"github.com/spf13/cobra"
)

var (
	E_INIT     = PREFIX + ":init"
	E_REGISTER = PREFIX + ":register"
	log        = defaultlogger.NewLogger(PREFIX)
)

type Plugin struct {
	plug.EventDispatcher
}

type Event struct {
	plug.PluginEventInterface
	CLI *CLI
}

type RegisterEvent struct {
	plug.PluginEventInterface
	RootCmd *cobra.Command
	CLI     *CLI
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

type CLI struct {
	Dis                        plug.PluginEventDispatcherInterface
	RootCmd                    cobra.Command
	initCalled, registerCalled bool
	args                       []string
	cmdName                    string
	doneCalled                 bool
	doneCallbacks              []func()
	Stderr                     io.Writer
}

func (c *CLI) OnDone(cb ...func()) {
	c.doneCallbacks = append(c.doneCallbacks, cb...)
}

func (c *CLI) Done() {
	if c.doneCalled {
		return
	}
	c.doneCalled = true
	if c.doneCallbacks != nil {
		log.Debug("calling done callbacks")
		for i, cb := range c.doneCallbacks {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Error("done callback #%d failed: %v", i, r)
					}
				}()
				cb()
			}()
		}
	}
}

func (c *CLI) SetArgs(args []string) *CLI {
	c.args = args
	return c
}

func (c *CLI) Args() (args []string) {
	return c.args
}

func (c *CLI) SysArgs() *CLI {
	c.RootCmd.Use = os.Args[0]
	c.args = os.Args[1:]
	return c
}

func (c *CLI) TriggerInit() error {
	return c.Dis.TriggerPlugins(&Event{plug.NewPluginEvent(E_INIT), c})
}

func (c *CLI) Init() error {
	if !c.initCalled {
		c.initCalled = true
		return c.TriggerInit()
	}
	return nil
}

func (c *CLI) TriggerRegister() error {
	return c.Dis.TriggerPlugins(&RegisterEvent{plug.NewPluginEvent(E_REGISTER), &c.RootCmd, c})
}

func (c *CLI) Register() error {
	if !c.registerCalled {
		c.registerCalled = true
		return c.TriggerRegister()
	}
	return nil
}

func (c *CLI) Execute() (err error) {
	if c.RootCmd.Use == "" {
		c.SysArgs()
	}

	defer c.Done()

	if !c.initCalled {
		err = c.Init()
		if err != nil {
			return
		}
	}

	c.RootCmd.SetArgs(c.args)

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

func (c *CLI) ExecuteAlone() {
	if err := c.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
