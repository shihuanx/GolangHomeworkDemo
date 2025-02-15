package service

import (
	"errors"
	"fmt"
	"log"
	"memoryDataBase/dao"
	"memoryDataBase/model"
)

type StudentMdbService struct {
	memoryDBDao *dao.MemoryDBDao
}

func NewStudentMdbService(db *dao.MemoryDBDao) *StudentMdbService {
	return &StudentMdbService{
		memoryDBDao: db,
	}
}

func (smdbs *StudentMdbService) AddStudent(student *model.Student) {
	log.Printf("向内存添加学生：%s", student.ID)
	smdbs.memoryDBDao.Set(student.ID, student, student.Expiration)
}

func (smdbs *StudentMdbService) GetStudent(studentId string) (*model.Student, error) {
	value, exists := smdbs.memoryDBDao.Get(studentId)
	if exists {
		student, ok := value.(*model.Student)
		if !ok {
			return nil, errors.New("类型断言失败")
		}
		log.Printf("从内存中查找学生：%s", studentId)
		log.Printf("%v", student)
		return student, nil
	}
	errMsg := fmt.Sprintf("找不到学号为：%s的学生", studentId)
	log.Printf("从内存中查找学生：%s失败：%v", studentId, errMsg)
	return nil, errors.New(errMsg)
}

func (smdbs *StudentMdbService) UpdateStudent(student *model.Student) error {
	err := smdbs.StudentExists(student.ID)
	if err != nil {
		return err
	}
	s, _ := smdbs.GetStudent(student.ID)
	for k, v := range student.Grades {
		s.Grades[k] = v
	}
	student.Grades = s.Grades
	smdbs.memoryDBDao.Update(student.ID, student)
	return nil
}

func (smdbs *StudentMdbService) DeleteStudent(studentId string) error {
	err := smdbs.StudentExists(studentId)
	if err != nil {
		log.Printf("从内存删除学生：%s失败：%v", studentId, err)
		return err
	}
	smdbs.memoryDBDao.Delete(studentId)
	log.Printf("从内存删除学生：%s", studentId)
	return nil
}

func (smdbs *StudentMdbService) StudentExists(studentId string) error {
	_, err := smdbs.GetStudent(studentId)
	return err
}

func (smdbs *StudentMdbService) PeriodicDelete() {
	smdbs.memoryDBDao.PeriodicDelete()
}
