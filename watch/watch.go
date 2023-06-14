package watch

import (
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/peterebden/go-cli-init/v5/logging"

	"github.com/please-build/puku/generate"
)

var log = logging.MustGetLogger()

func Watch(u *generate.Update, paths ...string) error {
	if len(paths) < 1 {
		return nil
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
					break
				}

				if filepath.Ext(event.Name) == ".go" {
					err := u.Update(filepath.Dir(event.Name))
					log.Infof("updating: %s", event.Name)
					if err != nil {
						log.Warningf("updating error: %s", err)
					}
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
