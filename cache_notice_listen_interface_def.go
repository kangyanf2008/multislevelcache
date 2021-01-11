package multislevelcache

/**
缓存一致性通知结构体定义
*/
type CacheNotice struct {
	CMD           byte   `json:"cmd"` //1添加到所级缓存，2更新所有级缓存，3删除所有级缓存，4清空所有级缓存，5添加一级缓存、6删除一级缓存
	Key           string `json:"key"`
	Value         string `json:"value"`
	expireSeconds int64  `json:"expire_seconds"` //过期时间
}

type CacheNoticeListenInterface interface {
	//发布集群更新缓存通知
	PubCacheNoticeCMD(notice *CacheNotice) error
	//订阅
	SubCacheNoticeCMD(cache *CacheManager)
}
