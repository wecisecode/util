package etcd

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"os"
	"strings"
	"sync"
	"time"
)

type wclientv3 struct {
	client *clientv3.Client
}

type wclientv3Locker struct {
	sess   *concurrency.Session
	key    string
	ttl    int64
	lk sync.Locker
}

func (locker *wclientv3Locker) Lock() {
	defer func() {
		recover()
	}()
	locker.lk.Lock()
}

func (locker *wclientv3Locker) Unlock() {
	ch := make(chan bool)
	go func() {
		timeout := time.After(time.Second*time.Duration(locker.ttl))
		select {
		case <-locker.sess.Done():
		case <-timeout:
			// WARNING: prevent the program from getting stuck here, but it is possible that the lock does not actually exit
		}

		close(ch)
	}()
	go func() {
		defer func() {
			recover()
		}()
		locker.lk.Unlock()
		_ = locker.sess.Close()
	}()
	<-ch
}

func (c *wclientv3) connect(endpoints []string, opts ...opt) error {
	config := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	}
	op := new(option)
	for _, fn := range opts {
		fn(op)
	}
	if op.auth {
		config.Username = op.user
		config.Password = op.pass
	}

	cli, err := clientv3.New(config)
	if err != nil {
		return err
	}
	c.client = cli
	return nil
}

func (c *wclientv3) Put(key, val string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err := c.client.Put(ctx, key, val)
	return err
}

func (c *wclientv3) PutTTL(key, val string, sec int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	leaseResp, err := c.client.Grant(ctx, sec)
	if err != nil {
		return err
	}
	_, err = c.client.Put(ctx, key, val, clientv3.WithLease(leaseResp.ID))
	return err
}

func (c *wclientv3) Get(key string) (val string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	resp, err := c.client.Get(ctx, key)
	if err != nil {
		return "", err
	}
	for _, v := range resp.Kvs {
		return string(v.Value), nil
	}
	return "", nil
}

func (c *wclientv3) GetNode(key string) (node *Node, err error) {
	var (
		topNode *Node
		/*
			{1:["/"], 2:["/foo", "/foo2"], 3:["/foo/bar", "/foo2/bar"], 4:["/foo/bar/test"]}
		*/
		all       = make(map[int][]*Node)
		min       int
		max       int
		prefixKey string
	)
	// parent
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	r, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if key == Separator {
		min = 1
		prefixKey = Separator
	} else {
		min = len(strings.Split(key, Separator))
		prefixKey = key + Separator
	}
	max = min
	all[min] = []*Node{{Key: key}}
	if r.Count != 0 {
		all[min][0].Value = string(r.Kvs[0].Value)
		all[min][0].CreateRevision = r.Kvs[0].CreateRevision
		all[min][0].ModRevision = r.Kvs[0].ModRevision
		if r.Kvs[0].Lease != 0 {
			all[min][0].TTL = c.getTTL(r.Kvs[0].Lease)
		}
	}

	//child
	resp, err := c.client.Get(ctx, prefixKey, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
	if err != nil {
		return nil, err
	}

	for _, v := range resp.Kvs {
		if string(v.Key) == Separator {
			continue
		}
		keys := strings.Split(string(v.Key), Separator) // /foo/bar
		var begin bool
		for i := range keys { // ["", "foo", "bar"]
			k := strings.Join(keys[0:i+1], Separator)
			if k == "" {
				continue
			}
			if key == Separator {
				begin = true
			} else if k == key {
				begin = true
				continue
			}
			if begin {
				node := &Node{
					Key: k,
				}
				if node.Key == string(v.Key) {
					node.Value = string(v.Value)
					node.CreateRevision = v.CreateRevision
					node.ModRevision = v.ModRevision
					if v.Lease != 0 {
						node.TTL = c.getTTL(v.Lease)
					}
				}
				level := len(strings.Split(k, Separator))
				if level > max {
					max = level
				}

				if _, ok := all[level]; !ok {
					all[level] = make([]*Node, 0)
				}
				levelNodes := all[level]
				var isExist bool
				for _, n := range levelNodes {
					if n.Key == k {
						isExist = true
					}
				}
				if !isExist {
					all[level] = append(all[level], node)
				}
			}
		}
	}

	// parent-child mapping
	for i := max; i > min; i-- {
		for _, a := range all[i] {
			for _, pa := range all[i-1] {
				if i == 2 {
					pa.Nodes = append(pa.Nodes, a)
					pa.Dir = true
				} else {
					if strings.HasPrefix(a.Key, pa.Key+Separator) {
						pa.Nodes = append(pa.Nodes, a)
						pa.Dir = true
					}
				}
			}
		}
	}
	topNode = all[min][0]
	return topNode, nil
}

func (c *wclientv3) Delete(key string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if _, err := c.client.Delete(ctx, key); err != nil {
		return err
	}
	return nil
}

func (c *wclientv3) DeleteDir(key string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if _, err = c.client.Delete(ctx, key+Separator, clientv3.WithPrefix()); err != nil {
		return err
	}
	if _, err := c.client.Delete(ctx, key); err != nil {
		return err
	}
	return nil
}

func (c *wclientv3) Watch(ctx context.Context, key string, recursive bool) (ech chan *Event) {
	ech = make(chan *Event, 5)
	go func() {
		// https://github.com/etcd-io/etcd/issues/8980
		// https://github.com/sensu/sensu-go/issues/3012
		leaderCtx := clientv3.WithRequireLeader(ctx)
		var watchChan clientv3.WatchChan
		if recursive {
			watchChan = c.client.Watch(leaderCtx, key, clientv3.WithPrefix())
		} else {
			watchChan = c.client.Watch(leaderCtx, key)
		}

		watchCtx, watchCancel := context.WithCancel(ctx)
		defer watchCancel()
	L:
		for {
			select {
			case wresp, ok := <-watchChan:
				if ok {
					for _, ev := range wresp.Events {
						node := &Node{Key: string(ev.Kv.Key), Value: string(ev.Kv.Value), Dir: false}
						var actionType ActionType
						switch ev.Type.String() {
						case "PUT":
							actionType = ActionPut
							if ev.Kv.Lease != 0 {
								node.TTL = c.getTTL(ev.Kv.Lease)
							}
							node.CreateRevision = ev.Kv.CreateRevision
							node.ModRevision = ev.Kv.ModRevision
						case "DELETE":
							actionType = ActionDelete
						}
						evt := &Event{Action: actionType, Node: node, WatchStatus: WatchStatusOK}
						ech <- evt
					}
				} else {
					time.Sleep(time.Second)
					_, _ = fmt.Fprintf(os.Stderr, "etcd '%s' watching channel was closed", key)
					clusterOk := true
					for _, ed := range c.client.Endpoints() {
						cx, cancel := context.WithTimeout(context.Background(), time.Second*3)
						if _, err := c.client.Status(cx, ed); err != nil {
							_, _ = fmt.Fprintf(os.Stderr, "etcd endpoint '%s' check error: %v", key, err)
							clusterOk = false
						}
						cancel()
					}
					if clusterOk {
						if recursive {
							watchChan = c.client.Watch(leaderCtx, key, clientv3.WithPrefix())
						} else {
							watchChan = c.client.Watch(leaderCtx, key)
						}
					}
				}
			case <-watchCtx.Done():
				close(ech)
				break L
			}
		}
	}()

	return ech
}

func (c *wclientv3) KeepAlive(ctx context.Context, sec int64) (stopCh chan bool, err error) {
	stopCh = make(chan bool)
	leaseResp, err := c.client.Grant(ctx, sec)
	if err != nil {
		return nil, err
	}
	ch, err := c.client.KeepAlive(ctx, leaseResp.ID)
	if err != nil {
		return nil, err
	}
	go func() {
		for range ch {

		}
		close(stopCh)
	}()
	return stopCh, nil
}

func (c *wclientv3) Close() error {
	return c.client.Close()
}

func (c *wclientv3) NewLocker(key string, ttl int64) (sync.Locker, error) {
	if ttl < 1 {
		ttl = 60
	}
	sess, err := concurrency.NewSession(c.client, concurrency.WithTTL(int(ttl)))
	if err != nil {
		return nil, err
	}

	lk := &wclientv3Locker{
		sess: sess,
		key:  key,
		ttl:  ttl,
		lk:   concurrency.NewLocker(sess, key),
	}

	return lk, nil
}

func (c *wclientv3) getTTL(lease int64) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	resp, err := c.client.Lease.TimeToLive(ctx, clientv3.LeaseID(lease))
	if err != nil {
		return 0
	}
	if resp.TTL == -1 {
		return 0
	}
	return resp.TTL
}
