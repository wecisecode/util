package filewatcher

import (
	"io/fs"
	"os"
	"sync"
	"time"
)

type PollWatchInfo struct {
	fileinfo  fs.FileInfo
	callbacks []func(file string, err error)
}

var filewatcher_mutex sync.Mutex
var filewatcher = map[string]*PollWatchInfo{}

func PollingWatchFile(file string, ncb func(file string, err error)) error {
	filewatcher_mutex.Lock()
	defer filewatcher_mutex.Unlock()
	newfilewatcher := false
	pwi := filewatcher[file]
	if pwi == nil {
		pwi = &PollWatchInfo{}
		filewatcher[file] = pwi
		newfilewatcher = true
	}
	pwi.callbacks = append(pwi.callbacks, ncb)
	forcecb := ncb
	var fcheck func()
	fcheck = func() {
		nfi, err := os.Stat(file)
		if forcecb != nil {
			// 新注册的回调函数
			forcecb(file, err)
			forcecb = nil
			if !newfilewatcher {
				// 重复定义，不再重复开启轮询时钟，也不更新缓存状态
				return
			}
		} else {
			ofi := pwi.fileinfo
			if nfi == nil && ofi != nil || nfi != nil && (ofi == nil || !nfi.ModTime().Equal(ofi.ModTime()) || nfi.Size() != ofi.Size()) {
				for _, cb := range pwi.callbacks {
					cb(file, err)
				}
			}
		}
		// 更新缓存状态，开启轮询时钟
		pwi.fileinfo = nfi
		time.AfterFunc(1*time.Second, fcheck)
	}
	fcheck()
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
