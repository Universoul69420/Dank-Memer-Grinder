package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/BridgeSenseDev/Dank-Memer-Grinder/config"
	"github.com/BridgeSenseDev/Dank-Memer-Grinder/instance"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx       context.Context
	cfg       *config.Config
	wg        *sync.WaitGroup
	instances []*instance.Instance
	wsMutex   sync.Mutex
}

func NewApp() *App {
	return &App{
		wg:      &sync.WaitGroup{},
		wsMutex: sync.Mutex{},
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load configuration
	configFile := "./config.json"
	cfg, err := config.ReadConfig(configFile)
	if err != nil {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "A fatal error occured!",
			Message: fmt.Sprintf("Failed to read config file: %s", err.Error()),
		})
		panic(fmt.Sprintf("Failed to read config file: %s", err.Error()))
	}

	a.cfg = &cfg
}

func (a *App) domReady(ctx context.Context) {
	if a.cfg != nil {
		a.StartInstances()
	}
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

func (a *App) shutdown(ctx context.Context) {
}

func (a *App) GetConfig() *config.Config {
	return a.cfg
}

func (a *App) UpdateConfig(newCfg *config.Config) error {

	a.cfg = newCfg

	configJSON, err := json.MarshalIndent(newCfg, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile("./config.json", configJSON, 0644)
	if err != nil {
		return err
	}

	for _, instance := range a.instances {
		instance.UpdateConfig(*newCfg)
	}

	return nil
}

func (a *App) UpdateInstanceToken(oldToken string, newToken string) {
	for _, in := range a.instances {
		if in.AccountCfg.Token == oldToken {
			in.AccountCfg.Token = newToken
			break
		}
	}
}

func (a *App) UpdateDiscordStatus(status string) {
	a.wsMutex.Lock()
	defer a.wsMutex.Unlock()

	for _, in := range a.instances {
		if in == nil || in.User.Status == status {
			continue
		}

		payload := map[string]interface{}{
			"op": 3,
			"d": map[string]interface{}{
				"since":      0,
				"activities": []map[string]interface{}{},
				"status":     status,
				"afk":        false,
			},
		}

		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			log.Error().Msgf("Error marshaling payload: %s", err)
			continue
		}

		err = in.Client.SendMessage(payloadJSON)
		if err != nil {
			log.Error().Msgf("Error setting Discord status: %s", err)
		}

		in.User.Status = status
	}
}
