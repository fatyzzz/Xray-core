package profile

import (
	"context"

	"github.com/xtls/xray-core/app/observatory"
	"github.com/xtls/xray-core/app/observatory/burst"
	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/features/extension"
	"google.golang.org/protobuf/proto"
)

type Manager struct {
	defaultTag    string
	observatories map[string]extension.Observatory
	ordered       []extension.Observatory
}

func (m *Manager) Type() interface{} {
	return extension.ObservatoryType()
}

func (m *Manager) Start() error {
	for _, entry := range m.ordered {
		if err := entry.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) Close() error {
	var errs []error
	for i := len(m.ordered) - 1; i >= 0; i-- {
		if err := m.ordered[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Combine(errs...)
}

func (m *Manager) GetObservation(ctx context.Context) (proto.Message, error) {
	if m.defaultTag == "" {
		return nil, errors.New("default observatory is not configured")
	}
	return m.GetObservationByTag(ctx, m.defaultTag)
}

func (m *Manager) GetObservationByTag(ctx context.Context, tag string) (proto.Message, error) {
	entry, ok := m.observatories[tag]
	if !ok || entry == nil {
		return nil, errors.New("unknown observatory tag: ", tag)
	}
	return entry.GetObservation(ctx)
}

func newObservatory(ctx context.Context, cfg *Observatory) (extension.Observatory, error) {
	if cfg == nil {
		return nil, errors.New("observatory config is nil")
	}
	if cfg.PingConfig != nil {
		return burst.New(ctx, &burst.Config{
			SubjectSelector: cfg.SubjectSelector,
			PingConfig:      cfg.PingConfig,
		})
	}
	return observatory.New(ctx, &observatory.Config{
		SubjectSelector:   cfg.SubjectSelector,
		ProbeUrl:          cfg.ProbeUrl,
		ProbeInterval:     cfg.ProbeInterval,
		EnableConcurrency: cfg.EnableConcurrency,
	})
}

func New(ctx context.Context, config *Config) (*Manager, error) {
	if config == nil {
		return nil, errors.New("observatories config is nil")
	}
	if len(config.Observatory) == 0 {
		return nil, errors.New("observatories config is empty")
	}

	observatories := make(map[string]extension.Observatory, len(config.Observatory))
	ordered := make([]extension.Observatory, 0, len(config.Observatory))
	defaultTag := ""
	for _, cfg := range config.Observatory {
		if cfg == nil {
			continue
		}
		if defaultTag == "" {
			defaultTag = cfg.Tag
		}
		entry, err := newObservatory(ctx, cfg)
		if err != nil {
			return nil, errors.New("failed to build observatory ", cfg.Tag).Base(err)
		}
		observatories[cfg.Tag] = entry
		ordered = append(ordered, entry)
	}
	if defaultTag == "" || len(ordered) == 0 {
		return nil, errors.New("observatories config is empty")
	}

	return &Manager{
		defaultTag:    defaultTag,
		observatories: observatories,
		ordered:       ordered,
	}, nil
}

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return New(ctx, config.(*Config))
	}))
}
