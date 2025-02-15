package service

import (
	"log"
	"memoryDataBase/dao"
	"memoryDataBase/model"
)

type StudentCacheService struct {
	cacheDao *dao.StudentCacheDao
}

func NewStudentCacheService(cacheDao *dao.StudentCacheDao) *StudentCacheService {
	return &StudentCacheService{
		cacheDao: cacheDao,
	}
}

func (scs *StudentCacheService) StudentExists(id string) error {
	_, err := scs.cacheDao.GetStudent(id)
	return err
}

func (scs *StudentCacheService) AddStudent(student *model.Student) error {
	if err := scs.cacheDao.AddStudent(student); err != nil {
		log.Printf("向缓存添加学生：%s失败：%v", student.ID, err)
		return err
	}
	log.Printf("向缓存添加学生学生：%s", student.ID)
	return nil
}

func (scs *StudentCacheService) GetStudentFromCache(id string) (*model.Student, error) {
	student, err := scs.cacheDao.GetStudent(id)
	if err != nil {
		log.Printf("从缓存查找学生：%s失败：%v", id, err)
		return nil, err
	}
	log.Printf("从缓存查找学生：%s", id)
	return student, nil
}

func (scs *StudentCacheService) UpdateStudent(student *model.Student) error {
	err := scs.StudentExists(student.ID)
	if err != nil {
		log.Printf("缓存中不存在学生：%s", student.ID)
		return err
	}
	s, _ := scs.cacheDao.GetStudent(student.ID)
	for k, v := range student.Grades {
		s.Grades[k] = v
	}
	student.Grades = s.Grades
	if err = scs.cacheDao.AddStudent(student); err != nil {
		log.Printf("向缓存中更新学生：%s失败：%v", student.ID, err)
		return err
	}
	return nil
}

func (scs *StudentCacheService) DeleteStudent(id string) error {
	err := scs.StudentExists(id)
	if err != nil {
		log.Printf("从缓存删除学生：%s失败：%v", id, err)
		return err
	}
	if err = scs.cacheDao.DeleteStudent(id); err != nil {
		log.Printf("从缓存删除学生：%s失败：%v", id, err)
		return err
	}
	log.Printf("从缓存删除学生：%s", id)
	return nil
}

func (scs *StudentCacheService) ReLoadCacheData(students []*model.Student) error {
	return scs.cacheDao.ReLoadCacheData(students)
}

func (scs *StudentCacheService) GetAllStudentsFromCache() ([]*model.Student, error) {
	return scs.cacheDao.GetAllStudents()
}
