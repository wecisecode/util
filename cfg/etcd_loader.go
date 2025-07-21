package cfg

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/wecisecode/util/etcd"
	"github.com/wecisecode/util/merrs"
)

func getEtcd() (etcd.Client, error) {
	etcdPath := os.Getenv("ETCDPATH")
	if etcdPath == "" {
		return nil, merrs.NewError(errors.New("ETCDPATH not set"))
	}
	etcdUser := os.Getenv("ETCDUSER")
	etcdPass := os.Getenv("ETCDPASS")
	//
	chcli := make(chan etcd.Client)
	cherr := make(chan error)
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	go func() {
		cli, err := etcd.NewClient(etcdPath, etcdUser, etcdPass)
		if err != nil {
			cherr <- merrs.NewError(err)
		}
		chcli <- cli
	}()
	select {
	case <-timer.C:
		err := merrs.NewError(fmt.Errorf("connect etcd://%s@%s timeout", etcdUser, etcdPath))
		return nil, err
	case cli := <-chcli:
		return cli, nil
	case err := <-cherr:
		return nil, err
	}
}

func etcdGet(cli etcd.Client, key string) (*etcd.Node, error) {
	chnode := make(chan *etcd.Node)
	cherr := make(chan error)
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	go func() {
		node, err := cli.GetNode(key)
		if err != nil {
			cherr <- merrs.NewError(err)
		}
		chnode <- node
	}()
	select {
	case <-timer.C:
		err := merrs.NewError(fmt.Errorf("etcd get %s timeout", key))
		return nil, err
	case node := <-chnode:
		return node, nil
	case err := <-cherr:
		return nil, err
	}
}

func (mc *mConfig) watchETCDFiles(cli etcd.Client, parserf CfgParser, etcdfiles []string, chcfginfo chan *CfgInfo, stopped <-chan struct{}) (err error) {
	for _, etcdfile := range etcdfiles {
		watchinfo := "etcd:/" + etcdfile
		mc.log.Debug("load config from", watchinfo)
		var etcdfilematcher *regexp.Regexp
		etcdfileprefix := etcdfile
		etcdfilerecursive := false
		if n := strings.Index(etcdfile, "*"); n >= 0 {
			if n = strings.LastIndex(etcdfile[:n], "/"); n < 0 || etcdfile[0] != '/' {
				return merrs.NewError(fmt.Errorf("etcd filename(%s) error, must begin with '/'", etcdfile))
			}
			etcdfileprefix = etcdfile[:n]
			if nn := strings.Index(etcdfile[n+1:], "/"); nn > 0 {
				etcdfilerecursive = true
			}
			if len(etcdfile[n+1:]) > 2 && etcdfile[n+1:n+3] == "*/" {
				etcdfile = etcdfile[:n+2] + etcdfile[n+3:]
			}
			etcdfile = strings.ReplaceAll(etcdfile, "**", "*")
			sregx := regexp.MustCompile(`([^\*\w])`).ReplaceAllString(etcdfile, "\\$1")
			mc.log.Debug("etcd wildcard matcher:", etcdfile, "recursive:", etcdfilerecursive)
			sregx = strings.ReplaceAll(sregx, "*", ".*")
			etcdfilematcher, err = regexp.Compile(sregx)
			if err != nil {
				return merrs.NewError(err)
			}
		}
		cache_value := map[string]string{}
		on_change := func(fcm map[string]string) {
			ci := CfgInfo{}
			for etcdfilename, value := range fcm {
				if v, ok := cache_value[etcdfilename]; !ok || v != value {
					key := "etcd:/" + etcdfilename
					if value != "" {
						sm, err := parserf(value)
						if err != nil {
							mc.log.Error("parse", key, "error", err)
						} else {
							ci[key] = sm
						}
					} else {
						ci[key] = nil
					}
				}
			}
			chcfginfo <- &ci
		}
		err = func() error {
			node, err := etcdGet(cli, etcdfileprefix)
			if err != nil {
				return err
			}
			fcm := map[string]string{}
			nodes := []*etcd.Node{node}
			if !etcdfilerecursive {
				nodes = append(nodes, node.Nodes...)
			}
			for i := 0; i < len(nodes); i++ {
				n := nodes[i]
				if n.Dir && etcdfilerecursive {
					sn, err := etcdGet(cli, n.Key)
					if err != nil {
						return err
					}
					nodes = append(nodes, sn.Nodes...)
				}
				if etcdfilematcher == nil || etcdfilematcher.MatchString(n.Key) {
					mc.log.Debug(n.Key, "loaded")
					fcm[n.Key] = n.Value
				}
			}
			on_change(fcm)
			return nil
		}()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithCancel(context.Background())
		ch := cli.Watch(ctx, etcdfileprefix, true)
		go func() {
			mc.log.Debug("start watching", watchinfo)
			defer mc.log.Debug("stop watching", watchinfo)
			for {
				select {
				case evt := <-ch:
					if etcdfilematcher != nil && !etcdfilematcher.MatchString(evt.Node.Key) {
						mc.log.Debug("ignore not match ETCD Path:", evt.Node.Key, "Action:", evt.Action)
					} else if evt.Action == etcd.ActionPut {
						mc.log.Debug(evt.Node.Key, "changed")
						on_change(map[string]string{evt.Node.Key: evt.Node.Value})
					} else if evt.Action == etcd.ActionDelete {
						mc.log.Debug(evt.Node.Key, "deleted")
						on_change(map[string]string{evt.Node.Key: evt.Node.Value})
					} else {
						mc.log.Debug("ignore Path:", evt.Node.Key, "Action:", evt.Action)
					}
				case <-ctx.Done():
					return
				case <-stopped:
					cancel()
					return
				}
			}
		}()
	}
	return nil
}

func (mc *mConfig) WithETCD(cli etcd.Client) Configure {
	for _, vcfg := range mc.mergeConfigure.Values() {
		vcfg.(*mConfig).WithETCD(cli)
	}
	mc.etcdclient = cli
	if mc.chetcdclient != nil {
		mc.chetcdclient <- cli
	}
	return mc
}

func (mc *mConfig) loadFromETCD(parserf CfgParser, etcdfiles ...string) (retchci <-chan *CfgInfo, err error) {
	mc.chetcdclient = make(chan etcd.Client)
	etcdclienterr := make(chan error)
	go func() {
		if mc.etcdclient == nil {
			cli, err := getEtcd()
			if err != nil {
				etcdclienterr <- err
				return
			}
			if mc.etcdclient == nil {
				mc.etcdclient = cli
				mc.chetcdclient <- cli
			}
		}
	}()
	chcfginfo := make(chan *CfgInfo, 1)
	go func() {
		var chstopwatch chan struct{}
		for {
			select {
			case <-mc.stopped:
				if chstopwatch != nil {
					chstopwatch <- struct{}{}
				}
				return
			case etcdclient := <-mc.chetcdclient:
				if chstopwatch == nil {
					chstopwatch = make(chan struct{})
				} else {
					chstopwatch <- struct{}{}
				}
				err := mc.watchETCDFiles(etcdclient, parserf, etcdfiles, chcfginfo, chstopwatch)
				if etcdclienterr != nil {
					etcdclienterr <- err
				} else {
					mc.log.Error(err)
				}
			}
		}
	}()
	//
	retchci = chcfginfo
	err = <-etcdclienterr
	etcdclienterr = nil
	return
}
