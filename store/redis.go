package store

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"time"
)

type RedisConf struct {
	Addr string `yaml:"addr"`
	Auth string `yaml:"auth"`
	DB   int    `yaml:"db"`
}

type redisStore struct {
	*redis.Client
}

func NewRedisStore(opt *redis.Options) *redisStore {
	client := redis.NewClient(opt)
	result, err := client.Ping().Result()
	if err != nil || result != "PONG" {
		panic(err)
	}
	return &redisStore{client}
}
func NewRedisStoreDefault(conf *RedisConf) *redisStore {
	opt, err := redis.ParseURL(fmt.Sprintf("redis://:%s@%s/%d", conf.Auth, conf.Addr, conf.DB))
	if err != nil {
		panic(err)
	}
	opt.ReadTimeout = time.Second * 2
	opt.WriteTimeout = time.Second * 2
	opt.PoolSize = 10

	return NewRedisStore(opt)
}

func (c *redisStore) Set(key string, k string, val *ResponseCache, ttl time.Duration) error {
	v, err := json.Marshal(val)
	if err != nil {
		return err
	}
	_, err = c.Client.TxPipelined(func(pipeline redis.Pipeliner) error {
		err := pipeline.HSet(key, k, v).Err()
		er2 := pipeline.Expire(key, ttl).Err()
		if er2 != nil {
			err = er2
		}
		return err
	})
	return err
}

func (c *redisStore) Get(key string, k string, val *ResponseCache) error {
	res, err := c.Client.TxPipelined(func(pipeline redis.Pipeliner) error {
		err := pipeline.HGet(key, k).Err()
		er2 := pipeline.TTL(key).Err()
		if er2 != nil {
			err = er2
		}
		return err
	})
	if err != nil {
		return err
	}
	r, err := (res[0]).(*redis.StringCmd).Bytes()
	if err != nil {
		return err
	}
	err = json.Unmarshal(r, val)
	if err != nil {
		return err
	}
	val.Expire = (res[1]).(*redis.DurationCmd).Val()
	return nil
}

func (c *redisStore) Remove(key string) error {
	_, err := c.Client.Del(key).Result()
	return err
}
