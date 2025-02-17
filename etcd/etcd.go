package etcd

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/wecisecode/util/crypto"
	"github.com/wecisecode/util/merrs"
)

var EncryptKey = ""

const (
	WatchStatusOK WatchStatus = "ok"

	//ActionSet ActionType = "set"
	//ActionUpdate ActionType = "update"
	//ActionExpire ActionType = "expire"

	ActionPut    ActionType = "put"
	ActionDelete ActionType = "delete"

	Separator = "/"
)

var (
	singleCli   Client
	singleCliMu sync.Mutex

	ClientVersion = 3
)

type Client interface {
	connect(endpoints []string, opts ...opt) error
	Put(key, val string) error
	PutTTL(key, val string, sec int64) error
	Get(key string) (val string, err error)
	GetNode(key string) (node *Node, err error)
	Delete(key string) error
	DeleteDir(key string) error
	Watch(ctx context.Context, key string, recursive bool) (ech chan *Event)
	KeepAlive(ctx context.Context, sec int64) (stopCh chan bool, err error)
	Close() error

	NewLocker(key string, ttl int64) (sync.Locker, error) // ttl: not less than 5
}

type Node struct {
	Key            string  `json:"key,omitempty"`
	Value          string  `json:"value,omitempty"`
	Dir            bool    `json:"dir"`
	Nodes          []*Node `json:"nodes,omitempty"`
	TTL            int64   `json:"ttl,omitempty"`
	CreateRevision int64   `json:"createrevision,omitempty"`
	ModRevision    int64   `json:"modrevision,omitempty"`
}

type Event struct {
	Action      ActionType
	Node        *Node
	WatchStatus WatchStatus
}

type WatchStatus string
type ActionType string

type option struct {
	user string
	pass string
	auth bool
}

type opt func(*option)

func optAuthEnable(enable bool) opt {
	return func(o *option) {
		o.auth = enable
	}
}

func optUserPass(user, pass string) opt {
	return func(o *option) {
		o.user = user
		o.pass = pass
	}
}

func New() (Client, error) {
	etcdPath := os.Getenv("ETCDPATH")
	if etcdPath == "" {
		return nil, errors.New("ETCDPATH not set")
	}
	etcdUser := os.Getenv("ETCDUSER")
	etcdPass := os.Getenv("ETCDPASS")
	return newClient(etcdPath, etcdUser, etcdPass)
}

func NewClient(addrs, user, pass string) (Client, error) {
	return newClient(addrs, user, pass)
}

func newClient(addrs, user, pass string) (Client, error) {
	var cli Client
	switch ClientVersion {
	case 2:
		//cliv2 := &wclientv2{}
		//cli = cliv2
		return nil, errors.New("unsupported version: v" + strconv.Itoa(ClientVersion))
	case 3:
		cliv3 := &wclientv3{}
		cli = cliv3
	default:
		cliv3 := &wclientv3{}
		cli = cliv3
	}
	var endpoints []string
	etcdPath := addrs
	if etcdPath == "" {
		return nil, errors.New("addrs is empty")
	}
	endpoints = strings.Split(etcdPath, ",")

	var opts []opt
	etcdUser := user
	etcdPass := pass
	if etcdUser != "" && etcdPass != "" {
		if EncryptKey == "" {
			return nil, merrs.New("need etcd.EncryptKey")
		}
		etcdPass = crypto.AesDecrypt(etcdPass, EncryptKey)
		opts = append(opts, optAuthEnable(true), optUserPass(etcdUser, etcdPass))
	}
	if err := cli.connect(endpoints, opts...); err != nil {
		return nil, err
	}
	return cli, nil
}

// Get singleton client
func Get() (Client, error) {
	singleCliMu.Lock()
	defer singleCliMu.Unlock()

	var err error
	if singleCli == nil {
		if singleCli, err = New(); err != nil {
			return nil, err
		}
	}
	return singleCli, nil
}

func Set(c Client) {
	singleCliMu.Lock()
	defer singleCliMu.Unlock()

	singleCli = c
}
