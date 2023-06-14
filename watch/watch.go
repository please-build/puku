package watch

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/peterebden/go-cli-init/v5/logging"

	"github.com/please-build/puku/generate"
)

var log = logging.MustGetLogger()

const debounceDuration = 200 * time.Millisecond

// debouncer batches up updates to paths, waiting for a debounceDuration to pass. This avoids running puku many times
// during git checkouts etc. but it also avoids inconsistent state when files are being moved around rapidly.
type debouncer struct {
	paths  map[string]struct{}
	timer  *time.Timer
	mux    sync.Mutex
	update *generate.Update
}

// updatePath adds a path to the batch and resets the timer to the deboundDuration
func (d *debouncer) updatePath(path string) {
	d.mux.Lock()
	defer d.mux.Unlock()

	d.paths[path] = struct{}{}
	if d.timer != nil {
		d.timer.Stop()
		d.timer.Reset(debounceDuration)
	} else {
		d.timer = time.NewTimer(debounceDuration)
		go d.wait()
	}
}

// wait waits for the timer to fire before updating the paths
func (d *debouncer) wait() {
	<-d.timer.C

	d.mux.Lock()
	paths := make([]string, 0, len(d.paths))
	for p := range d.paths {
		paths = append(paths, p)
	}

	if err := d.update.Update(paths...); err != nil {
		log.Warningf("failed to update: %v", err)
	}
	log.Info("updated paths: %v", strings.Join(paths, ", "))
	d.paths = map[string]struct{}{}
	d.mux.Unlock()

	//nolint:staticcheck
	d.wait() // infinite recursive calls are a lint error but it's what we want here
}

func Watch(u *generate.Update, paths ...string) error {
	if len(paths) < 1 {
		return nil
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	d := &debouncer{
		update: u,
		paths:  map[string]struct{}{},
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) && !event.Has(fsnotify.Remove) {
					break
				}

				if filepath.Ext(event.Name) == ".go" {
					d.updatePath(filepath.Dir(event.Name))
					break
				}

				if event.Has(fsnotify.Create) {
					if info, err := os.Lstat(event.Name); err == nil {
						if info.IsDir() {
							if err := add(watcher, event.Name); err != nil {
								log.Warningf("failed to set up watcher: %v", err)

							}
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Warningf("watcher error: %s", err)
			}
		}
	}()

	if err := add(watcher, paths...); err != nil {
		return err
	}
	select {}

}

func add(watcher *fsnotify.Watcher, paths ...string) error {
	for _, path := range paths {
		if path == "" {
			path = "."
		}
		err := watcher.Add(path)
		if err != nil {
			return err
		}
	}
	return nil
}
