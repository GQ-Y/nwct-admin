package frpswrap

import (
	"context"
	"fmt"

	"github.com/fatedier/frp/pkg/config"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/server"
)

type Runner struct {
	svc *server.Service
}

func LoadAndValidate(cfgFile string) (*v1.ServerConfig, error) {
	// v0.65.0 开始支持 strict_config；默认严格模式更安全
	svrCfg, _, err := config.LoadServerConfig(cfgFile, true)
	if err != nil {
		return nil, err
	}
	if warning, err := validation.ValidateServerConfig(svrCfg); warning != nil {
		// 保持与 frps CLI 一致：warning 不阻断
		_ = warning
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return svrCfg, nil
}

func NewRunner(cfg *v1.ServerConfig) (*Runner, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}
	log.InitLogger(cfg.Log.To, cfg.Log.Level, int(cfg.Log.MaxDays), cfg.Log.DisablePrintColor)
	svc, err := server.NewService(cfg)
	if err != nil {
		return nil, err
	}
	return &Runner{svc: svc}, nil
}

func (r *Runner) Run(ctx context.Context) {
	r.svc.Run(ctx)
}
