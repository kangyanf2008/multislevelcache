package multislevelcache

import (
	"errors"
	"sync"
	"time"
)

/**
缓存管理结构体
 */
type CacheManager struct {
	lv1               CacheDef     //一级缓存
	lv2               CacheDef     //二级缓存
	lv3               CacheLv3Def  //三级缓存
	updateLv1         sync.RWMutex //锁
	updateLv2         sync.RWMutex //锁2
	loadKeysFromLv3   chan string
	cacheNoticeListen CacheNoticeListenInterface //缓存订阅与发布通知接口
}


//使用工厂方法，创建缓存管理工具
//func CacheFactory(lv1Cfg *Lv1Config, lv2Cfg *Lv2RedisConfig, lv3Cfg *Lv3Config, noSyncFromLv3loadKeysBuffer int) (*CacheManager, error) {
func CacheFactory(lv1Cfg *Lv1Config, lv2Cfg *Lv2RedisConfig, lv3Cache CacheLv3Def, noSyncFromLv3loadKeysBuffer int,
	cacheNoticeListen CacheNoticeListenInterface) (*CacheManager, error) {
	ca := &CacheManager{
	}

	//1级缓存初始化
	if lv1Cfg != nil {
		ca.lv1 = lv1Cache(lv1Cfg)
	}

	//2级缓存初始化
	if lv2Cfg != nil {
		lv2Cache, err:= lv2RedisCache(lv2Cfg)
		if err != nil {
			return ca, err
		}
		ca.lv2 = lv2Cache
	}

	//3级缓存初始化
	//lv3Cache := lv3Cache(lv3Cfg)

	//如果配置三级缓存
	if lv3Cache != nil {
		if noSyncFromLv3loadKeysBuffer <= 0 { //如果小于等于0，则使用默认值1024
			noSyncFromLv3loadKeysBuffer = NO_SYNC_FROM_LV3_LOAD_KEYS_CHANNEL
		}
		ca.loadKeysFromLv3=make(chan string, noSyncFromLv3loadKeysBuffer)
		ca.lv3 = lv3Cache
		//异步加载缓存
		go ca.LoadFromLv3(lv1Cfg.ExpireSeconds, lv2Cfg.ExpireSeconds)
	}

	//启用协和监听缓存更新通知，缓存内容变化后，则进行缓存更新
	if cacheNoticeListen != nil {
		ca.cacheNoticeListen = cacheNoticeListen
		go ca.cacheNoticeListen.SubCacheNoticeCMD(ca)
	}
	return ca, nil
}

/**
查询缓存数据
*/
func (ca *CacheManager) Get(key string, syncReload bool, expireSeconds int64) (string, error) {
	//如果开启一级缓存，则从一级缓存取
	if ca.lv1 != nil {
		return ca.getFromLv1(key, syncReload, expireSeconds)
	}
	//如果开启二级缓存，则从二级缓存取
	if ca.lv2 != nil {
		return ca.getFromLv2(key, syncReload, expireSeconds)
	}
	//如果开启三级缓存，则从三级缓存取
	if ca.lv3 != nil {
		return ca.getFromLv3(key, syncReload)
	}
	return "", nil
}

//查询1级缓存
func (ca *CacheManager) getFromLv1(key string, syncReload bool, expireSeconds int64) (string, error) {
	//一级缓存查询
	v, e := ca.lv1.Get(key)
	if e != nil {
		return "", e
	}
	if v != "" {
		return v, nil
	}

	//二级缓存查询
	if ca.lv2 != nil {
		ca.updateLv1.Lock()
		defer ca.updateLv1.Unlock()
		//一级缓重读
		v, e = ca.lv1.Get(key)
		if e != nil {
			return "", e
		}
		if v != "" {
			return v, nil
		}

		//二级缓存查询
		v, e = ca.getFromLv2(key, syncReload, expireSeconds)
		if e != nil {
			return "", e
		}
		if ca.lv1 != nil && v != "" {
			ca.lv1.Set(key, v, expireSeconds)
			return v, nil
		}
	}

	//三级缓存查询，如果未开启二级缓存，则进行三级缓存查询
	if ca.lv2 == nil && ca.lv3 != nil {
		ca.updateLv1.Lock()
		defer ca.updateLv1.Unlock()

		v, e = ca.getFromLv3(key, syncReload)
		if e != nil {
			return "", e
		}

		//设置一级缓存
		if v != "" {
			ca.lv1.Set(key, v, expireSeconds)
		}
		return v, nil
	}
	return "", nil
}

func (ca *CacheManager) getFromLv2(key string, syncReload bool, reloadExpireSeconds int64) (string, error) {
	//查询二级缓存
	v, e := ca.lv2.Get(key)
	if e != nil {
		return "", e
	}
	if v != "" {
		return v, nil
	}

	//三级缓存查询
	if ca.lv3 != nil {

		//异步加载
		if !syncReload {
			//放入队列异步加载
			//TODO 解决队列满问题
			ca.loadKeysFromLv3 <- key
			return "", nil
		}

		ca.updateLv2.Lock()
		defer ca.updateLv2.Unlock()
		//二级缓存重读
		v, e = ca.lv2.Get(key)
		if e != nil {
			return "", e
		}
		//三级缓存读取
		v, e = ca.lv3.Get(key)
		if e != nil {
			return "", e
		}
		//进行二级缓存保存
		if v != "" {
			ca.lv2.Set(key, v, reloadExpireSeconds)
			return v, nil
		}
	}
	return "", nil
}

//从三级缓存异步加载一二级缓存
func (ca *CacheManager) LoadFromLv3(lv1ExpireSeconds, lv2ExpireSeconds int64) {
	//异步加载
	for {
		select {
		case key := <-ca.loadKeysFromLv3:
			//三级缓存读取
			v, e := ca.lv3.Get(key)
			if e != nil {
				//todo 记录日志
			}
			//缓存到二级缓存
			if v != "" && ca.lv2 != nil {
				ca.lv2.Set(key, v, lv2ExpireSeconds)
			}
			//缓存到一级缓存
			if v != "" && ca.lv1 != nil {
				ca.lv1.Set(key, v, lv1ExpireSeconds)
			}
		default: //无内容，则进行睡眠1秒
			time.Sleep(time.Second)
		}
	}
}

//从三级缓存查询数据
func (ca *CacheManager) getFromLv3(key string, syncReload bool) (string, error) {
	return ca.lv3.Get(key)
}

//添加或更新缓存
func (ca *CacheManager) cacheAdd(notice *CacheNotice) error {
	var err error
	//先更新3级缓存，再更新2级缓存，最后更新3级缓存
	if ca.lv3 != nil {
		err = ca.lv3.Set(notice.Key, notice.Value, notice.expireSeconds)
	}
	if err != nil {
		return err
	}
	if ca.lv2 != nil {
		err = ca.lv2.Set(notice.Key, notice.Key, notice.expireSeconds)
	}
	if err != nil {
		return err
	}
	if ca.lv1 != nil {
		err = ca.lv1.Set(notice.Key, notice.Key, notice.expireSeconds)
	}
	return err
}

//更新缓存
func (ca *CacheManager) CacheDel(notice *CacheNotice) error {
	var err error
	//先更新3级缓存，再更新2级缓存，最后更新3级缓存
	if ca.lv3 != nil {
		err = ca.lv3.Del(notice.Key)
	}
	if err != nil {
		return err
	}
	if ca.lv2 != nil {
		err = ca.lv2.del(notice.Key)
	}
	if err != nil {
		return err
	}
	if ca.lv1 != nil {
		err = ca.lv1.del(notice.Key)
	}
	return err
}

//清空缓存
func (ca *CacheManager) CacheClearAll(notice *CacheNotice) error {
	var err error
	//先更新3级缓存，再更新2级缓存，最后更新3级缓存
	if ca.lv3 != nil {
		err = ca.lv3.ClearAll()
	}
	if err != nil {
		return err
	}
	if ca.lv2 != nil {
		err = ca.lv2.clearAll()
	}
	if err != nil {
		return err
	}
	if ca.lv1 != nil {
		err = ca.lv1.clearAll()
	}
	return err
}

//通知回调函数
func (ca *CacheManager) CacheNotice(notice *CacheNotice) error {
	switch notice.CMD {//1添加到所级缓存，2更新所有级缓存，3删除所有级缓存，4清空所有级缓存，5添加一级缓存、6删除一级缓存
	case CACHE_NOTICE_CMD_1:
		return ca.cacheAdd(notice)
	case CACHE_NOTICE_CMD_2:
		return ca.cacheAdd(notice)
	case CACHE_NOTICE_CMD_3:
		return ca.CacheDel(notice)
	case CACHE_NOTICE_CMD_4:
		return ca.CacheClearAll(notice)
	case CACHE_NOTICE_CMD_5:
		return ca.CacheLv1(notice.Key, notice.Value, notice.expireSeconds)
	case CACHE_NOTICE_CMD_6:
		return ca.CacheDelLv1(notice.Key)
	default:
		return errors.New("not def CMD=" + string(notice.CMD))
	}
}

//发布缓存通知
func (ca *CacheManager) PubCacheNotice(notice *CacheNotice) error {
	if ca.cacheNoticeListen == nil {
		return errors.New("CacheNoticeListenInterface not def impl")
	}
	return ca.cacheNoticeListen.PubCacheNoticeCMD(notice)
}

//缓存1级缓存
func (ca *CacheManager)  CacheLv1(key, value string, expireSeconds int64 ) error {
	return ca.lv1.Set(key, value, expireSeconds)
}

//删除1级缓存
func (ca *CacheManager)  CacheDelLv1(key string) error {
	return ca.lv1.del(key)
}

//缓存2,1级缓存
func (ca *CacheManager)  CacheLv1LV2(key, value string, expireSeconds int64 ) error {
	err := ca.lv2.Set(key, value, expireSeconds)
	if err != nil {
		return err
	}
	return ca.CacheLv1(key, value, expireSeconds)
}

//删除2,1级缓存
func (ca *CacheManager)  CacheDelLV2Lv1(key string) error {
	err := ca.lv2.del(key)
	if err != nil {
		return err
	}
	return ca.CacheDelLv1(key)
}