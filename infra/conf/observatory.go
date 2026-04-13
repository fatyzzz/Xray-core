package conf

import (
	"encoding/json"

	"github.com/xtls/xray-core/app/observatory"
	"github.com/xtls/xray-core/app/observatory/burst"
	observatoryprofile "github.com/xtls/xray-core/app/observatory/profile"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/infra/conf/cfgcommon/duration"
	"google.golang.org/protobuf/proto"
)

type ObservatoryConfig struct {
	SubjectSelector   []string          `json:"subjectSelector"`
	ProbeURL          string            `json:"probeURL"`
	ProbeInterval     duration.Duration `json:"probeInterval"`
	EnableConcurrency bool              `json:"enableConcurrency"`
}

func (o *ObservatoryConfig) Build() (proto.Message, error) {
	return &observatory.Config{SubjectSelector: o.SubjectSelector, ProbeUrl: o.ProbeURL, ProbeInterval: int64(o.ProbeInterval), EnableConcurrency: o.EnableConcurrency}, nil
}

type BurstObservatoryConfig struct {
	SubjectSelector []string `json:"subjectSelector"`
	// health check settings
	HealthCheck *healthCheckSettings `json:"pingConfig,omitempty"`
}

func (b BurstObservatoryConfig) Build() (proto.Message, error) {
	if b.HealthCheck == nil {
		return nil, errors.New("BurstObservatory requires a valid pingConfig")
	}
	if result, err := b.HealthCheck.Build(); err == nil {
		return &burst.Config{SubjectSelector: b.SubjectSelector, PingConfig: result.(*burst.HealthPingConfig)}, nil
	} else {
		return nil, err
	}
}

type TaggedObservatoryConfig struct {
	Tag               string               `json:"tag"`
	SubjectSelector   []string             `json:"subjectSelector"`
	ProbeURL          string               `json:"probeURL,omitempty"`
	ProbeInterval     duration.Duration    `json:"probeInterval,omitempty"`
	EnableConcurrency bool                 `json:"enableConcurrency,omitempty"`
	HealthCheck       *healthCheckSettings `json:"pingConfig,omitempty"`
}

type ObservatoriesConfig struct {
	Observatories []*TaggedObservatoryConfig `json:"observatories"`
}

func (o *ObservatoriesConfig) UnmarshalJSON(data []byte) error {
	var observatories []*TaggedObservatoryConfig
	if err := json.Unmarshal(data, &observatories); err == nil {
		o.Observatories = observatories
		return nil
	}

	type observatoriesAlias ObservatoriesConfig
	var wrapped observatoriesAlias
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return err
	}
	o.Observatories = wrapped.Observatories
	return nil
}

func (o *ObservatoriesConfig) Build() (proto.Message, error) {
	config := &observatoryprofile.Config{}
	if len(o.Observatories) == 0 {
		return nil, errors.New("observatories requires at least one entry")
	}
	for _, item := range o.Observatories {
		if item == nil || item.Tag == "" {
			return nil, errors.New("observatories requires tag")
		}
		if item.HealthCheck != nil && (item.ProbeURL != "" || item.ProbeInterval != 0 || item.EnableConcurrency) {
			return nil, errors.New("observatory ", item.Tag, " cannot mix pingConfig with probeURL/probeInterval/enableConcurrency")
		}
		entry := &observatoryprofile.Observatory{
			Tag:               item.Tag,
			SubjectSelector:   item.SubjectSelector,
			ProbeUrl:          item.ProbeURL,
			ProbeInterval:     int64(item.ProbeInterval),
			EnableConcurrency: item.EnableConcurrency,
		}
		if item.HealthCheck != nil {
			raw, err := item.HealthCheck.Build()
			if err != nil {
				return nil, err
			}
			entry.PingConfig = raw.(*burst.HealthPingConfig)
		}
		config.Observatory = append(config.Observatory, entry)
	}
	return config, nil
}
