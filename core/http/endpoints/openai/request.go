package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-skynet/LocalAI/core/config"
	"github.com/go-skynet/LocalAI/core/http/ctx"
	"github.com/go-skynet/LocalAI/core/schema"
	"github.com/go-skynet/LocalAI/pkg/model"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func readRequest(c *fiber.Ctx, ml *model.ModelLoader, o *config.ApplicationConfig, firstModel bool) (string, *schema.OpenAIRequest, error) {
	input := new(schema.OpenAIRequest)

	// Get input data from the request body
	if err := c.BodyParser(input); err != nil {
		return "", nil, fmt.Errorf("failed parsing request body: %w", err)
	}

	received, _ := json.Marshal(input)

	context, cancel := context.WithCancel(o.Context)
	input.Context = context
	input.Cancel = cancel

	log.Debug().Msgf("Request received: %s", string(received))

	// TEMPORARY STUB DURING DEVELOPMENT
	fce := ctx.NewFiberContentExtractor(ml, o)

	modelFile, err := fce.ModelFromContext(c, input.Model, firstModel)

	return modelFile, input, err
}

func mergeRequestWithConfig(modelFile string, input *schema.OpenAIRequest, cm *config.BackendConfigLoader, loader *model.ModelLoader, debug bool, threads, ctx int, f16 bool) (*config.BackendConfig, *schema.OpenAIRequest, error) {

	cfg, err := cm.LoadBackendConfigFileByName(modelFile, loader.ModelPath,
		config.LoadOptionDebug(debug),
		config.LoadOptionThreads(threads),
		config.LoadOptionContextSize(ctx),
		config.LoadOptionF16(f16),
	)

	// Set the parameters for the language model prediction
	cfg.UpdateFromOpenAIRequest(input)

	return cfg, input, err
}
