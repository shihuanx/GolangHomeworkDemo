package service

import (
	"encoding/json"
	"fmt"
	raftfpk "github.com/hashicorp/raft"
	"log"
	"memoryDataBase/interfaces"
	"memoryDataBase/model"
	"memoryDataBase/raft"
	"memoryDataBase/raft/fsm"
	"strings"
	"time"
)

type StudentService struct {
	MdbService   *StudentMdbService
	MysqlService *StudentMysqlService
	CacheService *StudentCacheService
	raftNode     *raftfpk.Raft
}

func NewStudentService(mdbService *StudentMdbService, mysqlService *StudentMysqlService, cacheService *StudentCacheService, localID string) (*StudentService, error) {
	ss := &StudentService{
		MdbService:   mdbService,
		MysqlService: mysqlService,
		CacheService: cacheService,
	}

	initializer := &raft.RaftInitializerImpl{}
	// 初始化 Raft 节点
	raftNode, err := initializer.InitRaft(localID, ss)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Raft node: %w", err)
	}
	ss.raftNode = raftNode

	return ss, nil
}

// 确保实现 StudentServiceInterface 接口
var _ interfaces.StudentServiceInterface = (*StudentService)(nil)

// StudentNotFoundErr 判断错误是不是没有找到学生之类的错误 如果是那就继续去下一个数据源找 不返回
func (ss *StudentService) StudentNotFoundErr(studentId string, err error) bool {
	studentNotFoundErrMsg := fmt.Sprintf("找不到学号为：%s的学生", studentId)
	return strings.Contains(err.Error(), studentNotFoundErrMsg)
}

func (ss *StudentService) applyRaftCommand(operation string, student *model.Student, id string) error {
	// 创建 Raft 命令
	cmd := fsm.StudentCommand{
		Operation: operation,
		Student:   student,
		Id:        id,
	}
	// 序列化命令
	cmdData, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal add student command: %w", err)
	}
	// 提交命令到 Raft 节点
	future := ss.raftNode.Apply(cmdData, 500)
	if err = future.Error(); err != nil {
		return fmt.Errorf("failed to apply add student command to Raft: %w", err)
	}
	// 处理响应
	result := future.Response()
	if resultErr, ok := result.(error); ok {
		return resultErr
	}
	return nil
}

// RestoreCacheData 恢复缓存机制 mysql有事务可以很方便地回滚 此函数专门用于恢复缓存的数据
func (ss *StudentService) RestoreCacheData(id string) error {
	//如果要恢复数据 mysql的事务会回滚 所以这个时候找到的学生还是一开始的学生
	studentBackUp, err := ss.MysqlService.GetStudentFromMysql(id)
	if err != nil {
		log.Printf("尝试通过学生id：%s获取学生时失败：%v", id, err)
		return err
	}
	if err = ss.CacheService.AddStudent(studentBackUp); err != nil {
		log.Printf("尝试恢复缓存数据时失败: %v", err)
		return err
	}
	return nil
}

func (ss *StudentService) ReloadCacheDataInternal() {
	students, err := ss.MysqlService.GetHotStudentsFromMysql()
	if err != nil {
		log.Printf("获得访问最多的学生时出错：%v", err)
	}
	err = ss.CacheService.ReLoadCacheData(students)
	if err != nil {
		log.Printf("重新加载缓存失败：%v", err)
	}
	log.Printf("已重新加载缓存: %v", time.Now())
}

func (ss *StudentService) PeriodicDeleteInternal() {
	log.Printf("定期删除内存中的过期键：%v", time.Now())
	ss.MdbService.PeriodicDelete()
}

func (ss *StudentService) LoadCacheToMemory() error {
	students, err := ss.CacheService.GetAllStudentsFromCache()
	if err != nil {
		log.Printf("从缓存中获取所有学生时失败：%v", err)
		return err
	}
	for _, student := range students {
		ss.MdbService.AddStudent(student)
	}
	log.Printf("从缓存加载到内存")
	return nil
}

func (ss *StudentService) LoadDateBaseToMemory() error {
	students, err := ss.MysqlService.GetHotStudentsFromMysql()
	if err != nil {
		log.Printf("从数据库中获取热门学生时失败：%v", err)
		return err
	}
	for _, student := range students {
		ss.MdbService.AddStudent(student)
	}
	log.Printf("从数据库中加载到内存")
	return nil
}

func (ss *StudentService) AddStudentInternal(student *model.Student) error {
	// 开始 MySQL 事务
	tx := ss.MysqlService.mysqlDao.DB.Begin()
	if tx.Error != nil {
		log.Printf("开启 MySQL 事务失败：%v", tx.Error)
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			// 发生 panic 时回滚事务
			tx.Rollback()
			log.Printf("事务已回滚：%v", r)
		}
	}()

	// 在 MySQL 数据库事务中添加学生信息
	if err := ss.MysqlService.AddStudentToMysql(tx, student); err != nil {
		return err
	}

	// MySQL 数据库事务提交成功后，尝试添加到缓存
	if err := ss.CacheService.AddStudent(student); err != nil {
		tx.Rollback()
		log.Printf("事务已回滚")
		return err
	}
	// 最后添加到内存数据库
	ss.MdbService.AddStudent(student)
	if err := tx.Commit().Error; err != nil {
		log.Printf("提交事务失败：%v", err)
		return err
	}
	ss.MysqlService.AddStudentCount(student.ID)
	return nil
}

func (ss *StudentService) GetStudent(id string) (*model.Student, error) {
	// 先从内存中查找学生
	student, memoryErr := ss.MdbService.GetStudent(id)
	if student != nil {
		ss.MysqlService.AddStudentCount(id)
		log.Printf("从内存中查找到了学生：%s", id)
		return student, nil
	}

	//再从缓存中查找学生
	student, cacheErr := ss.CacheService.GetStudentFromCache(id)
	if student != nil {
		ss.MysqlService.AddStudentCount(id)
		log.Printf("从缓存中查找到了学生：%s", id)
		//向内存中添加学生
		if ss.StudentNotFoundErr(id, memoryErr) {
			ss.MdbService.AddStudent(student)
			log.Printf("从缓存向内存中添加学生：%s", id)
		}
		return student, nil
	}

	// 最后从数据库中查找学生
	student, mysqlErr := ss.MysqlService.GetStudentFromMysql(id)
	if mysqlErr != nil {
		return nil, mysqlErr
	}
	if student != nil {
		ss.MysqlService.AddStudentCount(id)
		log.Printf("在数据库中查找到了学生：%s", id)
		//向内存和缓存中添加学生
		if ss.StudentNotFoundErr(id, memoryErr) {
			ss.MdbService.AddStudent(student)
			log.Printf("从数据库向内存中添加学生：%s", id)
		}
		if ss.StudentNotFoundErr(id, cacheErr) {
			err := ss.CacheService.AddStudent(student)
			if err != nil {
				log.Printf("从数据库向缓存中添加学生：%s失败：%v", id, err)
			} else {
				log.Printf("从数据库向缓存中添加学生：%s", id)
			}
		}
		return student, nil
	}
	//其实这是不可到达的代码 因为不存在既没有错误又没有学生的情况。。没有学生也是一个错误
	return nil, nil
}

func (ss *StudentService) UpdateStudentInternal(student *model.Student) error {
	// 开始 MySQL 事务
	tx := ss.MysqlService.mysqlDao.DB.Begin()
	if tx.Error != nil {
		log.Printf("开启 MySQL 事务失败：%v", tx.Error)
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			// 发生 panic 时回滚事务
			tx.Rollback()
			log.Printf("事务已回滚：%v", r)
		}
	}()

	if err := ss.MysqlService.UpdateStudent(tx, student); err != nil {
		log.Printf("更新学生：%s时失败：%v", student.ID, err)
		return err
	}
	if err := ss.CacheService.UpdateStudent(student); err != nil {
		if !ss.StudentNotFoundErr(student.ID, err) {
			log.Printf("已回滚数据库事务")
			log.Printf("更新缓存中的学生：%s时失败：%v", student.ID, err)
			tx.Rollback()
			return err
		}
	}
	if err := ss.MdbService.UpdateStudent(student); err != nil {
		if !ss.StudentNotFoundErr(student.ID, err) {
			log.Printf("更新内存中的学生：%s时失败：%v", student.ID, err)
			tx.Rollback()
			if err = ss.RestoreCacheData(student.ID); err != nil {
				log.Printf("尝试恢复缓存数据时失败：%v", err)
				return err
			}
			log.Printf("回滚数据库事务并恢复缓存数据")
			return err
		}
	}
	if err := tx.Commit().Error; err != nil {
		log.Printf("回滚事务失败：%v", err)
		return err
	}
	ss.MysqlService.AddStudentCount(student.ID)
	return nil
}

func (ss *StudentService) DeleteStudentInternal(id string) error {
	// 开始 MySQL 事务
	tx := ss.MysqlService.mysqlDao.DB.Begin()
	if tx.Error != nil {
		log.Printf("开启 MySQL 事务失败：%v", tx.Error)
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			// 发生 panic 时回滚事务
			tx.Rollback()
			log.Printf("事务已回滚：%v", r)
		}
	}()

	if err := ss.CacheService.DeleteStudent(id); err != nil {
		if !ss.StudentNotFoundErr(id, err) {
			tx.Rollback()
			log.Printf("已回滚数据库事务")
			log.Printf("从缓存中删除学生：%s失败：%v", id, err)
			return err
		}
		log.Printf("缓存中不存在学生：%s", id)
	}
	if err := ss.MdbService.DeleteStudent(id); err != nil {
		if !ss.StudentNotFoundErr(id, err) {
			tx.Rollback()
			log.Printf("从内存中删除学生：%s失败：%v", id, err)
			if err = ss.RestoreCacheData(id); err != nil {
				log.Printf("尝试恢复缓存数据时失败：%v", err)
				return err
			}
			log.Printf("回滚数据库事务并恢复缓存")
		}
		log.Printf("内存中不存在学生：%s", id)
	}
	if err := tx.Commit().Error; err != nil {
		log.Printf("回滚事务失败：%v", err)
		return err
	}
	ss.MysqlService.DeleteStudentCount(id)
	return nil
}

func (ss *StudentService) ReloadCacheData(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := ss.applyRaftCommand("reloadCacheData", nil, "")
			if err != nil {
				log.Printf("分布式加载缓存数据失败: %v，跳过这次操作：%v", err, time.Now())
				continue
			}
		}
	}
}

func (ss *StudentService) PeriodicDelete(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := ss.applyRaftCommand("periodicDelete", nil, "")
			if err != nil {
				log.Printf("分布式删除内存数据库过期键失败：: %v，跳过这次操作：%v", err, time.Now())
				continue
			}
		}
	}
}

func (ss *StudentService) AddStudent(student *model.Student) error {
	return ss.applyRaftCommand("add", student, "")
}

func (ss *StudentService) UpdateStudent(student *model.Student) error {
	return ss.applyRaftCommand("update", student, "")
}

func (ss *StudentService) DeleteStudent(id string) error {
	return ss.applyRaftCommand("delete", nil, id)
}
