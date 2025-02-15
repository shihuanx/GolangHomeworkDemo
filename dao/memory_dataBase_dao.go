package dao

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

var Expiration = time.Hour

// MemoryDBDao 内存数据库 DAO 结构体
type MemoryDBDao struct {
	dataMap map[string]interface{}
	expires map[string]time.Time
	rwLock  sync.RWMutex
}

// NewMemoryDBDao 创建一个新的内存数据库实例
func NewMemoryDBDao() *MemoryDBDao {
	mdb := &MemoryDBDao{
		dataMap: make(map[string]interface{}),
		expires: make(map[string]time.Time),
	}
	return mdb
}

// Set 设置键值对并设置过期时间
func (mdb *MemoryDBDao) Set(key string, value interface{}, expiration int64) {
	nanoseconds := expiration * int64(time.Second)
	duration := time.Duration(nanoseconds)
	mdb.rwLock.Lock()
	defer mdb.rwLock.Unlock()
	//如果过期时间大于0 就设置过期时间 如果过期时间为0说明这个键永不过期
	if expiration > 0 {
		mdb.expires[key] = time.Now().Add(duration)
		mdb.dataMap[key] = value
		log.Printf("已添加键：%s 值：%v 过期时间：%v", key, value, mdb.expires[key])
	} else {
		mdb.dataMap[key] = value
		log.Printf("已添加键：%s 值：%v", key, value)
	}
}

// Get 获取键对应的值
func (mdb *MemoryDBDao) Get(key string) (interface{}, bool) {
	mdb.rwLock.RLock()
	defer mdb.rwLock.RUnlock()
	expire, exists := mdb.expires[key]
	if exists {
		if time.Now().After(expire) {
			mdb.deleteKey(key)
			log.Printf("键：%s在：%v时已经过期：", key, expire)
			return nil, false
		}
		mdb.expires[key] = time.Now().Add(Expiration)
		log.Printf("已延长键：%s过期时间至：%v", key, mdb.expires[key])
		return mdb.dataMap[key], true
	}
	value, exists := mdb.dataMap[key]
	return value, exists
}

// Update 更新键对应的值
func (mdb *MemoryDBDao) Update(key string, value interface{}) bool {
	mdb.rwLock.Lock()
	defer mdb.rwLock.Unlock()
	expire, exists := mdb.expires[key]
	if exists {
		if time.Now().After(expire) {
			mdb.deleteKey(key)
			log.Printf("键：%s在：%v时已经过期：", key, expire)
			return false
		}
		mdb.expires[key] = time.Now().Add(Expiration)
		log.Printf("已延长键：%s过期时间至：%v", key, mdb.expires[key])
		mdb.dataMap[key] = value
		log.Printf("修改键：%s的值为：%v", key, mdb.dataMap[key])
		return true
	}
	if _, exists = mdb.dataMap[key]; exists {
		mdb.dataMap[key] = value
		log.Printf("修改键：%s的值为：%v", key, mdb.dataMap[key])
		return true
	}
	log.Printf("不存在键：%s", key)
	return false
}

// Delete 删除指定键
func (mdb *MemoryDBDao) Delete(key string) {
	mdb.rwLock.Lock()
	defer mdb.rwLock.Unlock()
	mdb.deleteKey(key)
	log.Printf("删除键: %s", key)
}

// Count 获取数据库中键值对的数量
func (mdb *MemoryDBDao) Count() int {
	mdb.rwLock.RLock()
	defer mdb.rwLock.RUnlock()
	return len(mdb.dataMap)
}

// deleteKey 删除数据和过期时间
func (mdb *MemoryDBDao) deleteKey(key string) {
	delete(mdb.dataMap, key)
	delete(mdb.expires, key)
}

// PeriodicDelete 定期删除过期键
func (mdb *MemoryDBDao) PeriodicDelete() {
	mdb.rwLock.Lock()
	keys := make([]string, 0, len(mdb.expires))
	for key := range mdb.expires {
		keys = append(keys, key)
	}
	// 随机选择一定数量的键进行检查
	if len(keys) > 0 {
		sampleSize := 10 // 每次检查 10 个键
		if len(keys) < sampleSize {
			sampleSize = len(keys)
		}
		rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
		for _, key := range keys[:sampleSize] {
			if expire, exists := mdb.expires[key]; exists && time.Now().After(expire) {
				mdb.deleteKey(key)
				log.Printf("定期删除过期键：%s", key)
			}
		}
	}
	mdb.rwLock.Unlock()
}
