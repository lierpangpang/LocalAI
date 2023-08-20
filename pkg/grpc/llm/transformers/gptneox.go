package transformers

// This is a wrapper to statisfy the GRPC service interface
// It is meant to be used by the main executable that is the server for the specific backend type (falcon, gpt3, etc)
import (
	"fmt"

	"github.com/go-skynet/LocalAI/pkg/grpc/base"
	pb "github.com/go-skynet/LocalAI/pkg/grpc/proto"
	"github.com/rs/zerolog/log"

	transformers "github.com/go-skynet/go-ggml-transformers.cpp"
)

type GPTNeoX struct {
	base.BaseSingleton

	gptneox *transformers.GPTNeoX
}

func (llm *GPTNeoX) Load(opts *pb.ModelOptions) error {
	if llm.Base.State != pb.StatusResponse_UNINITIALIZED {
		log.Warn().Msgf("gptneox backend loading %s while already in state %s!", opts.Model, llm.Base.State.String())
	}

	model, err := transformers.NewGPTNeoX(opts.ModelFile)
	llm.gptneox = model
	return err
}

func (llm *GPTNeoX) Predict(opts *pb.PredictOptions) (string, error) {
	return llm.gptneox.Predict(opts.Prompt, buildPredictOptions(opts)...)
}

// fallback to Predict
func (llm *GPTNeoX) PredictStream(opts *pb.PredictOptions, results chan string) error {
	go func() {
		res, err := llm.gptneox.Predict(opts.Prompt, buildPredictOptions(opts)...)

		if err != nil {
			fmt.Println("err: ", err)
		}
		results <- res
		close(results)
	}()
	return nil
}
