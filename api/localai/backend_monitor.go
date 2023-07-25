package localai

import (
	"fmt"
	"strings"

	config "github.com/go-skynet/LocalAI/api/config"

	"github.com/go-skynet/LocalAI/api/options"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	gopsutil "github.com/shirou/gopsutil/v3/process"
)

type BackendMonitorRequest struct {
	Model string `json:"model" yaml:"model"`
}

type BackendMonitorResponse struct {
	MemoryInfo    *gopsutil.MemoryInfoStat
	MemoryPercent float32
	CPUPercent    float64
}

type BackendMonitor struct {
	configLoader *config.ConfigLoader
	options      *options.Option // Taking options in case we need to inspect ExternalGRPCBackends, though that's out of scope for now, hence the name.
}

func NewBackendMonitor(configLoader *config.ConfigLoader, options *options.Option) BackendMonitor {
	return BackendMonitor{
		configLoader: configLoader,
		options:      options,
	}
}

func (bm *BackendMonitor) SampleBackend(model string) (*BackendMonitorResponse, error) {
	config, exists := bm.configLoader.GetConfig(model)
	var backend string
	if exists {
		backend = config.Model
	} else {
		// Last ditch effort: use it raw, see if a backend happens to match.
		backend = model
	}

	if !strings.HasSuffix(backend, ".bin") {
		backend = fmt.Sprintf("%s.bin", backend)
	}

	pid, err := bm.options.Loader.GetGRPCPID(backend)

	if err != nil {
		log.Error().Msgf("model %s : failed to find pid %+v", model, err)
		return nil, err
	}

	// Name is slightly frightening but this does _not_ create a new process, rather it looks up an existing process by PID.
	backendProcess, err := gopsutil.NewProcess(int32(pid))

	if err != nil {
		log.Error().Msgf("model %s [PID %d] : error getting process info %+v", model, pid, err)
		return nil, err
	}

	memInfo, err := backendProcess.MemoryInfo()

	if err != nil {
		log.Error().Msgf("model %s [PID %d] : error getting memory info %+v", model, pid, err)
		return nil, err
	}

	memPercent, err := backendProcess.MemoryPercent()
	if err != nil {
		log.Error().Msgf("model %s [PID %d] : error getting memory percent %+v", model, pid, err)
		return nil, err
	}

	cpuPercent, err := backendProcess.CPUPercent()
	if err != nil {
		log.Error().Msgf("model %s [PID %d] : error getting cpu percent %+v", model, pid, err)
		return nil, err
	}

	return &BackendMonitorResponse{
		MemoryInfo:    memInfo,
		MemoryPercent: memPercent,
		CPUPercent:    cpuPercent,
	}, nil
}

func BackendMonitorEndpoint(bm BackendMonitor) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		input := new(BackendMonitorRequest)
		// Get input data from the request body
		if err := c.BodyParser(input); err != nil {
			return err
		}

		val, err := bm.SampleBackend(input.Model)
		if err != nil {
			log.Warn().Msgf("backend monitor (currently only supports local node grpc backends) error during %s, %+v", input.Model, err)
			return err
		}

		return c.JSON(val)
	}
}
