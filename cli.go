package cli

import (
	"fmt"
	"os"

	"path/filepath"

	"github.com/aghape/plug"
	"github.com/spf13/cobra"
)

var (
	E_INIT     = PREFIX + ":init"
	E_REGISTER = PREFIX + ":register"
)

type Plugin struct {
	plug.EventDispatcher
}

type Event struct {
	plug.PluginEventInterface
}

type RegisterEvent struct {
	plug.PluginEventInterface
	RootCmd *cobra.Command
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

func TriggerInit(dis plug.PluginEventDispatcherInterface) (err error) {
	return dis.TriggerPlugins(&Event{plug.NewPluginEvent(E_INIT)})
}

func TriggerRegister(dis plug.PluginEventDispatcherInterface, rootCmd *cobra.Command) (err error) {
	return dis.TriggerPlugins(&RegisterEvent{plug.NewPluginEvent(E_REGISTER), rootCmd})
}

func TriggerAll(dis plug.PluginEventDispatcherInterface, rootCmd *cobra.Command) (err error) {
	err = TriggerInit(dis)
	if err != nil {
		return
	}
	return TriggerAll(dis, rootCmd)
}

type CLI struct {
	Dis                        plug.PluginEventDispatcherInterface
	RootCmd                    *cobra.Command
	initCalled, registerCalled bool
}

func (c *CLI) Init() error {
	if !c.initCalled {
		c.initCalled = true
		return TriggerInit(c.Dis)
	}
	return nil
}

func (c *CLI) Register() error {
	if !c.registerCalled {
		c.registerCalled = true
		if c.RootCmd == nil {
			c.RootCmd = &cobra.Command{Use: filepath.Base(os.Args[0])}
		}
		return TriggerRegister(c.Dis, c.RootCmd)
	}
	return nil
}

func (c *CLI) Execute() (err error) {
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
	return c.RootCmd.Execute()
}

func (c *CLI) ExecuteAlone() {
	if err := c.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
