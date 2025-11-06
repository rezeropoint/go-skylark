package engine

import (
	"context"

	"github.com/rezeropoint/go-skylark/core"
	"github.com/rezeropoint/go-skylark/internal/cache"
	"github.com/rezeropoint/go-skylark/internal/flows"
	"github.com/rezeropoint/go-skylark/internal/forms"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type skylarkEngine struct {
	cache *cache.SkylarkCache
	flows flows.SkylarkFlowRegistry
	forms forms.SkylarkFormRegistry
}

// newSkylarkRegistry 创建新的流程注册表
func newSkylarkEngine(config *Config, redisClient *redis.Redis) (*skylarkEngine, error) {
	if config == nil {
		return nil, core.ErrConfigNil
	}

	cache := cache.NewSkylarkCache(redisClient, config.Cache)

	flows, err := flows.NewSkylarkFlowRegistry(&flows.Config{}, cache)
	if err != nil {
		return nil, err
	}

	forms, err := forms.NewSkylarkFormRegistry(&forms.Config{}, cache)
	if err != nil {
		return nil, err
	}

	return &skylarkEngine{
		cache: cache,
		flows: flows,
		forms: forms,
	}, nil
}

func (e *skylarkEngine) CreateFlow(ctx context.Context, app string, flowID int64, userID int64, authHeader string, data map[string]core.TypedValue) error {
	return e.flows.CreateFlow(ctx, app, flowID, userID, authHeader, data)
}

func (e *skylarkEngine) CreateFormRow(ctx context.Context, app string, formID int64, userID int64, authHeader string, data map[string]core.TypedValue) error {
	return e.forms.CreateFormRow(ctx, app, formID, userID, authHeader, data)
}

func (e *skylarkEngine) UpdateFlowJourneyStatus(ctx context.Context, app string, flowID int64, journeyID int64, assignmentID int64, userID int64, authHeader string, operation string, options flows.UpdateJourneyStatusOptions) error {
	return e.flows.UpdateJourneyStatus(ctx, app, flowID, journeyID, assignmentID, userID, authHeader, operation, options)
}
