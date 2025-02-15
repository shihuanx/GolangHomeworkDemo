package service

import (
	"gorm.io/gorm"
	"log"
	"memoryDataBase/dao"
	"memoryDataBase/model"
	"strings"
)

type StudentMysqlService struct {
	mysqlDao *dao.StudentMysqlDao
}

func NewStudentMysqlService(mysqlDao *dao.StudentMysqlDao) *StudentMysqlService {
	return &StudentMysqlService{
		mysqlDao: mysqlDao,
	}
}

func (sms *StudentMysqlService) ConvertToStudent(studentDB *model.StudentDB) (*model.Student, error) {
	grades := make(map[string]float64)
	result, err := sms.mysqlDao.GetGrade(studentDB.ID)
	if err != nil {
		return nil, err
	}

	for _, v := range result {
		grades[v.Subject] = v.Score
	}
	return &model.Student{
		ID:     studentDB.ID,
		Name:   studentDB.Name,
		Gender: studentDB.Gender,
		Class:  studentDB.Class,
		Grades: grades,
	}, nil
}

func (sms *StudentMysqlService) StudentExists(id string) error {
	_, err := sms.mysqlDao.GetStudent(id)
	return err
}

func (sms *StudentMysqlService) StudentCountExists(id string) error {
	_, err := sms.mysqlDao.GetStudentCount(id)
	return err
}

func (sms *StudentMysqlService) AddStudentToMysql(tx *gorm.DB, student *model.Student) error {
	if err := sms.mysqlDao.AddStudentToMysql(tx, student); err != nil {
		tx.Rollback()
		log.Printf("向学生表添加学生：%s失败：%v", student.ID, err)
		return err
	}
	log.Printf("向数据库添加学生：%s", student.ID)

	// 在事务中添加学生成绩信息
	for k, v := range student.Grades {
		if err := sms.mysqlDao.AddGradeToMysql(tx, k, v, student.ID); err != nil {
			tx.Rollback()
			log.Printf("向成绩表添加学生：%s的成绩失败：%v", student.ID, err)
			return err
		}
	}
	log.Printf("向数据库添加学生的成绩：%s", student.ID)
	return nil
}

func (sms *StudentMysqlService) GetStudentFromMysql(studentId string) (*model.Student, error) {
	var studentDB *model.StudentDB
	var student *model.Student
	studentDB, err := sms.mysqlDao.GetStudent(studentId)
	if err != nil {
		log.Printf("从数据库查找学生：%s失败：%v", studentId, err)
		return nil, err
	}
	log.Printf("从数据库查找学生：%s", studentId)
	student, err = sms.ConvertToStudent(studentDB)
	if err != nil {
		log.Printf("数据库中学生：%s转化出错：%v", studentId, err)
		return nil, err
	}
	log.Printf("数据库中学生：%s转化成功", student.ID)
	return student, nil
}

func (sms *StudentMysqlService) UpdateStudent(tx *gorm.DB, student *model.Student) error {
	err := sms.StudentExists(student.ID)
	if err != nil {
		log.Printf("数据库不存在学生：%s", student.ID)
		tx.Rollback()
		return err
	}

	if err = sms.mysqlDao.UpdateStudent(tx, student); err != nil {
		tx.Rollback()
		log.Printf("在数据库更新学生：%s失败：%v", student.ID, err)
		return err
	}
	log.Printf("在数据库更新学生：%s", student.ID)
	//向成绩表插入数据 先判断是否存在 如果存在就更新 不存在就添加
	if student.Grades != nil {
		for subject, grade := range student.Grades {
			exists, err := sms.mysqlDao.GetGradeBySubject(student.ID, subject)
			if err != nil {
				tx.Rollback()
				log.Printf("通过学科：%s查找学生：%s的记录失败：%v", student.ID, subject, err)
				return err
			}
			if exists != nil {
				//成绩记录已存在 修改
				if err = sms.mysqlDao.UpdateGrade(tx, subject, grade, student.ID); err != nil {
					tx.Rollback()
					log.Printf("向成绩表添加成绩失败，学生：%s，错误：%v", student.ID, err)
					return err
				}
			} else {
				//成绩记录不存在 插入
				if err = sms.mysqlDao.AddGradeToMysql(tx, subject, grade, student.ID); err != nil {
					tx.Rollback()
					log.Printf("向成绩表添加学生：%s的成绩：%s失败：%v", student.ID, subject, err)
					return err
				}
			}
		}
	}
	log.Printf("在数据库更新学生：%s的成绩", student.ID)
	return nil
}
func (sms *StudentMysqlService) DeleteStudent(tx *gorm.DB, id string) error {
	if err := sms.StudentExists(id); err != nil {
		log.Printf("数据库不存在学生：%s", id)
		tx.Rollback()
		return err
	}

	if err := sms.mysqlDao.DeleteStudent(tx, id); err != nil {
		tx.Rollback()
		log.Printf("删除学生：%s失败：%v", id, err)
		return err
	}
	log.Printf("删除学生：%s", id)

	if err := sms.mysqlDao.DeleteScore(tx, id); err != nil {
		tx.Rollback()
		log.Printf("删除学生：%s的成绩时失败：%v", id, err)
		return err
	}
	log.Printf("删除学生：%s的成绩", id)
	return nil
}

func (sms *StudentMysqlService) GetHotStudentsFromMysql() ([]*model.Student, error) {
	var hotStudents []*model.StudentCount
	var students []*model.Student
	hotStudents, err := sms.GetHotStudentCount()
	if err != nil {
		return nil, err
	}
	for _, studentRecord := range hotStudents {
		student, err := sms.GetStudentFromMysql(studentRecord.StudentId)
		if err != nil {
			log.Printf("从数据库转化学生：%s失败：%v", studentRecord.StudentId, err)
			return nil, err
		}
		students = append(students, student)
	}
	return students, nil
}

func (sms *StudentMysqlService) AddStudentCount(id string) {
	record, err := sms.GetStudentCountFromMysql(id)
	if err != nil {
		if strings.Contains(err.Error(), "不存在学生记录") {
			if err = sms.mysqlDao.AddStudentCount(id); err != nil {
				log.Printf("添加学生：%s的记录时出错：%v", id, err)
			} else {
				log.Printf("添加学生：%s记录", id)
			}
		} else {
			log.Printf("添加学生记录：%s时失败：%v", id, err)
		}
	} else {
		record.Count++
		if err = sms.mysqlDao.UpdateStudentCount(record); err != nil {
			log.Printf("更新学生：%s访问次数时出错：%v", id, err)
		} else {
			log.Printf("更新学生：%s的访问次数：%s", id, record.StudentId)
		}
	}
}

func (sms *StudentMysqlService) GetStudentCountFromMysql(id string) (*model.StudentCount, error) {
	return sms.mysqlDao.GetStudentCount(id)
}

func (sms *StudentMysqlService) GetHotStudentCount() ([]*model.StudentCount, error) {
	studentCounts, err := sms.mysqlDao.GetHotStudentCounts()
	if err != nil {
		log.Printf("获取访问最高的学生记录出错：%v", err)
		return nil, err
	}
	return studentCounts, nil
}

func (sms *StudentMysqlService) DeleteStudentCount(id string) {
	if err := sms.StudentCountExists(id); err != nil {
		log.Printf("学生记录不存在：%s", id)
	}
	if err := sms.mysqlDao.DeleteStudentCount(id); err != nil {
		log.Printf("删除学生：%s记录时失败：%v", id, err)
	}
}
