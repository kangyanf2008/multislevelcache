package multislevelcache

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis"
)

//redis配置信息
type CacheNoticeListenRedisConfig struct {
	ClusterClient *redis.ClusterClient //集群客户端
	Client        *redis.Client        //客户端
	IsCluster     bool                 //是否为集群模式
	PubSubChannel string               //发布订阅管道
}

type cacheNoticeListenImplRedis struct {
	cfg *CacheNoticeListenRedisConfig
}

//redis实现缓存更新通知和监听
func NewCacheNoticeListenImplRedis(cfg *CacheNoticeListenRedisConfig) (CacheNoticeListenInterface, error) {
	//检查参数是否合法
	if cfg == nil {
		return nil, errors.New("NewCacheNoticeListenImplRedis param cfg is nil")
	}
	if cfg.IsCluster {
		if cfg.ClusterClient == nil {
			return nil, errors.New("NewCacheNoticeListenImplRedis param cfg.ClusterClient is nil")
		}
	} else if cfg.Client == nil {
		return nil, errors.New("NewCacheNoticeListenImplRedis param cfg.Client is nil")
	}

	if cfg.PubSubChannel == "" {
		cfg.PubSubChannel = "multi_level_cache_channel"
	}
	return &cacheNoticeListenImplRedis{cfg: cfg}, nil
}

//推送缓存更新通知
func (listen *cacheNoticeListenImplRedis) PubCacheNoticeCMD(notice *CacheNotice) error {
	if notice == nil {
		return errors.New("PubCacheNoticeCMD param is nil")
	}
	if listen.cfg == nil {
		return errors.New("cacheNoticeListenImplRedis.cfg is nil")
	}

	//进行json格式化
	noticeJson, err := json.Marshal(notice)
	if err != nil {
		return err
	}
	if listen.cfg.IsCluster { //集群模式
		if listen.cfg.ClusterClient != nil {
			listen.cfg.ClusterClient.Publish(listen.cfg.PubSubChannel, noticeJson)
		}
	} else {
		if listen.cfg.Client != nil {
			listen.cfg.Client.Publish(listen.cfg.PubSubChannel, noticeJson)
		}
	}
	return errors.New("cacheNoticeListenImplRedis cfg not set redis client connect")
}

//监听缓存更新通知
func (listen *cacheNoticeListenImplRedis) SubCacheNoticeCMD(cacheManager *CacheManager) {
	if cacheManager != nil && listen.cfg != nil {
		var sub *redis.PubSub
		if listen.cfg.IsCluster && listen.cfg.ClusterClient != nil {
			sub = listen.cfg.ClusterClient.Subscribe(listen.cfg.PubSubChannel)
		} else if listen.cfg.Client != nil {
			sub = listen.cfg.Client.Subscribe(listen.cfg.PubSubChannel)
		}
		if sub != nil {
			fmt.Println("begin subscribe [" + listen.cfg.PubSubChannel + "]")
			for {
				v, err := sub.ReceiveMessage()
				if err != nil {
					fmt.Println(err)
				}
				if v != nil {
					noticeJson := v.Payload
					cacheNotice := &CacheNotice{}
					err = json.Unmarshal([]byte(noticeJson), cacheNotice)
					//通知缓存管理器，数据已经发生变化
					if err != nil {
						fmt.Println(err)
						continue
					}
					cacheManager.CacheNotice(cacheNotice)
				}
			}
		}
	}

}
