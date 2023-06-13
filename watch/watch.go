package watch

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/peterebden/go-cli-init/v5/logging"

	"github.com/please-build/puku/generate"
	"github.com/please-build/puku/workspace"
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
				if filepath.Ext(event.Name) != ".go" {
					break
				}
				if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
					break
				}
				err := u.Update(filepath.Dir(event.Name))
				log.Infof("updating: %s", event.Name)
				if err != nil {
					log.Warningf("updating error: %s", err)
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
		walkSubdirs := false
		if strings.HasSuffix(path, "...") {
			path = strings.TrimSuffix(path, "...")
			walkSubdirs = true
		}
		if path == "" {
			path = "."
		}
		if walkSubdirs {
			err := workspace.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
				if !d.IsDir() {
					return nil
				}
				if err := watcher.Add(path); err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return err
			}
		} else {
			err := watcher.Add(path)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
