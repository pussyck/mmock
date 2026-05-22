package config

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/jmartin82/mmock/v3/internal/config/logger"
	"github.com/jmartin82/mmock/v3/pkg/mock"
)

var log = logger.Log

var ErrFilePathIsNotUnderConfigPath = errors.New("File path is not under config path")
var ErrMockDoesntExist = errors.New("Definition doesn't exist")

type Mapping interface {
	Set(URI string, mock mock.Definition) error
	Delete(URI string) error
	Get(URI string) (mock.Definition, bool)
	List() []mock.Definition
}

// PrioritySort mock array sorted by priority
type PrioritySort []mock.Definition

func (s PrioritySort) Len() int {
	return len(s)
}
func (s PrioritySort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s PrioritySort) Less(i, j int) bool {
	return s[i].Control.Priority > s[j].Control.Priority
}

type ConfigMapping struct {
	mapper   *FSMapper
	mapping  map[string]mock.Definition
	sorted   []mock.Definition
	path     string
	fsListen atomic.Bool
	fsUpdate chan struct{}
	sync.RWMutex
}

func NewConfigMapping(path string, mapper *FSMapper, fsUpdate chan struct{}) *ConfigMapping {
	cm := &ConfigMapping{path: path, mapper: mapper, mapping: make(map[string]mock.Definition), fsUpdate: fsUpdate}
	cm.populate()
	cm.fsBind()
	go cm.listenFsChanges()
	return cm
}

func (fm *ConfigMapping) listenFsChanges() {
	for {
		if _, ok := <-fm.fsUpdate; !ok {
			return
		}
		if fm.fsIsBind() {
			fm.populate()
		}

	}
}

func (fm *ConfigMapping) Get(URI string) (mock.Definition, bool) {
	URI = fm.sanitizeURI(URI)
	defer fm.RUnlock()
	fm.RLock()
	mock, ok := fm.mapping[URI]
	return mock, ok
}

func (fm *ConfigMapping) Set(URI string, mock mock.Definition) error {
	fm.fsUnBind()
	defer fm.fsBind()
	defer fm.Unlock()
	fm.Lock()
	URI = fm.sanitizeURI(URI)
	fileName, err := fm.resolveFile(URI)
	if err != nil {
		return err
	}

	if err := fm.mapper.Write(fileName, mock); err != nil {
		return err
	}

	fm.mapping[URI] = mock
	fm.refreshSortedMapping()
	return nil
}
func (fm *ConfigMapping) Delete(URI string) error {

	fm.fsUnBind()
	defer fm.fsBind()
	defer fm.Unlock()
	fm.Lock()
	URI = fm.sanitizeURI(URI)
	fileName, err := fm.resolveFile(URI)
	if err != nil {
		return err
	}

	if err := os.Remove(fileName); err != nil {
		return err
	}

	delete(fm.mapping, URI)
	fm.refreshSortedMapping()
	return nil
}

func (fm *ConfigMapping) List() []mock.Definition {
	defer fm.RUnlock()
	fm.RLock()
	mocks := make([]mock.Definition, len(fm.sorted))
	copy(mocks, fm.sorted)

	return mocks
}

func (fm *ConfigMapping) populate() {
	mapping := make(map[string]mock.Definition)
	sorted := make([]mock.Definition, 0)
	if err := filepath.Walk(fm.path, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			log.Errorf("Error walking config path %s: %v\n", filePath, err)
			return nil
		}
		if fileInfo == nil || fileInfo.IsDir() {
			return nil
		}

		URI := strings.TrimPrefix(filePath, fm.path)
		mock, err := fm.read(URI)
		if err != nil {
			log.Errorf("Error %v. Loading config: %v\n", err, URI)
			return nil
		}

		mapping[mock.URI] = mock
		sorted = append(sorted, mock)
		return nil
	}); err != nil {
		log.Errorf("Error walking config path %s: %v\n", fm.path, err)
	}
	sort.Sort(PrioritySort(sorted))

	defer fm.Unlock()
	fm.Lock()
	fm.mapping = mapping
	fm.sorted = sorted
}

func (fm *ConfigMapping) load(URI string) error {
	mock, err := fm.read(URI)
	if err != nil {
		return err
	}

	defer fm.Unlock()
	fm.Lock()
	fm.mapping[mock.URI] = mock
	fm.refreshSortedMapping()

	return nil
}

func (fm *ConfigMapping) refreshSortedMapping() {
	sorted := make([]mock.Definition, 0, len(fm.mapping))
	for _, mock := range fm.mapping {
		sorted = append(sorted, mock)
	}
	sort.Sort(PrioritySort(sorted))
	fm.sorted = sorted
}

func (fm *ConfigMapping) read(URI string) (mock.Definition, error) {
	URI = fm.sanitizeURI(URI)
	fileName, errf := fm.resolveFile(URI)
	if errf != nil {
		return mock.Definition{}, errf
	}

	mock, err := fm.mapper.Read(fileName)
	mock.URI = URI
	if err != nil {
		return mock, err

	}
	if err := mock.Validate(); err != nil {
		log.Errorf("Invalid mock definition in: %s error: %s\n", fileName, err)
		return mock, ErrInvalidMockDefinition
	}

	return mock, nil
}

func (fm *ConfigMapping) resolveFile(URI string) (string, error) {
	filename, err := filepath.Abs(path.Join(fm.path, URI))
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(filename, fm.path) {
		log.Error("File path not under the config path\n")
		return "", ErrFilePathIsNotUnderConfigPath
	}
	return filename, nil
}

func (fm *ConfigMapping) fsUnBind() {
	fm.fsListen.Store(false)
}

func (fm *ConfigMapping) fsBind() {
	fm.fsListen.Store(true)
}

func (fm *ConfigMapping) fsIsBind() bool {
	return fm.fsListen.Load()
}

func (fm *ConfigMapping) sanitizeURI(URI string) string {
	return strings.Trim(strings.TrimPrefix(URI, string(os.PathSeparator)), " ")
}
