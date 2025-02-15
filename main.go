package main

import (
	"log"
	"memoryDataBase/cache"
	"memoryDataBase/controller"
	"memoryDataBase/dao"
	"memoryDataBase/database"
	"memoryDataBase/routers"
	"memoryDataBase/service"
	"time"
)

func main() {
	// 初始化数据库和缓存
	dsn := "root:1234@tcp(127.0.0.1:3306)/mdb?charset=utf8mb4&parseTime=True&loc=Local"
	err := database.InitDB(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize mysqlDataBase: %v", err)
	}
	cache.InitRedis("192.168.88.128:6379", "123456", 0)

	// 初始化 DAO
	studentCacheDao := dao.NewStudentCacheDao(cache.RedisClient)
	studentMysqlDao := dao.NewStudentMysqlDao(database.DB)
	memoryDBDao := dao.NewMemoryDBDao()

	// 初始化服务
	studentCacheService := service.NewStudentCacheService(studentCacheDao)
	studentMysqlService := service.NewStudentMysqlService(studentMysqlDao)
	studentMdbService := service.NewStudentMdbService(memoryDBDao)
	studentService, err := service.NewStudentService(studentMdbService, studentMysqlService, studentCacheService, "127.0.0.1")
	if err != nil {
		log.Fatalf("初始化学生服务层失败：%v", err)
	}

	// 初始化控制器
	studentController := controller.NewStudentController(studentService)

	//启动时加载缓存数据到内存
	if err = studentService.LoadCacheToMemory(); err != nil {
		log.Printf("加载缓存到内存时失败")
		if err = studentService.LoadDateBaseToMemory(); err != nil {
			log.Printf("加载数据库中的数据到内存时失败")
		}
	}

	go func() {
		studentService.ReloadCacheData(time.Hour)
	}()

	go func() {
		studentService.PeriodicDelete(time.Hour)
	}()

	r := routers.SetUpStudentRouter(studentController)
	r.Run(":8080")
}
