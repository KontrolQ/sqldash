package redishost

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"sqldash/models"

	redisclient "github.com/redis/go-redis/v9"
)

type Profile string

const (
	ReadOnly  Profile = "reader"
	ReadWrite Profile = "writer"
	Inspect   Profile = "inspector"
	Monitor   Profile = "monitor"
	Subscribe Profile = "subscriber"
	Publish   Profile = "publisher"
)

type manifest struct {
	Version      int
	DatabaseID   uint
	Identity     string
	Generation   uint64
	ProcessID    int
	ProcessStart string
}

type aclUser struct {
	Hash      string
	Profile   Profile
	Subscribe bool
	Publish   bool
}

type aclDocument struct {
	DatabaseID uint
	Identity   string
	Users      map[string]aclUser
}

type controlRequest struct {
	Action       string
	DatabaseID   uint
	Identity     string
	Generation   uint64
	RestoreToken string
}

type controlResponse struct {
	Identity   string
	Generation uint64
	Failure    string
}

type managedInstance struct {
	record   models.Database
	instance models.RedisInstance
	command  *exec.Cmd
	finished chan error
	lock     *os.File
	control  *redisclient.Client
}

type manager struct {
	mutex     sync.Mutex
	ready     atomic.Bool
	stopping  atomic.Bool
	processes map[uint]*managedInstance
	server    *http.Server
	reconcile func(context.Context, *models.Database) error
}
