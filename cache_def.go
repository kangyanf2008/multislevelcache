package multislevelcache

type CacheDef interface{

	/**
	查询缓存数据
	param1: key 		缓存key
	return1: 			缓存值
	return2: 			错误
	*/
	Get(key string) (string, error)

	/**
	删除缓存
	*/
	del(key string) error

	/**
	设置值，带过期时间
	param1:key				参数key
	param2:val				参数值
	param2:expireSeconds	缓存参数过期时间（秒）,小于0为永不过期
	return1:				异常返回错误
	*/
	Set(key, val string, expireSeconds int64) error

	/**
	清空缓存值，如果二级组态设置groupID,则进行二级缓存清除
	*/
	clearAll() error
}

//提供第三方接口对接
type CacheLv3Def interface {
	Get(key string) (string, error)
	Set(key, val string, expireSeconds int64) error
	Del(key string) error
	ClearAll() error
}