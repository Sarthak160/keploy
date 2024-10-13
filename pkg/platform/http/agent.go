//go:build !windows

// Client side code to communicate with the agent server
package http

import (
	"bytes"
	"context"
	_ "embed" // necessary for embedding
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/events"
	"go.keploy.io/server/v2/config"
	"go.keploy.io/server/v2/pkg/core/app"
	"go.keploy.io/server/v2/pkg/core/hooks"
	"go.keploy.io/server/v2/pkg/models"
	kdocker "go.keploy.io/server/v2/pkg/platform/docker"
	"go.keploy.io/server/v2/utils"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type AgentClient struct {
	logger       *zap.Logger
	dockerClient kdocker.Client //embedding the docker client to transfer the docker client methods to the core object
	apps         sync.Map
	client       http.Client
	conf         *config.Config
}

//go:embed assets/initStop.sh
var initStopScript []byte

func New(logger *zap.Logger, client kdocker.Client, c *config.Config) *AgentClient {

	return &AgentClient{
		logger:       logger,
		dockerClient: client,
		client:       http.Client{},
		conf:         c,
	}
}

func (a *AgentClient) GetIncoming(ctx context.Context, id uint64, opts models.IncomingOptions) (<-chan *models.TestCase, error) {
	requestBody := models.IncomingReq{
		IncomingOptions: opts,
		ClientID:        id,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		utils.LogError(a.logger, err, "failed to marshal request body for incoming request")
		return nil, fmt.Errorf("error marshaling request body for incoming request: %s", err.Error())
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost:%d/agent/incoming", a.conf.Agent.Port), bytes.NewBuffer(requestJSON))
	if err != nil {
		utils.LogError(a.logger, err, "failed to create request for incoming request")
		return nil, fmt.Errorf("error creating request for incoming request: %s", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	res, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get incoming: %s", err.Error())
	}

	// Ensure response body is closed when we're done
	go func() {
		<-ctx.Done()
		if res.Body != nil {
			_ = res.Body.Close()
		}
	}()

	// Create a channel to stream TestCase data
	tcChan := make(chan *models.TestCase)

	go func() {
		defer close(tcChan)
		defer func() {
			err := res.Body.Close()
			if err != nil {
				utils.LogError(a.logger, err, "failed to close response body for incoming request")
			}
		}()

		decoder := json.NewDecoder(res.Body)

		for {
			var testCase models.TestCase
			if err := decoder.Decode(&testCase); err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					// End of the stream
					break
				}
				utils.LogError(a.logger, err, "failed to decode test case from stream")
				break
			}

			select {
			case <-ctx.Done():
				// If the context is done, exit the loop
				return
			case tcChan <- &testCase:
				// Send the decoded test case to the channel
			}
		}
	}()

	return tcChan, nil
}

func (a *AgentClient) GetOutgoing(ctx context.Context, id uint64, opts models.OutgoingOptions) (<-chan *models.Mock, error) {
	requestBody := models.OutgoingReq{
		OutgoingOptions: opts,
		ClientID:        id,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		utils.LogError(a.logger, err, "failed to marshal request body for mock outgoing")
		return nil, fmt.Errorf("error marshaling request body for mock outgoing: %s", err.Error())
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost:%d/agent/outgoing", a.conf.Agent.Port), bytes.NewBuffer(requestJSON))
	if err != nil {
		utils.LogError(a.logger, err, "failed to create request for mock outgoing")
		return nil, fmt.Errorf("error creating request for mock outgoing: %s", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	res, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %s", err.Error())
	}

	// Create a channel to stream Mock data
	mockChan := make(chan *models.Mock)

	go func() {
		defer close(mockChan)
		defer func() {
			err := res.Body.Close()
			if err != nil {
				utils.LogError(a.logger, err, "failed to close response body for mock outgoing")
			}
		}()

		decoder := json.NewDecoder(res.Body)

		// Read from the response body as a stream
		for {
			var mock models.Mock
			if err := decoder.Decode(&mock); err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					// End of the stream
					break
				}
				utils.LogError(a.logger, err, "failed to decode mock from stream")
				// break, it will exit the loop if there is any decoding error from the stream
			}

			select {
			case <-ctx.Done():
				// If the context is done, exit the loop
				return
			case mockChan <- &mock:
			}
		}
	}()

	return mockChan, nil
}

func (a *AgentClient) MockOutgoing(ctx context.Context, id uint64, opts models.OutgoingOptions) error {
	// make a request to the server to mock outgoing
	requestBody := models.OutgoingReq{
		OutgoingOptions: opts,
		ClientID:        id,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		utils.LogError(a.logger, err, "failed to marshal request body for mock outgoing")
		return fmt.Errorf("error marshaling request body for mock outgoing: %s", err.Error())
	}

	// mock outgoing request
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost:%d/agent/mock", a.conf.Agent.Port), bytes.NewBuffer(requestJSON))
	if err != nil {
		utils.LogError(a.logger, err, "failed to create request for mock outgoing")
		return fmt.Errorf("error creating request for mock outgoing: %s", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	res, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request for mockOutgoing: %s", err.Error())
	}

	var mockResp models.AgentResp
	err = json.NewDecoder(res.Body).Decode(&mockResp)
	if err != nil {
		return fmt.Errorf("failed to decode response body for mock outgoing: %s", err.Error())
	}

	if mockResp.Error != nil {
		return mockResp.Error
	}

	return nil

}

func (a *AgentClient) SetMocks(ctx context.Context, id uint64, filtered []*models.Mock, unFiltered []*models.Mock) error {
	requestBody := models.SetMocksReq{
		Filtered:   filtered,
		UnFiltered: unFiltered,
		ClientID:   id,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		utils.LogError(a.logger, err, "failed to marshal request body for setmocks")
		return fmt.Errorf("error marshaling request body for setmocks: %s", err.Error())
	}

	// mock outgoing request
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("http://localhost:%d/agent/setmocks", a.conf.Agent.Port), bytes.NewBuffer(requestJSON))
	if err != nil {
		utils.LogError(a.logger, err, "failed to create request for setmocks outgoing")
		return fmt.Errorf("error creating request for set mocks: %s", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	res, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request for setmocks: %s", err.Error())
	}

	var mockResp models.AgentResp
	err = json.NewDecoder(res.Body).Decode(&mockResp)
	if err != nil {
		return fmt.Errorf("failed to decode response body for setmocks: %s", err.Error())
	}

	if mockResp.Error != nil {
		return mockResp.Error
	}

	return nil
}

func (a *AgentClient) GetConsumedMocks(ctx context.Context, id uint64) ([]string, error) {
	// Create the URL with query parameters
	url := fmt.Sprintf("http://localhost:%d/agent/consumedmocks?id=%d", a.conf.Agent.Port, id)

	// Create a new GET request with the query parameter
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %s", err.Error())
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request for mockOutgoing: %s", err.Error())
	}

	defer func() {
		err := res.Body.Close()
		if err != nil {
			utils.LogError(a.logger, err, "failed to close response body for getconsumedmocks")
		}
	}()

	var consumedMocks []string
	err = json.NewDecoder(res.Body).Decode(&consumedMocks)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response body: %s", err.Error())
	}

	return consumedMocks, nil
}

func (a *AgentClient) UnHook(_ context.Context, _ uint64) error {
	return nil
}

func (a *AgentClient) GetContainerIP(_ context.Context, id uint64) (string, error) {

	app, err := a.getApp(id)
	if err != nil {
		utils.LogError(a.logger, err, "failed to get app")
		return "", err
	}

	ip := app.ContainerIPv4Addr()
	a.logger.Debug("ip address of the target app container", zap.Any("ip", ip))
	if ip == "" {
		return "", fmt.Errorf("failed to get the IP address of the app container. Try increasing --delay (in seconds)")
	}

	return ip, nil
}

func (a *AgentClient) Run(ctx context.Context, id uint64, _ models.RunOptions) models.AppError {

	app, err := a.getApp(id)
	if err != nil {
		utils.LogError(a.logger, err, "failed to get app while running")
		return models.AppError{AppErrorType: models.ErrInternal, Err: err}
	}

	runAppErrGrp, runAppCtx := errgroup.WithContext(ctx)

	appErrCh := make(chan models.AppError, 1)

	defer func() {
		err := runAppErrGrp.Wait()

		if err != nil {
			utils.LogError(a.logger, err, "failed to stop the app")
		}
	}()

	runAppErrGrp.Go(func() error {
		defer utils.Recover(a.logger)
		defer close(appErrCh)
		appErr := app.Run(runAppCtx)
		if appErr.Err != nil {
			utils.LogError(a.logger, appErr.Err, "error while running the app")
			appErrCh <- appErr
		}
		return nil
	})

	select {
	case <-runAppCtx.Done():
		return models.AppError{AppErrorType: models.ErrCtxCanceled, Err: nil}
	case appErr := <-appErrCh:
		return appErr
	}
}

func (a *AgentClient) Setup(ctx context.Context, cmd string, opts models.SetupOptions) (uint64, error) {

	// clientID := utils.GenerateID()
	var clientID uint64

	isDockerCmd := utils.IsDockerCmd(utils.CmdType(opts.CommandType))

	// check if the agent is running
	isAgentRunning := a.isAgentRunning(ctx)

	if !isAgentRunning {
		// Start the keploy agent as a detached process and pipe the logs into a file
		if !isDockerCmd && runtime.GOOS != "linux" {
			return 0, fmt.Errorf("Operating system not supported for this feature")
		}
		if isDockerCmd {
			// run the docker container instead of the agent binary
			go func() {
				if err := a.StartInDocker(ctx, a.logger); err != nil {
					a.logger.Error("failed to start docker agent", zap.Error(err))
				}
			}()
		} else {
			// Open the log file in append mode or create it if it doesn't exist
			logFile, err := os.OpenFile("keploy_agent.log", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				utils.LogError(a.logger, err, "failed to open log file")
				return 0, err
			}

			defer func() {
				err := logFile.Close()
				if err != nil {
					utils.LogError(a.logger, err, "failed to close agent log file")
				}
			}()
			agentCmd := exec.Command("sudo", "keployv2", "agent")
			agentCmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // Detach the process

			// Redirect the standard output and error to the log file
			agentCmd.Stdout = logFile
			agentCmd.Stderr = logFile

			err = agentCmd.Start()
			if err != nil {
				utils.LogError(a.logger, err, "failed to start keploy agent")
				return 0, err
			}

			a.logger.Info("keploy agent started", zap.Int("pid", agentCmd.Process.Pid))
		}
	}

	runningChan := make(chan bool)

	// Start a goroutine to check if the agent is running
	go func() {
		for {
			if a.isAgentRunning(ctx) {
				// Signal that the agent is running
				runningChan <- true
				return
			}
			time.Sleep(1 * time.Second) // Poll every second
		}
	}()

	var inode uint64
	if <-runningChan {
		// check if its docker then create a init container first
		// and then the app container
		usrApp := app.NewApp(a.logger, clientID, cmd, a.dockerClient, app.Options{
			DockerNetwork: opts.DockerNetwork,
			Container:     opts.Container,
			DockerDelay:   opts.DockerDelay,
		})
		a.apps.Store(clientID, usrApp)

		err := usrApp.Setup(ctx)
		if err != nil {
			utils.LogError(a.logger, err, "failed to setup app")
			return 0, err
		}

		if isDockerCmd {
			opts.DockerNetwork = usrApp.KeployNetwork
			// Start the init container to get the pid namespace
			inode, err = a.Initcontainer(ctx, app.Options{
				DockerNetwork: opts.DockerNetwork,
				Container:     opts.Container,
				DockerDelay:   opts.DockerDelay,
			})
			if err != nil {
				utils.LogError(a.logger, err, "failed to setup init container")
			}
		}
		opts.ClientID = clientID
		opts.AppInode = inode
		// Register the client with the server
		err = a.RegisterClient(ctx, opts)
		if err != nil {
			utils.LogError(a.logger, err, "failed to register client")
			return 0, err
		}
	}

	isAgentRunning = a.isAgentRunning(ctx)
	if !isAgentRunning {
		return 0, fmt.Errorf("keploy agent is not running, please start the agent first")
	}

	return clientID, nil
}

func (a *AgentClient) getApp(id uint64) (*app.App, error) {
	ap, ok := a.apps.Load(id)
	if !ok {
		return nil, fmt.Errorf("app with id:%v not found", id)
	}

	// type assertion on the app
	h, ok := ap.(*app.App)
	if !ok {
		return nil, fmt.Errorf("failed to type assert app with id:%v", id)
	}

	return h, nil
}

// RegisterClient registers the client with the server
func (a *AgentClient) RegisterClient(ctx context.Context, opts models.SetupOptions) error {

	isAgent := a.isAgentRunning(ctx)
	if !isAgent {
		return fmt.Errorf("keploy agent is not running, please start the agent first")
	}

	// Register the client with the server
	clientPid := uint32(os.Getpid())

	// start the app container and get the inode number
	// keploy agent would have already runnning,
	var inode uint64
	var err error
	if runtime.GOOS == "linux" {
		// send the network info to the kernel
		inode, err = hooks.GetSelfInodeNumber()
		if err != nil {
			a.logger.Error("failed to get inode number")
		}
	}

	// Register the client with the server
	requestBody := models.RegisterReq{
		SetupOptions: models.SetupOptions{
			DockerNetwork: opts.DockerNetwork,
			ClientNsPid:   clientPid,
			Mode:          opts.Mode,
			ClientID:      0,
			ClientInode:   inode,
			IsDocker:      a.conf.Agent.IsDocker,
			AppInode:      opts.AppInode,
		},
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		utils.LogError(a.logger, err, "failed to marshal request body for register client")
		return fmt.Errorf("error marshaling request body for register client: %s", err.Error())
	}

	resp, err := a.client.Post(fmt.Sprintf("http://localhost:%d/agent/register", a.conf.Agent.Port), "application/json", bytes.NewBuffer(requestJSON))
	if err != nil {
		utils.LogError(a.logger, err, "failed to send register client request")
		return fmt.Errorf("error sending register client request: %s", err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to register client: %s", resp.Status)
	}

	// TODO: Read the response body in which we return the app id
	var RegisterResp models.AgentResp
	err = json.NewDecoder(resp.Body).Decode(&RegisterResp)
	if err != nil {
		utils.LogError(a.logger, err, "failed to decode response body for register client")
		return fmt.Errorf("error decoding response body for register client: %s", err.Error())
	}

	if RegisterResp.Error != nil {
		return RegisterResp.Error
	}

	return nil
}

func (a *AgentClient) StartInDocker(ctx context.Context, logger *zap.Logger) error {

	// Start the keploy agent in a Docker container
	agentCtx := context.WithoutCancel(ctx)

	err := kdocker.StartInDocker(agentCtx, logger, &config.Config{
		InstallationID: a.conf.InstallationID,
	})
	if err != nil {
		utils.LogError(logger, err, "failed to start keploy agent in docker")
		return err
	}
	return nil
}

func (a *AgentClient) Initcontainer(ctx context.Context, opts app.Options) (uint64, error) {
	// Create a temporary file for the embedded initStop.sh script
	initFile, err := os.CreateTemp("", "initStop.sh")
	if err != nil {
		a.logger.Error("failed to create temporary file", zap.Error(err))
		return 0, err
	}
	defer func() {
		err := os.Remove(initFile.Name())
		if err != nil {
			a.logger.Error("failed to remove temporary file", zap.Error(err))
		}
	}()

	_, err = initFile.Write(initStopScript)
	if err != nil {
		a.logger.Error("failed to write script to temporary file", zap.Error(err))
		return 0, err
	}

	// Close the file after writing to avoid 'text file busy' error
	if err := initFile.Close(); err != nil {
		a.logger.Error("failed to close temporary file", zap.Error(err))
		return 0, err
	}

	err = os.Chmod(initFile.Name(), 0755)
	if err != nil {
		a.logger.Error("failed to make temporary script executable", zap.Error(err))
		return 0, err
	}

	// Create a channel to signal when the container starts
	containerStarted := make(chan struct{})

	// Start the Docker events listener in a separate goroutine
	go func() {
		events, errs := a.dockerClient.Events(ctx, events.ListOptions{})
		for {
			select {
			case event := <-events:
				if event.Type == "container" && event.Action == "start" && event.Actor.Attributes["name"] == "keploy-init" {
					a.logger.Info("Container keploy-init started")
					containerStarted <- struct{}{}
					return
				}
			case err := <-errs:
				a.logger.Error("Error while listening to Docker events", zap.Error(err))
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start the init container to get the PID namespace inode
	cmdCancel := func(cmd *exec.Cmd) func() error {
		return func() error {
			a.logger.Info("sending SIGINT to the Initcontainer", zap.Any("cmd.Process.Pid", cmd.Process.Pid))
			err := utils.SendSignal(a.logger, -cmd.Process.Pid, syscall.SIGINT)
			return err
		}
	}

	cmd := fmt.Sprintf("docker run --network=%s --name keploy-init --rm -v%s:/initStop.sh alpine /initStop.sh", opts.DockerNetwork, initFile.Name())

	// Execute the command
	grp, ok := ctx.Value(models.ErrGroupKey).(*errgroup.Group)
	if !ok {
		return 0, fmt.Errorf("failed to get errorgroup from the context")
	}

	grp.Go(func() error {
		println("Executing the init container command")
		cmdErr := utils.ExecuteCommand(ctx, a.logger, cmd, cmdCancel, 25*time.Second)
		if cmdErr.Err != nil && cmdErr.Type == utils.Init {
			utils.LogError(a.logger, cmdErr.Err, "failed to execute init container command")
		}

		println("Init container stopped")
		return nil
	})

	// Wait for the container to start or context to cancel
	select {
	case <-containerStarted:
		a.logger.Info("keploy-init container is running")
	case <-ctx.Done():
		return 0, fmt.Errorf("context canceled while waiting for container to start")
	}

	// Get the PID of the container's first process
	inspect, err := a.dockerClient.ContainerInspect(ctx, "keploy-init")
	if err != nil {
		a.logger.Error("failed to inspect container", zap.Error(err))
		return 0, err
	}

	pid := inspect.State.Pid
	a.logger.Info("Container PID", zap.Int("pid", pid))

	// Extract inode from the PID namespace
	pidNamespaceInode, err := kdocker.ExtractPidNamespaceInode(pid)
	if err != nil {
		a.logger.Error("failed to extract PID namespace inode", zap.Error(err))
		return 0, err
	}

	a.logger.Info("PID Namespace Inode", zap.String("inode", pidNamespaceInode))
	iNode, err := strconv.ParseUint(pidNamespaceInode, 10, 64)
	if err != nil {
		a.logger.Error("failed to convert inode to uint64", zap.Error(err))
		return 0, err
	}
	return iNode, nil
}

func (a *AgentClient) isAgentRunning(ctx context.Context) bool {

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost:%d/agent/health", a.conf.Agent.Port), nil)
	if err != nil {
		utils.LogError(a.logger, err, "failed to send request to the agent server")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		a.logger.Info("Keploy agent is not running in background, starting the agent")
		return false
	}
	a.logger.Info("Setup request sent to the server", zap.String("status", resp.Status))
	return true
}
