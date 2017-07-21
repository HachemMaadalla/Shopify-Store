package kit

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"

	"github.com/Shopify/themekit/kittest"
)

func TestNewFileReader(t *testing.T) {
	kittest.GenerateProject()
	defer kittest.Cleanup()
	client := ThemeClient{Config: &Configuration{Directory: kittest.FixtureProjectPath}}
	watcher, err := newFileWatcher(client, "", true, fileFilter{}, func(ThemeClient, Asset, EventType) {})
	assert.Nil(t, err)
	assert.Equal(t, true, watcher.IsWatching())
	watcher.StopWatching()
}

func TestWatchDirectory(t *testing.T) {
	kittest.GenerateProject()
	defer kittest.Cleanup()
	filter, _ := newFileFilter(kittest.FixtureProjectPath, []string{}, []string{})
	w, _ := fsnotify.NewWatcher()
	newWatcher := &FileWatcher{
		filter:      filter,
		mainWatcher: w,
		client:      ThemeClient{Config: &Configuration{Directory: kittest.FixtureProjectPath}},
	}
	newWatcher.watch()
	assert.Nil(t, newWatcher.mainWatcher.Remove(filepath.Join(kittest.FixtureProjectPath, "assets")))
}

func TestWatchSymlinkDirectory(t *testing.T) {
	kittest.GenerateProject()
	defer kittest.Cleanup()
	filter, _ := newFileFilter(kittest.SymlinkProjectPath, []string{}, []string{})
	w, _ := fsnotify.NewWatcher()
	newWatcher := &FileWatcher{
		filter:      filter,
		mainWatcher: w,
		client:      ThemeClient{Config: &Configuration{Directory: kittest.SymlinkProjectPath}},
	}
	assert.Nil(t, newWatcher.watch())
	assert.Nil(t, newWatcher.mainWatcher.Remove(filepath.Join(kittest.FixtureProjectPath, "assets")))
}

func TestWatchConfig(t *testing.T) {
	kittest.GenerateProject()
	kittest.GenerateConfig("example.myshopify.com", true)
	defer kittest.Cleanup()
	filter, _ := newFileFilter(kittest.FixtureProjectPath, []string{}, []string{})
	w, _ := fsnotify.NewWatcher()
	newWatcher := &FileWatcher{
		done:          make(chan bool),
		filter:        filter,
		configWatcher: w,
	}

	err := newWatcher.WatchConfig("nope", make(chan bool))
	assert.NotNil(t, err)

	err = newWatcher.WatchConfig("config.yml", make(chan bool))
	assert.Nil(t, err)
}

func TestWatchFsEvents(t *testing.T) {
	kittest.GenerateProject()
	defer kittest.Cleanup()
	assetChan := make(chan Asset, 100)
	eventChan := make(chan fsnotify.Event)
	var wg sync.WaitGroup
	wg.Add(2)

	filter, _ := newFileFilter(kittest.FixtureProjectPath, []string{}, []string{})

	newWatcher := &FileWatcher{
		done:          make(chan bool),
		filter:        filter,
		mainWatcher:   &fsnotify.Watcher{Events: eventChan},
		client:        ThemeClient{Config: &Configuration{Directory: kittest.FixtureProjectPath}},
		configWatcher: &fsnotify.Watcher{Events: make(chan fsnotify.Event)},
	}

	newWatcher.callback = func(client ThemeClient, asset Asset, event EventType) {
		assert.Equal(t, Update, event)
		assetChan <- asset
		wg.Done()
	}

	go newWatcher.watchFsEvents()

	go func() {
		writes := []fsnotify.Event{
			{Name: filepath.Join(kittest.FixtureProjectPath, "templates", "template.liquid"), Op: fsnotify.Write},
			{Name: filepath.Join(kittest.FixtureProjectPath, "templates", "template.liquid"), Op: fsnotify.Write},
			{Name: filepath.Join(kittest.FixtureProjectPath, "templates", "template.liquid"), Op: fsnotify.Write},
			{Name: filepath.Join(kittest.FixtureProjectPath, "templates", "customers", "test.liquid"), Op: fsnotify.Write},
		}
		for _, fsEvent := range writes {
			eventChan <- fsEvent
		}
	}()

	wg.Wait()
	// test that the events are debounced
	assert.Equal(t, 2, len(assetChan))
}

func TestReloadConfig(t *testing.T) {
	kittest.GenerateProject()
	kittest.GenerateConfig("example.myshopify.com", true)
	defer kittest.Cleanup()
	reloadChan := make(chan bool, 100)

	configWatcher, _ := fsnotify.NewWatcher()
	newWatcher := &FileWatcher{
		done:          make(chan bool),
		mainWatcher:   &fsnotify.Watcher{Events: make(chan fsnotify.Event)},
		configWatcher: configWatcher,
	}

	newWatcher.callback = func(client ThemeClient, asset Asset, event EventType) {}
	err := newWatcher.WatchConfig("config.yml", reloadChan)
	assert.Nil(t, err)

	go newWatcher.watchFsEvents()
	configWatcher.Events <- fsnotify.Event{Name: "config.yml", Op: fsnotify.Write}

	_, ok := <-newWatcher.done
	assert.False(t, ok)
	assert.Equal(t, newWatcher.IsWatching(), false)
}

func TestStopWatching(t *testing.T) {
	kittest.GenerateProject()
	defer kittest.Cleanup()
	client := ThemeClient{Config: &Configuration{Directory: kittest.FixtureProjectPath}}
	watcher, err := newFileWatcher(client, "", true, fileFilter{}, func(ThemeClient, Asset, EventType) {})
	assert.Nil(t, err)
	assert.Equal(t, true, watcher.IsWatching())
	watcher.StopWatching()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, false, watcher.IsWatching())
}

func TestOnReload(t *testing.T) {
	kittest.GenerateProject()
	kittest.GenerateConfig("example.myshopify.com", true)
	defer kittest.Cleanup()
	reloadChan := make(chan bool, 100)

	configWatcher, _ := fsnotify.NewWatcher()
	newWatcher := &FileWatcher{
		done:          make(chan bool),
		mainWatcher:   &fsnotify.Watcher{Events: make(chan fsnotify.Event)},
		configWatcher: configWatcher,
		client:        ThemeClient{Config: &Configuration{Directory: kittest.FixtureProjectPath}},
	}

	err := newWatcher.WatchConfig("config.yml", reloadChan)
	assert.Nil(t, err)
	newWatcher.onReload()

	assert.Equal(t, len(reloadChan), 1)
	assert.Equal(t, newWatcher.IsWatching(), false)
}

func TestOnEvent(t *testing.T) {
	kittest.GenerateProject()
	defer kittest.Cleanup()
	newWatcher := &FileWatcher{
		waitNotify:     false,
		recordedEvents: newEventMap(),
		callback:       func(client ThemeClient, asset Asset, event EventType) {},
		client:         ThemeClient{Config: &Configuration{Directory: kittest.FixtureProjectPath}},
	}

	event1 := fsnotify.Event{Name: filepath.Join(kittest.FixtureProjectPath, "templates", "template.liquid"), Op: fsnotify.Write}
	event2 := fsnotify.Event{Name: filepath.Join(kittest.FixtureProjectPath, "templates", "customers", "test.liquid"), Op: fsnotify.Write}

	assert.Equal(t, newWatcher.recordedEvents.Count(), 0)
	newWatcher.onEvent(event1)
	assert.Equal(t, newWatcher.recordedEvents.Count(), 1)
	newWatcher.onEvent(event1)
	assert.Equal(t, newWatcher.recordedEvents.Count(), 1)
	newWatcher.onEvent(event2)
	assert.Equal(t, newWatcher.recordedEvents.Count(), 2)
}

func TestTouchNotifyFile(t *testing.T) {
	kittest.GenerateProject()
	defer kittest.Cleanup()
	notifyPath := "notifyTestFile"
	newWatcher := &FileWatcher{
		notify: notifyPath,
	}
	_, err := os.Stat(notifyPath)
	assert.True(t, os.IsNotExist(err))
	newWatcher.waitNotify = true
	newWatcher.touchNotifyFile()
	_, err = os.Stat(notifyPath)
	assert.False(t, os.IsNotExist(err))
	assert.False(t, newWatcher.waitNotify)
	os.Remove(notifyPath)
}

func TestHandleEvent(t *testing.T) {
	kittest.GenerateProject()
	defer kittest.Cleanup()
	writes := []struct {
		Name          string
		Event         fsnotify.Op
		ExpectedEvent EventType
	}{
		{Name: filepath.Join(kittest.FixtureProjectPath, "assets", "application.js"), Event: fsnotify.Create, ExpectedEvent: Update},
		{Name: filepath.Join(kittest.FixtureProjectPath, "assets", "application.js"), Event: fsnotify.Write, ExpectedEvent: Update},
		{Name: filepath.Join(kittest.FixtureProjectPath, "assets", "application.js"), Event: fsnotify.Remove, ExpectedEvent: Remove},
		{Name: filepath.Join(kittest.FixtureProjectPath, "assets", "application.js"), Event: fsnotify.Rename, ExpectedEvent: Remove},
	}

	for _, write := range writes {
		watcher := &FileWatcher{callback: func(client ThemeClient, asset Asset, event EventType) {
			assert.Equal(t, pathToProject(kittest.FixtureProjectPath, filepath.Join(kittest.FixtureProjectPath, "assets", "application.js")), asset.Key)
			assert.Equal(t, write.ExpectedEvent, event)
		},
			client: ThemeClient{Config: &Configuration{Directory: kittest.FixtureProjectPath}},
		}
		watcher.handleEvent(fsnotify.Event{Name: write.Name, Op: write.Event})
	}
}
