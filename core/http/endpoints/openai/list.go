package openai

import (
	"regexp"

	"github.com/go-skynet/LocalAI/core/services"
	"github.com/go-skynet/LocalAI/pkg/datamodel"
	"github.com/go-skynet/LocalAI/pkg/model"
	"github.com/gofiber/fiber/v2"
)

func ListModelsEndpoint(cl *services.ConfigLoader, ml *model.ModelLoader) func(ctx *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		models, err := ml.ListModels()
		if err != nil {
			return err
		}
		var mm map[string]interface{} = map[string]interface{}{}

		dataModels := []datamodel.OpenAIModel{}

		var filterFn func(name string) bool
		filter := c.Query("filter")

		// If filter is not specified, do not filter the list by model name
		if filter == "" {
			filterFn = func(_ string) bool { return true }
		} else {
			// If filter _IS_ specified, we compile it to a regex which is used to create the filterFn
			rxp, err := regexp.Compile(filter)
			if err != nil {
				return err
			}
			filterFn = func(name string) bool {
				return rxp.MatchString(name)
			}
		}

		// By default, exclude any loose files that are already referenced by a configuration file.
		excludeConfigured := c.QueryBool("excludeConfigured", true)

		// Start with the known configurations
		for _, c := range cl.GetAllConfigs() {
			if excludeConfigured {
				mm[c.Model] = nil
			}

			if filterFn(c.Name) {
				dataModels = append(dataModels, datamodel.OpenAIModel{ID: c.Name, Object: "model"})
			}
		}

		// Then iterate through the loose files:
		for _, m := range models {
			// And only adds them if they shouldn't be skipped.
			if _, exists := mm[m]; !exists && filterFn(m) {
				dataModels = append(dataModels, datamodel.OpenAIModel{ID: m, Object: "model"})
			}
		}

		return c.JSON(struct {
			Object string                  `json:"object"`
			Data   []datamodel.OpenAIModel `json:"data"`
		}{
			Object: "list",
			Data:   dataModels,
		})
	}
}
