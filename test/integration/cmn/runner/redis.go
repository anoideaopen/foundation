package runner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	runnerFbk "github.com/hyperledger/fabric/integration/nwo/runner"
	dcontainer "github.com/moby/moby/api/types/container"
	dnetwork "github.com/moby/moby/api/types/network"
	dcli "github.com/moby/moby/client"
	"github.com/redis/go-redis/v9"
	"github.com/tedsuo/ifrit"
)

const (
	RedisDefaultImage = "redis:7.2.4"
)

// RedisDB manages the execution of an instance of a dockerized RedisDB for tests.
type RedisDB struct {
	Client        dcli.APIClient
	Image         string
	HostIP        string
	HostPort      int
	ContainerPort dnetwork.Port
	Name          string
	StartTimeout  time.Duration
	Binds         []string

	OutputStream io.Writer

	creator          string
	containerID      string
	hostAddress      string
	containerAddress string
	address          string

	mutex   sync.Mutex
	stopped bool
}

// Run runs a RedisDB container. It implements the ifrit.Runner interface
func (r *RedisDB) Run(sigCh <-chan os.Signal, ready chan<- struct{}) error {
	if r.Image == "" {
		r.Image = RedisDefaultImage
	}

	if r.Name == "" {
		r.Name = runnerFbk.DefaultNamer()
	}

	if r.HostIP == "" {
		r.HostIP = "127.0.0.1"
	}

	if r.ContainerPort.IsZero() || r.ContainerPort.Num() == 0 {
		r.ContainerPort = dnetwork.MustParsePort("6379/tcp")
	}

	if r.StartTimeout == 0 {
		r.StartTimeout = runnerFbk.DefaultStartTimeout
	}

	if r.Client == nil {
		client, err := dcli.New(dcli.FromEnv)
		if err != nil {
			return err
		}
		r.Client = client
	}

	hostConfig := &dcontainer.HostConfig{
		Binds: r.Binds,
		PortBindings: map[dnetwork.Port][]dnetwork.PortBinding{
			r.ContainerPort: {{
				HostIP:   netip.MustParseAddr(r.HostIP),
				HostPort: strconv.Itoa(r.HostPort),
			}},
		},
		AutoRemove: true,
	}

	container, err := r.Client.ContainerCreate(context.Background(), dcli.ContainerCreateOptions{
		Config: &dcontainer.Config{
			Image: r.Image,
		},
		HostConfig: hostConfig,
		Name:       r.Name,
	})
	if err != nil {
		return err
	}
	r.containerID = container.ID

	_, err = r.Client.ContainerStart(context.Background(), container.ID, dcli.ContainerStartOptions{})
	if err != nil {
		return err
	}
	defer func() { err = r.Stop() }()

	res, err := r.Client.ContainerInspect(context.Background(), container.ID, dcli.ContainerInspectOptions{})
	if err != nil {
		return err
	}
	r.hostAddress = net.JoinHostPort(
		res.Container.NetworkSettings.Ports[r.ContainerPort][0].HostIP.String(),
		res.Container.NetworkSettings.Ports[r.ContainerPort][0].HostPort,
	)
	r.containerAddress = net.JoinHostPort(
		res.Container.NetworkSettings.Networks[res.Container.HostConfig.NetworkMode.NetworkName()].IPAddress.String(),
		strconv.Itoa(int(r.ContainerPort.Num())),
	)

	streamCtx, streamCancel := context.WithCancel(context.Background())
	defer streamCancel()
	go r.streamLogs(streamCtx)

	containerExit := r.wait()
	ctx, cancel := context.WithTimeout(context.Background(), r.StartTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return fmt.Errorf("database in container %s did not start: %w", r.containerID, ctx.Err())
	case <-containerExit:
		return errors.New("container exited before ready")
	case <-r.ready(ctx, r.hostAddress):
		r.address = r.hostAddress
	case <-r.ready(ctx, r.containerAddress):
		r.address = r.containerAddress
	}

	cancel()
	close(ready)

	for {
		select {
		case err := <-containerExit:
			return err
		case <-sigCh:
			if err := r.Stop(); err != nil {
				return err
			}
		}
	}
}

func endpointReady(ctx context.Context, addr string) bool {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	redisOpts := &redis.UniversalOptions{
		Addrs:    []string{addr},
		Password: "",
		ReadOnly: false,
	}
	client := redis.NewUniversalClient(redisOpts)

	status := client.Ping(ctx)
	if status.Err() != nil {
		return false
	}

	if status.Val() != "PONG" {
		return false
	}

	return true
}

func (r *RedisDB) ready(ctx context.Context, addr string) <-chan struct{} {
	readyCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			if endpointReady(ctx, addr) {
				close(readyCh)
				return
			}
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}()

	return readyCh
}

func (r *RedisDB) wait() <-chan error {
	exitCh := make(chan error)
	go func() {
		resWait := r.Client.ContainerWait(context.Background(), r.containerID, dcli.ContainerWaitOptions{})
		select {
		case res := <-resWait.Result:
			err := fmt.Errorf("redisdb: process exited with %d", res.StatusCode)
			exitCh <- err
		case err := <-resWait.Error:
			exitCh <- err
		}
	}()

	return exitCh
}

func (r *RedisDB) streamLogs(ctx context.Context) {
	if r.OutputStream == nil {
		return
	}

	go func() {
		res, err := r.Client.ContainerLogs(ctx, r.containerID, dcli.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		})
		if err != nil {
			fmt.Fprintf(r.OutputStream, "log stream ended with error: %s", err)
		}
		defer res.Close()

		reader := bufio.NewReader(res)

		for {
			select {
			case <-ctx.Done():
				fmt.Fprint(r.OutputStream, "log stream ended with cancel context")
				return
			default:
				// Loop forever dumping lines of text into the containerLogger
				// until the pipe is closed
				line, err := reader.ReadString('\n')
				if len(line) > 0 {
					r.OutputStream.Write([]byte(line))
				}
				switch err {
				case nil:
				case io.EOF:
					fmt.Fprintf(r.OutputStream, "Container %s has closed its IO channel", r.containerID)
					return
				default:
					fmt.Fprintf(r.OutputStream, "Error reading container output: %s", err)
					return
				}
			}
		}
	}()
}

// Address returns the address successfully used by the readiness check.
func (r *RedisDB) Address() string {
	return r.address
}

// HostAddress returns the host address where this RedisDB instance is available.
func (r *RedisDB) HostAddress() string {
	return r.hostAddress
}

// ContainerAddress returns the container address where this RedisDB instance
// is available.
func (r *RedisDB) ContainerAddress() string {
	return r.containerAddress
}

// ContainerID returns the container ID of this RedisDB
func (r *RedisDB) ContainerID() string {
	return r.containerID
}

// Start starts the RedisDB container using an ifrit runner
func (r *RedisDB) Start() error {
	r.creator = string(debug.Stack())
	p := ifrit.Invoke(r)

	select {
	case <-p.Ready():
		return nil
	case err := <-p.Wait():
		return err
	}
}

// Stop stops and removes the RedisDB container
func (r *RedisDB) Stop() error {
	r.mutex.Lock()
	if r.stopped {
		r.mutex.Unlock()
		return errors.New("container " + r.containerID + " already stopped")
	}
	r.stopped = true
	r.mutex.Unlock()

	t := 0
	_, err := r.Client.ContainerStop(context.Background(), r.containerID, dcli.ContainerStopOptions{
		Timeout: &t,
	})
	if err != nil {
		return err
	}

	return nil
}
