package runtime

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"sqldash/config"
	"sqldash/secretstore"
	"sqldash/supervisor"
)

type Runtime struct {
	mutex    sync.Mutex
	lock     *os.File
	prepared bool
	ready    bool
}

type administratorIntent struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	SystemIdentifier string `json:"systemIdentifier"`
	Initialized      bool   `json:"initialized"`
	Ready            bool   `json:"ready"`
}

func New() *Runtime {
	return &Runtime{}
}

func (self *Runtime) Prepare(ctx context.Context) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if !config.Postgres.Enabled {
		return ErrDisabled
	}
	if err := config.VerifyPostgres(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if self.prepared {
		return nil
	}
	if !config.Postgres.Managed {
		_, err := self.readAdministrator()
		self.prepared = err == nil
		return err
	}
	if err := managedPlatform(); err != nil {
		return err
	}
	credentials, err := postgresCredentials()
	if err != nil {
		return err
	}
	if err = ensureDirectory(config.PostgresDirectory(), nil, 0o711); err != nil {
		return err
	}
	self.lock, err = acquireClusterLock(filepath.Join(config.PostgresDirectory(), clusterLockFileName))
	if err != nil {
		return err
	}
	prepared := false
	defer func() {
		if !prepared {
			releaseClusterLock(self.lock)
			self.lock = nil
		}
	}()
	if err = verifyBinaries(ctx, credentials); err != nil {
		return err
	}
	newCluster, err := inspectDataDirectory()
	if err != nil {
		return err
	}
	if running, runningError := clusterRunning(config.PostgresDataPath()); runningError != nil {
		return runningError
	} else if running {
		return ErrClusterRunning
	}
	store, err := secretstore.Open(config.PostgresSecretsPath())
	if err != nil {
		return err
	}
	intent, err := self.administratorForPreparation(store, newCluster)
	if err != nil {
		return err
	}
	for _, directory := range []string{filepath.Dir(config.PostgresDataPath()), config.PostgresDataPath(), config.PostgresConfigPath(), config.PostgresRuntimePath()} {
		if err = ensureDirectory(directory, credentials, 0o700); err != nil {
			return err
		}
	}
	if err = ensureDirectory(config.PostgresStagingPath(), nil, 0o711); err != nil {
		return err
	}
	if err = ownDataDirectory(config.PostgresDataPath(), credentials); err != nil {
		return err
	}
	if newCluster {
		bootstrapPath := filepath.Join(config.PostgresConfigPath(), bootstrapFileName)
		if err = writePrivateFile(bootstrapPath, []byte(intent.Password+"\n"), credentials); err != nil {
			return err
		}
		if err = initialize(ctx, credentials, bootstrapPath); err != nil {
			return err
		}
		stillEmpty, inspectionError := inspectDataDirectory()
		if inspectionError != nil {
			return inspectionError
		}
		if stillEmpty {
			return ErrClusterDamaged
		}
	}
	identifier, err := clusterIdentifier(ctx, credentials)
	if err != nil {
		return err
	}
	if intent.SystemIdentifier != "" && intent.SystemIdentifier != identifier {
		return ErrClusterIdentity
	}
	intent.SystemIdentifier = identifier
	intent.Initialized = true
	if err = writeAdministrator(store, intent); err != nil {
		return err
	}
	if err = writeConfiguration(credentials); err != nil {
		return err
	}
	self.prepared = true
	prepared = true
	return nil
}

func (self *Runtime) AdminPassword() (string, error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	intent, err := self.readAdministrator()
	if err != nil {
		return "", err
	}
	return intent.Password, nil
}

func (self *Runtime) SystemIdentifier() (string, error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	intent, err := self.readAdministrator()
	if err != nil {
		return "", err
	}
	return intent.SystemIdentifier, nil
}

func (self *Runtime) ConfirmReady(ctx context.Context) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if !self.prepared {
		return errors.New("PostgreSQL runtime has not been prepared")
	}
	store, err := secretstore.Open(config.PostgresSecretsPath())
	if err != nil {
		return err
	}
	intent, err := readAdministrator(store)
	if err != nil {
		return err
	}
	if config.Postgres.Managed && (!intent.Initialized || intent.SystemIdentifier == "") {
		return ErrClusterIdentity
	}
	if !intent.Ready {
		intent.Ready = true
		if err = writeAdministrator(store, intent); err != nil {
			return err
		}
	}
	if config.Postgres.Managed {
		bootstrapPath := filepath.Join(config.PostgresConfigPath(), bootstrapFileName)
		if err = removePrivateFile(bootstrapPath); err != nil {
			return err
		}
		if err = removeBootstrapPendingFiles(); err != nil {
			return err
		}
	}
	self.ready = true
	return nil
}

func (self *Runtime) Ready() bool {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return self.ready
}

func (self *Runtime) StoppedSafe(ctx context.Context) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return self.stoppedSafe(ctx)
}

func (self *Runtime) stoppedSafe(ctx context.Context) error {
	if !config.Postgres.Managed {
		return errors.New("unmanaged PostgreSQL shutdown must be confirmed by its operator")
	}
	if err := managedPlatform(); err != nil {
		return err
	}
	if running, err := clusterRunning(config.PostgresDataPath()); err != nil {
		return err
	} else if running {
		return ErrClusterRunning
	}
	credentials, err := postgresCredentials()
	if err != nil {
		return err
	}
	identifier, state, err := readClusterControl(ctx, credentials)
	if err != nil {
		return err
	}
	intent, err := self.readAdministrator()
	if err != nil {
		return err
	}
	if intent.SystemIdentifier != identifier {
		return ErrClusterIdentity
	}
	if state != "shut down" {
		return errors.New("PostgreSQL did not complete a clean shutdown; cluster quiescence is unconfirmed")
	}
	return nil
}

func (self *Runtime) Close() error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.lock == nil {
		self.prepared = false
		self.ready = false
		return nil
	}
	if err := self.stoppedSafe(context.Background()); err != nil {
		return err
	}
	releaseClusterLock(self.lock)
	self.lock = nil
	self.prepared = false
	self.ready = false
	return nil
}

func (self *Runtime) readAdministrator() (*administratorIntent, error) {
	if _, err := os.Lstat(config.PostgresSecretsPath()); errors.Is(err, os.ErrNotExist) {
		return nil, ErrAdministratorMissing
	} else if err != nil {
		return nil, err
	}
	store, err := secretstore.Open(config.PostgresSecretsPath())
	if err != nil {
		return nil, err
	}
	return readAdministrator(store)
}

func readAdministrator(store *secretstore.Store) (*administratorIntent, error) {
	sealed, err := store.Get(secretNamespace, administratorSecretName)
	if errors.Is(err, secretstore.ErrNotFound) {
		return nil, ErrAdministratorMissing
	}
	if err != nil {
		return nil, err
	}
	defer clear(sealed)
	var intent administratorIntent
	if err = json.Unmarshal(sealed, &intent); err != nil || intent.Username != config.Postgres.AdminUsername || intent.Password == "" || len(intent.Password) > 4096 || strings.ContainsRune(intent.Password, 0) || config.Postgres.Managed && strings.ContainsAny(intent.Password, "\r\n") {
		return nil, ErrAdministratorMissing
	}
	return &intent, nil
}

func writeAdministrator(store *secretstore.Store, intent *administratorIntent) error {
	contents, err := json.Marshal(intent)
	if err != nil {
		return err
	}
	defer clear(contents)
	return store.Set(secretNamespace, administratorSecretName, contents)
}

func (self *Runtime) administratorForPreparation(store *secretstore.Store, newCluster bool) (*administratorIntent, error) {
	intent, err := readAdministrator(store)
	if err == nil {
		if newCluster && (intent.Initialized || intent.Ready || intent.SystemIdentifier != "") {
			return nil, ErrClusterDamaged
		}
		return intent, nil
	}
	if !errors.Is(err, ErrAdministratorMissing) || !newCluster {
		return nil, err
	}
	if _, existingError := store.Get(secretNamespace, administratorSecretName); !errors.Is(existingError, secretstore.ErrNotFound) {
		return nil, ErrAdministratorMissing
	}
	passwordBytes := make([]byte, 48)
	if _, err = rand.Read(passwordBytes); err != nil {
		return nil, err
	}
	password := base64.RawURLEncoding.EncodeToString(passwordBytes)
	clear(passwordBytes)
	intent = &administratorIntent{Username: config.Postgres.AdminUsername, Password: password}
	if err = writeAdministrator(store, intent); err != nil {
		return nil, err
	}
	return intent, nil
}

func (self *Runtime) Process(readiness func(context.Context) error) (supervisor.Process, error) {
	if !config.Postgres.Enabled {
		return supervisor.Process{}, ErrDisabled
	}
	if !config.Postgres.Managed {
		return supervisor.Process{}, errors.New("unmanaged PostgreSQL has no supervised process")
	}
	if readiness == nil {
		return supervisor.Process{}, ErrReadinessRequired
	}
	if err := managedPlatform(); err != nil {
		return supervisor.Process{}, err
	}
	credentials, err := postgresCredentials()
	if err != nil {
		return supervisor.Process{}, err
	}
	return supervisor.Process{
		Name:                ProcessName,
		Path:                filepath.Join(config.Postgres.BinaryDirectory, "postgres"),
		Arguments:           []string{"-D", config.PostgresDataPath(), "-c", "config_file=" + filepath.Join(config.PostgresConfigPath(), configurationFileName)},
		Directory:           config.PostgresDirectory(),
		Environment:         postgresEnvironment(),
		IsolatedEnvironment: true,
		Credentials:         credentials,
		Prepare:             self.Prepare,
		Readiness: func(ctx context.Context) error {
			if err := readiness(ctx); err != nil {
				return err
			}
			return self.ConfirmReady(ctx)
		},
		ReadyTimeout:  config.Postgres.StartupTimeout,
		StopSignal:    fastShutdownSignal(),
		StopGrace:     config.Postgres.ShutdownTimeout,
		ShutdownGroup: ShutdownGroup,
		OnStopped: func(err error) {
			self.mutex.Lock()
			self.ready = false
			self.mutex.Unlock()
		},
	}, nil
}
