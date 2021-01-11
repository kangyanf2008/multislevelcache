package multislevelcache

import (
	"errors"
	"github.com/coocood/freecache"
)

type Lv1Config struct {
	CacheSize     int   //缓存数量 default  1024*1024
	ExpireSeconds int64 //过期时间
}

type Lv1CacheDef struct {
	cache *freecache.Cache
	cfg *Lv1Config
}


//初始化一级缓存
func lv1Cache(cfg *Lv1Config) CacheDef {
	//默认1024*1024
	if cfg.CacheSize <= 0 {
		cfg.CacheSize = 1024*1024
	}
	//初始化freeCache实例,实始化实例
	cache := freecache.NewCache(cfg.CacheSize)
	//debug.SetGCPercent(20) go gc参数
	return &Lv1CacheDef{cache: cache, cfg:cfg}
}

//读取缓存值
func (ca *Lv1CacheDef) Get(key string) (string, error) {
	v, err := ca.cache.Get([]byte(string(key)))
	if err != nil {
		//如果1级缓存提示异常 Entry not found，则忽略
		if  err.Error() == LV1_CACHE_IGNORE_ERROR {
			return "",nil
		}
		return "", err
	}
	return string(v), err
}

/**
删除缓存
*/
func (ca *Lv1CacheDef) del(key string) error {
	isSuccess := ca.cache.Del([]byte(key))
	if isSuccess {
		return nil
	}
	return errors.New("delete Lv1CacheDef fail key=" + key)
}

/**
设置值，带过期时间
param1:key				参数key
param2:val				参数值
param2:expireSeconds	缓存参数过期时间（秒）,小于0为永不过期
return1:				异常返回错误
*/
func (ca *Lv1CacheDef) Set(key, val string, expireSeconds int64) error {
	var expire int
	if expireSeconds > 0 { //使用传入时间
		expire = int(expireSeconds)
	} else if ca.cfg.ExpireSeconds > 0 { //默认过期时间
		expire = int(ca.cfg.ExpireSeconds)
	}
	return ca.cache.Set([]byte(key), []byte(val), expire)
}

/**
清空缓存值，如果二级组态设置groupID,则进行二级缓存清除
*/
func (ca *Lv1CacheDef) clearAll() error {
	ca.cache.Clear()
	return nil
}