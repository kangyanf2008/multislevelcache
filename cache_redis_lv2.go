package multislevelcache

import (
	"errors"
	"github.com/go-redis/redis"
	"time"
)

type Lv2RedisConfig struct {
	ClusterClient *redis.ClusterClient
	Client        *redis.Client
	ExpireSeconds int64 //过期时间
	HashKey       string
	IsCluster     bool
}

type Lv2RedisCacheDef struct {
	cfg *Lv2RedisConfig
}

//初始化二级缓存
func lv2RedisCache(cfg *Lv2RedisConfig) (CacheDef, error) {
	if cfg.IsCluster && cfg.ClusterClient == nil {
		return nil, errors.New("Lv2RedisCacheDef init error，redis ClusterClient not init")
	}
	if !cfg.IsCluster && cfg.Client == nil {
		return nil, errors.New("Lv2RedisCacheDef init error，redis client not init")
	}
	return &Lv2RedisCacheDef{cfg: cfg}, nil
}

//读取缓存值
func (ca *Lv2RedisCacheDef) Get(key string) (string, error) {
	var stringCmd *redis.StringCmd
	//集群模式
	if ca.cfg.IsCluster {
		//如果配置HashKey,则从hash中读取
		if ca.cfg.HashKey != "" {
			stringCmd = ca.cfg.ClusterClient.HGet(ca.cfg.HashKey, key)
		} else {
			stringCmd = ca.cfg.ClusterClient.Get(key)
		}
	} else { //单机模式
		if ca.cfg.HashKey != "" {
			stringCmd = ca.cfg.Client.HGet(ca.cfg.HashKey, key)
		} else {
			stringCmd = ca.cfg.Client.Get(key)
		}
	}

	//判断是否为需要忽略异常 ｛redis: nil｝
	if stringCmd.Err() != nil && stringCmd.Err().Error() == LV2_CACHE_IGNORE_ERROR {
		return "", nil
	}
	return stringCmd.Val(), stringCmd.Err()
}

/**
删除缓存
*/
func (ca *Lv2RedisCacheDef) del(key string) error {
	var intCmd *redis.IntCmd
	//集群模式
	if ca.cfg.IsCluster {
		//如果配置HashKey,则从hash中读取
		if ca.cfg.HashKey != "" {
			intCmd = ca.cfg.ClusterClient.HDel(ca.cfg.HashKey, key)
		} else {
			intCmd = ca.cfg.ClusterClient.Del(key)
		}
	} else { //单机模式
		if ca.cfg.HashKey != "" {
			intCmd = ca.cfg.Client.HDel(ca.cfg.HashKey, key)
		} else {
			intCmd = ca.cfg.Client.Del(key)
		}
	}

	return intCmd.Err()
}

/**
设置值，带过期时间
param1:key				参数key
param2:val				参数值
param2:expireSeconds	缓存参数过期时间（秒）,小于0为永不过期
return1:				异常返回错误
*/
func (ca *Lv2RedisCacheDef) Set(key, val string, expireSeconds int64) error {
	var expire time.Duration
	if expireSeconds > 0 { //使用传入时间
		expire = time.Second * time.Duration(expireSeconds)
	} else if ca.cfg.ExpireSeconds > 0 { //默认过期时间
		expire = time.Second * time.Duration(ca.cfg.ExpireSeconds)
	}

	var statusCmd *redis.StatusCmd
	var boolCmd *redis.BoolCmd
	//集群模式
	if ca.cfg.IsCluster {
		//如果配置HashKey,则从hash中读取
		if ca.cfg.HashKey != "" {
			boolCmd = ca.cfg.ClusterClient.HSet(ca.cfg.HashKey, key, val)
		} else {
			statusCmd = ca.cfg.ClusterClient.Set(key, val, expire)
		}
	} else { //单机模式
		if ca.cfg.HashKey != "" {
			boolCmd = ca.cfg.Client.HSet(ca.cfg.HashKey, key, val)
		} else {
			statusCmd = ca.cfg.Client.Set(key, val, expire)
		}
	}

	if boolCmd != nil && ca.cfg.HashKey != "" {
		return boolCmd.Err()
	}
	if statusCmd != nil {
		return statusCmd.Err()
	}
	return nil
}

/**
清空redis缓存,如果为存储为hash类型，则清空里面所有key。
如果为string类型，则flushDB当前库
*/
func (ca *Lv2RedisCacheDef) clearAll() error {
	var statusCmd *redis.StatusCmd
	var intCmd *redis.IntCmd

	//集群模式
	if ca.cfg.IsCluster {
		if ca.cfg.HashKey != "" {
			if ca.cfg.HashKey != "" { //如果为hash存储格式，则删除hashHey下所有元素
				intCmd = ca.cfg.ClusterClient.Del(ca.cfg.HashKey)
			} else { //如果是普通string类型，则直接清空当前连接库
				statusCmd = ca.cfg.ClusterClient.FlushDB()
			}
		}
	} else {                      //单机模式
		if ca.cfg.HashKey != "" { //如果为hash存储格式，则删除hashHey下所有元素
			intCmd = ca.cfg.Client.Del(ca.cfg.HashKey)
		} else { //如果是普通string类型，则直接清空当前连接库
			statusCmd = ca.cfg.Client.FlushDB()
		}
	}

	//判断返回结果
	if intCmd != nil && ca.cfg.HashKey != "" {
		return intCmd.Err()
	}
	if statusCmd != nil {
		return statusCmd.Err()
	}

	return nil
}
