package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"jiayou_backend_spider/config"
	"jiayou_backend_spider/list"
	"jiayou_backend_spider/option"
	"jiayou_backend_spider/utils"
	"jiayou_backend_spider/utils/wait"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jinzhu/copier"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrBadSessionImpl = errors.New("bad session impl")

const internalExpiredCollection = "__internal_expired_collection__"
const internalExpiredKey = "__expired_at__"

var _ KV = &redisKVStor{}
var _ KV = &mongoKVStor{}

var _ SessionProvider = &redisProvider{}
var _ SessionProvider = &mongoProvider{}

type KVItem struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}
type KV interface {
	Set(key string, val any, option *option.Option) error
	Get(key string, val any, option *option.Option) error
	Del(key string, option *option.Option) error
	Exists(key string, option *option.Option) (bool, error)
	Keys(*option.Option) ([]string, error)
	Count(*option.Option) (int64, error)
	Clear(*option.Option) error
}

type redisValue struct {
	Value any `json:"value"`
}
type redisKVStor struct {
	ctx    context.Context
	client *redis.Client
}

func (stor *redisKVStor) marshal(value any) ([]byte, error) {
	return json.Marshal(redisValue{Value: value})
}
func (stor *redisKVStor) Set(key string, value any, option *option.Option) error {
	if stor.client != nil {
		if raw, err := stor.marshal(value); err != nil {
			return err
		} else {
			if cmd := stor.client.Set(stor.ctx, key, raw, 0); cmd.Err() != nil {
				return cmd.Err()
			}
		}
	}
	return nil
}
func (stor *redisKVStor) Get(key string, v any, option *option.Option) error {
	if stor.client != nil {
		if cmd := stor.client.Get(stor.ctx, key); cmd.Err() != nil {
			return cmd.Err()
		} else {
			if raw, err := cmd.Bytes(); err != nil {
				return err
			} else {
				var redisValue redisValue
				if err := json.Unmarshal(raw, &redisValue); err != nil {
					return err
				}
				return copier.CopyWithOption(v, redisValue.Value, copier.Option{DeepCopy: true})
			}
		}
	}
	return nil
}
func (stor *redisKVStor) Del(key string, option *option.Option) error {
	if stor.client != nil {
		if cmd := stor.client.Del(stor.ctx, key); cmd.Err() != nil {
			return cmd.Err()
		}
	}
	return nil
}
func (stor *redisKVStor) Exists(key string, option *option.Option) (bool, error) {
	if stor.client != nil {
		if c, err := stor.client.Exists(stor.ctx, key).Result(); err != nil {
			return false, err
		} else {
			return c > 0, nil
		}
	}
	return false, nil
}
func (stor *redisKVStor) Keys(option *option.Option) ([]string, error) {
	if stor.client != nil {
		var keys = "*"
		if v, ok := option.AsString("keys"); ok {
			keys = v
		}
		if c, err := stor.client.Keys(stor.ctx, keys).Result(); err != nil {
			return nil, err
		} else {
			return c, nil
		}
	}
	return nil, nil
}

func (stor *redisKVStor) Count(option *option.Option) (int64, error) {
	if stor.client != nil {
		if c, err := stor.Keys(option); err != nil {
			return 0, err
		} else {
			return int64(len(c)), nil
		}
	}
	return 0, nil
}
func (stor *redisKVStor) Clear(option *option.Option) error {
	if stor.client != nil {
		if _, ok := option.AsString("keys"); !ok {
			if cmd := stor.client.FlushDB(stor.ctx); cmd.Err() != nil {
				return cmd.Err()
			}
		} else {
			if keys, err := stor.Keys(option); err != nil {
				return err
			} else {
				for _, key := range keys {
					if err := stor.Del(key, nil); err != nil {
						return err
					}
				}
			}
		}

	}
	return nil
}

type mongoKVStor struct {
	ctx        context.Context
	collection *mongo.Collection
}

func (stor *mongoKVStor) Set(key string, value any, option *option.Option) error {
	if stor.collection != nil {
		var upsert = true
		_, err := stor.collection.UpdateOne(stor.ctx,
			bson.M{"key": key},
			bson.M{"$set": bson.E{Key: key, Value: value}},
			&options.UpdateOptions{Upsert: &upsert})
		if err != nil {
			return err
		}
	}
	return nil
}

func (stor *mongoKVStor) Get(key string, v any, option *option.Option) error {
	if stor.collection != nil {
		var filter = bson.M{"key": key}
		var r primitive.M
		if err := stor.collection.FindOne(stor.ctx, filter).Decode(&r); err != nil {
			return err
		}
		return copier.CopyWithOption(v, r["value"], copier.Option{DeepCopy: true})
	}
	return nil
}
func (stor *mongoKVStor) Del(key string, option *option.Option) error {
	if stor.collection != nil {
		var filter = bson.M{"key": key}
		if _, err := stor.collection.DeleteOne(stor.ctx, filter); err != nil {
			return err
		}
	}
	return nil
}
func (stor *mongoKVStor) Exists(key string, option *option.Option) (bool, error) {
	if stor.collection != nil {
		var filter = bson.M{"key": key}
		r := stor.collection.FindOne(stor.ctx, filter)
		if r.Err() != nil {
			if errors.Is(r.Err(), mongo.ErrNoDocuments) {
				return false, nil
			}
			return false, r.Err()
		}
		return true, nil
	}
	return false, nil
}
func (stor *mongoKVStor) Keys(option *option.Option) ([]string, error) {
	if stor.collection != nil {
		var limit int64
		if option != nil {
			if val, ok := option.AsInt("limit"); ok && val > 0 {
				limit = val
			}
		}
		var filter = bson.M{}
		cursor, err := stor.collection.Find(stor.ctx, filter, &options.FindOptions{Limit: &limit})
		if err != nil {
			return nil, err
		}
		var values bson.D
		if err := cursor.All(stor.ctx, &values); err != nil {
			return nil, err
		}
		var keys []string
		for _, v := range values {
			keys = append(keys, v.Key)
		}
		return keys, nil
	}
	return nil, nil
}
func (stor *mongoKVStor) Clear(option *option.Option) error {
	if stor.collection != nil {
		_, err := stor.collection.DeleteMany(stor.ctx, bson.M{"key": bson.M{"$exists": true}})
		if err != nil {
			return err
		}
	}
	return nil
}
func (stor *mongoKVStor) Count(option *option.Option) (int64, error) {
	if stor.collection != nil {
		return stor.collection.CountDocuments(stor.ctx, bson.M{"key": bson.M{"$exists": true}})
	}
	return 0, nil
}

type redisIntervalEventWrapper struct {
	KV
	ctx    context.Context
	client *redis.Client
}

func (event redisIntervalEventWrapper) Expired(key string, expired time.Duration, opt *option.Option) error {
	if ok, _ := event.Exists(key, nil); ok {
		return event.client.ZAdd(event.ctx, internalExpiredCollection, &redis.Z{Score: float64(time.Now().Add(expired).UnixMilli()), Member: key}).Err()
	}
	return nil
}

type redisProvider struct {
	sess   *Session
	opts   *redis.Options
	client *redis.Client
}

func (provider *redisProvider) expired() {
	if provider.client != nil {
		items, _ := provider.client.ZRangeByScore(provider.sess.opts.Ctx, internalExpiredCollection, &redis.ZRangeBy{
			Max: strconv.FormatInt(time.Now().UnixMilli(), 10),
		}).Result()
		var members []any
		for _, item := range items {
			members = append(members, item)
			//notify
			if result := provider.client.Get(provider.sess.opts.Ctx, item); result.Err() == nil {
				var redisValue redisValue
				if raw, err := result.Bytes(); err == nil && json.Unmarshal(raw, &redisValue) == nil {
					provider.sess.opts.OnKeyExpired(KVItem{Key: item, Value: redisValue.Value})
				}
			}
		}
		if len(members) > 0 {
			provider.client.ZRem(provider.sess.opts.Ctx, internalExpiredCollection, members...)
			provider.client.Del(provider.sess.opts.Ctx, items...)
		}
	}
}
func (provider *redisProvider) Prepare(sess *Session) error {
	if sess == nil {
		return errors.New("bad session")
	}
	opts, err := redis.ParseURL(sess.opts.Uri)
	if err != nil {
		return err
	}
	provider.opts = opts
	if provider.sess.opts.OnKeyExpired != nil {
		// expired check coroutine
		go func() {
			//check expire default 100ms
			wait.Wait(func(_ *wait.Context) (any, bool) {
				provider.expired()
				return nil, false
			}, wait.WithCtx(provider.sess.opts.Ctx))
		}()
	}
	return nil
}
func (provider *redisProvider) Ping() bool {
	if provider.client != nil {
		if err := provider.client.Ping(provider.sess.opts.Ctx).Err(); err != nil {
			provider.client.Close()
			provider.client = nil
			return false
		}
		return true
	}
	return false
}
func (provider *redisProvider) Connect() error {
	if provider.client == nil {
		var client = redis.NewClient(provider.opts)
		if r := client.Ping(provider.sess.opts.Ctx); r.Err() != nil {
			return r.Err()
		}
		provider.client = client
	}
	return nil
}
func (provider *redisProvider) KV() KV {
	return redisIntervalEventWrapper{
		KV: &redisKVStor{
			ctx:    provider.sess.opts.Ctx,
			client: provider.client,
		},
		ctx:    provider.sess.opts.Ctx,
		client: provider.client,
	}
}
func (provider *redisProvider) Raw() any {
	return provider.client
}

type mongoIntervalEventWrapper struct {
	KV
	collection *mongo.Collection
}

func (event mongoIntervalEventWrapper) Expired(key string, expired time.Duration, opt *option.Option) error {
	return (&mongoKVStor{collection: event.collection}).Set(key, bson.M{internalExpiredKey: time.Now().Add(expired)}, nil)
}

type mongoProvider struct {
	sess              *Session
	dbName            string
	collectionName    string
	client            *mongo.Client
	collection        *mongo.Collection
	collectionExpired *mongo.Collection
	opts              *options.ClientOptions
}

func (provider *mongoProvider) expired() int64 {
	var c int64
	if provider.collectionExpired != nil && provider.collection != nil {
		if r, e := provider.collectionExpired.Find(provider.sess.opts.Ctx, bson.M{"value." + internalExpiredKey: bson.M{"$lte": time.Now()}}); e == nil {
			defer r.Close(provider.sess.opts.Ctx)
			for r.Next(provider.sess.opts.Ctx) {
				var m bson.M
				if err := r.Decode(&m); err == nil {
					var key = m["key"].(string)
					//unmarshal
					if c := provider.collection.FindOne(provider.sess.opts.Ctx, bson.M{"key": key}); c.Err() == nil {
						if err := c.Decode(&m); err == nil {
							if raw, err := bson.Marshal(m); err == nil {
								var m map[string]any
								if err := bson.Unmarshal(raw, &m); err == nil {
									provider.sess.opts.OnKeyExpired(KVItem{Key: key, Value: m["value"]})
								}
							}
						}

					}
					if rr, err := provider.collection.DeleteOne(provider.sess.opts.Ctx, bson.M{"key": key}); err == nil {
						c += rr.DeletedCount
					}
					provider.collectionExpired.DeleteOne(provider.sess.opts.Ctx, bson.M{"key": key})

				}
			}
		}
	}
	return c
}
func (provider *mongoProvider) Prepare(sess *Session) error {
	if sess == nil {
		return errors.New("bad session")
	}
	_uri, err := url.Parse(sess.opts.Uri)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(_uri.Scheme, "db") {
		_uri.Scheme = _uri.Scheme + "db"
	}
	opts := options.Client().ApplyURI(_uri.String())
	if opts.Validate() != nil {
		return opts.Validate()
	}
	provider.dbName = _uri.Path[1:]
	provider.collectionName = _uri.Query().Get("collection")
	if provider.collectionName == "" {
		return errors.New("mongo collection not found")
	}
	//expired check coroutine
	if provider.sess.opts.OnKeyExpired != nil {
		go func() {
			wait.Wait(func(_ *wait.Context) (any, bool) {
				return provider.expired(), false
			}, wait.WithCtx(sess.opts.Ctx),
				wait.WithBackoffFunc(utils.FixedDuration(sess.opts.KeyExpiredInterval)))
		}()
	}
	provider.opts = opts
	return nil
}

func (provider *mongoProvider) Ping() bool {
	if provider.client != nil {
		if err := provider.client.Ping(provider.sess.opts.Ctx, nil); err != nil {
			provider.client.Disconnect(provider.sess.opts.Ctx)
			provider.client = nil
			provider.collection = nil
			provider.collectionExpired = nil
			return false
		}
		return true
	}
	return false
}
func (provider *mongoProvider) Connect() error {
	if provider.client == nil {
		db, err := mongo.Connect(provider.sess.opts.Ctx, provider.opts)
		if err != nil {
			return err
		}
		provider.client = db
		provider.collection = provider.client.Database(provider.dbName).Collection(provider.collectionName)
		if provider.sess.opts.OnKeyExpired != nil {
			provider.collectionExpired = provider.client.Database(provider.dbName).Collection(internalExpiredCollection)
		}
	}
	return nil
}
func (provider *mongoProvider) KV() KV {
	return mongoIntervalEventWrapper{
		KV:         &mongoKVStor{collection: provider.collection, ctx: provider.sess.opts.Ctx},
		collection: provider.collectionExpired,
	}
}

func (provider *mongoProvider) Client() any {
	return provider.client
}

type Scheme string

const (
	Redis Scheme = "redis"
	Mongo Scheme = "mongo"
)

type OnKeyExpired func(KVItem)

var DefaultOptions = func() *Options {
	return config.TryValidate(&Options{
		Ctx: context.Background(),
	}, nil)
}

type Options struct {
	Ctx                context.Context
	OnKeyExpired       OnKeyExpired
	Name               string        `json:"name" validate:"required"`
	Uri                string        `json:"uri" validate:"required"`
	PingInterval       time.Duration `json:"ping_interval" validate:"gt=0" default:"15s"`
	KeyExpiredInterval time.Duration `json:"key_expired_interval" validate:"required_with=Type,gt=0" default:"1s"`
}

type Session struct {
	provider   SessionProvider
	latestPing time.Time
	opts       *Options
}

func (session *Session) Scheme() Scheme {
	before, _, _ := strings.Cut(session.opts.Uri, "://")
	return Scheme(before)
}
func (session *Session) KV() KV {
	return session.provider.KV()
}
func (session *Session) Client() (any, error) {
	if v, ok := session.provider.(Client); ok {
		return v.Client(), nil
	}
	return nil, ErrBadSessionImpl

}

type Client interface {
	Client() any
}
type SessionProvider interface {
	Prepare(*Session) error
	Ping() bool
	Connect() error
	KV() KV
}
type Expired interface {
	Expired(string, time.Duration, *option.Option) error
}

type Sessions struct {
	ss    *list.List[*Session]
	lck   sync.Mutex
	index int
}

func (ss *Sessions) Add(sess ...*Session) error {
	for _, s := range sess {
		if s != nil {
			if ss.ss.ContainF(func(session *Session) bool {
				return session.opts.Name == s.opts.Name
			}) {
				return fmt.Errorf("session[%s] existed", s.opts.Name)
			}
		}
	}
	return nil
}
func (ss *Sessions) Size() int {
	return ss.ss.Size()
}
func (ss *Sessions) Clear() *Sessions {
	ss.ss.Clear()
	return ss
}
func (ss *Sessions) Top() { ss.index = 0 }
func (ss *Sessions) Last() {
	if ss.Size() == 0 {
		return
	}
	ss.index = ss.Size() - 1
}
func (ss *Sessions) Index(index int) {
	if ss.Size() == 0 || index >= ss.Size() || index < 0 {
		return
	}
	ss.index = index
}
func (ss *Sessions) ForName(name string) *Sessions {
	var sess = ss.ss.Filter(func(ds *Session) bool {
		return ds.opts.Name == name
	})
	ss.ss.Clear()
	ss.ss = list.New[*Session]()
	ss.ss.PushR(sess...)
	return ss
}
func (ss *Sessions) ForHost(host string) *Sessions {
	var sess = ss.ss.Filter(func(ds *Session) bool {
		return strings.Contains(ds.opts.Uri, host)
	})
	ss.ss.Clear()
	ss.ss = list.New[*Session]()
	ss.ss.PushR(sess...)
	return ss
}
func (ss *Sessions) ForScheme(scheme Scheme) *Sessions {
	var sess = ss.ss.Filter(func(ds *Session) bool {
		return ds.Scheme() == scheme
	})
	ss.ss.Clear()
	ss.ss = list.New[*Session]()
	ss.ss.PushR(sess...)
	return ss
}
func (ss *Sessions) LookUp() (*Session, error) {
	if ss.ss.Size() == 0 {
		return nil, errors.New("db session not found")
	}
	defer func() {
		ss.index = 0
	}()
	var firstSession = ss.ss.Index(ss.index)
	if ss.lck.TryLock() {
		if firstSession.latestPing.IsZero() || time.Since(firstSession.latestPing) > firstSession.opts.PingInterval {
			if !firstSession.provider.Ping() {
				if err := firstSession.provider.Connect(); err != nil {
					return nil, err
				}
				firstSession.latestPing = time.Now()
			}
		}
		ss.lck.Unlock()
	}
	return firstSession, nil
}

func NewSession(opts *Options) (*Session, error) {
	var session = &Session{
		opts: opts,
	}
	if session.opts == nil {
		session.opts = DefaultOptions()
	}
	_url, err := url.Parse(session.opts.Uri)
	if err != nil {
		return nil, err
	}
	switch session.Scheme() {
	case Redis:
		session.provider = &redisProvider{sess: session}
	case Mongo:
		session.provider = &mongoProvider{sess: session}
	}
	if session.provider != nil {
		if err := session.provider.Prepare(session); err != nil {
			return nil, err
		}
		if err := session.provider.Connect(); err != nil {
			return nil, err
		}
		return session, nil
	}
	return nil, fmt.Errorf("db[%s] not support", _url.Scheme)
}
func NewSessions() *Sessions {
	return &Sessions{ss: list.New[*Session]()}
}
