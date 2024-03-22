package e2e_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/sashabaranov/go-openai"
)

var pool *dockertest.Pool
var resource *dockertest.Resource
var client *openai.Client

var containerImage = os.Getenv("LOCALAI_IMAGE")
var containerImageTag = os.Getenv("LOCALAI_IMAGE_TAG")
var modelsDir = os.Getenv("LOCALAI_MODELS_DIR")
var apiPort = os.Getenv("LOCALAI_API_PORT")

func TestLocalAI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LocalAI E2E test suite")
}

var _ = BeforeSuite(func() {

	if containerImage == "" {
		Fail("LOCALAI_IMAGE is not set")
	}
	if containerImageTag == "" {
		Fail("LOCALAI_IMAGE_TAG is not set")
	}
	if apiPort == "" {
		apiPort = "8080"
	}

	p, err := dockertest.NewPool("")
	Expect(err).To(Not(HaveOccurred()))
	Expect(p.Client.Ping()).To(Succeed())

	pool = p

	// get cwd
	cwd, err := os.Getwd()
	Expect(err).To(Not(HaveOccurred()))
	md := cwd + "/models"

	if modelsDir != "" {
		md = modelsDir
	}

	proc := runtime.NumCPU()
	options := &dockertest.RunOptions{
		Repository: containerImage,
		Tag:        containerImageTag,
		//	Cmd:        []string{"server", "/data"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"8080/tcp": []docker.PortBinding{{HostPort: apiPort}},
		},
		Env:    []string{"MODELS_PATH=/models", "DEBUG=true", "THREADS=" + fmt.Sprint(proc)},
		Mounts: []string{md + ":/models"},
	}

	r, err := pool.RunWithOptions(options)
	Expect(err).To(Not(HaveOccurred()))

	resource = r

	defaultConfig := openai.DefaultConfig("")
	defaultConfig.BaseURL = "http://localhost:" + apiPort + "/v1"

	// Wait for API to be ready
	client = openai.NewClientWithConfig(defaultConfig)

	Eventually(func() error {
		_, err := client.ListModels(context.TODO())
		return err
	}, "20m").ShouldNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(pool.Purge(resource)).To(Succeed())
	//dat, err := os.ReadFile(resource.Container.LogPath)
	//Expect(err).To(Not(HaveOccurred()))
	//Expect(string(dat)).To(ContainSubstring("GRPC Service Ready"))
	//fmt.Println(string(dat))
})

var _ = AfterEach(func() {
	//Expect(dbClient.Clear()).To(Succeed())
})
