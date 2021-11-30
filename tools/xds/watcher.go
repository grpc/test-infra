package xds

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

// Watch watches a given file directory and return the Event.
func Watch(directory string, message chan<- fsnotify.Event) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					message <- event
				} else if event.Op&fsnotify.Create == fsnotify.Create {
					log.Println("created file:", event.Name)
					message <- event
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Println("removed file:", event.Name)
					message <- event
				} else if event.Op&fsnotify.Chmod == fsnotify.Chmod {
					log.Println("moved file:", event.Name)
					message <- event
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(directory)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
