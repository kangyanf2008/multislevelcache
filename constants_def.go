package multislevelcache

const (
	//默认异步同步管理大小
	NO_SYNC_FROM_LV3_LOAD_KEYS_CHANNEL int = 1024

	////1添加到所级缓存，2更新所有级缓存，3删除所有级缓存，4清空所有级缓存，5添加一级缓存、6删除一级缓存
	CACHE_NOTICE_CMD_1 byte = 1
	CACHE_NOTICE_CMD_2 byte = 2
	CACHE_NOTICE_CMD_3 byte = 3
	CACHE_NOTICE_CMD_4 byte = 4
	CACHE_NOTICE_CMD_5 byte = 5
	CACHE_NOTICE_CMD_6 byte = 6
	CACHE_NOTICE_CMD_7 byte = 7

	//error def
	LV1_CACHE_IGNORE_ERROR  = "Entry not found" //一级缓存忽略异常
	LV2_CACHE_IGNORE_ERROR  = "redis: nil" //二级缓存忽略异常

)
