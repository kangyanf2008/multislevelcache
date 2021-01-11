package multislevelcache

/*
type Lv3Config struct {
	cfg []interface{}
	getFun func(key string, value ...interface{})(string, error)
}

type lv3FuncDef struct {
	cfg *Lv3Config
}

//初始化三级数据查询实现
func lv3Cache(cfg *Lv3Config) CacheLv3Def{
	return &lv3FuncDef{cfg: cfg}
}

//查询数据
func (ca *lv3FuncDef) Get(key string) (string, error)  {
	return ca.cfg.getFun(key, ca.cfg.cfg...)
}*/