package filewatcher

import (
	"io/fs"
	"os"
	"time"

	"github.com/wecisecode/util/cmap"
)

type PollWatchInfo struct {
	fileinfo  fs.FileInfo
	callbacks []func(file string, err error)
}

var filewatcher = cmap.New[string, *PollWatchInfo]()

func init() {
	go func() {
		t := time.NewTimer(1 * time.Second)
		for range t.C {
			filewatcher.IterCb(checkfilechange)
			t.Reset(1 * time.Second)
		}
	}()
}

func checkfilechange(file string, pwi *PollWatchInfo) {
	nfi, err := os.Stat(file)
	ofi := pwi.fileinfo
	if nfi == nil && ofi != nil || nfi != nil && (ofi == nil || !nfi.ModTime().Equal(ofi.ModTime()) || nfi.Size() != ofi.Size()) {
		for _, cb := range pwi.callbacks {
			go cb(file, err)
		}
	}
	// 更新缓存状态，开启轮询时钟
	pwi.fileinfo = nfi
}

func PollingWatchFile(file string, ncb func(file string, err error)) error {
	pwi, _ := filewatcher.GetWithNew(file, func() (*PollWatchInfo, error) {
		return &PollWatchInfo{}, nil
	})
	pwi.callbacks = append(pwi.callbacks, ncb)
	_, err := os.Stat(file)
	ncb(file, err)
	if err != nil {
		return err
	}
	// fsnotify 这个Watcher不好使，文件改名后再改回来就监测不到了，文件变化实时性要求不高的情况下，通过轮询方式监测更靠谱
	// watcher, err := fsnotify.NewWatcher()
	// if err != nil {
	// 	return err
	// }
	// go func() {
	// 	logger.Infof("watching %s", ini_file)
	// 	for {
	// 		select {
	// 		case event := <-watcher.Events:
	// 			if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
	// 				logger.Info("config file", ini_file, "changed")
	// 				err = db.loadIniFile(ini_file)
	// 				if err != nil {
	// 					logger.Error("load config file", ini_file, "error:", err)
	// 					watcher.Add(ini_file)
	// 				}
	// 			}
	// 		case err := <-watcher.Errors:
	// 			logger.Error("watching config file", ini_file, "error:", err)
	// 		}
	// 	}
	// }()
	// watcher.Add(ini_file)
	return nil
}
