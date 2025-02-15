package dao

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"memoryDataBase/model"
)

type StudentMysqlDao struct {
	DB *gorm.DB
}

func NewStudentMysqlDao(db *gorm.DB) *StudentMysqlDao {
	return &StudentMysqlDao{
		DB: db,
	}
}

func (d *StudentMysqlDao) GetStudent(id string) (*model.StudentDB, error) {
	var studentDB model.StudentDB
	result := d.DB.Raw("select * from student where id = ?", id).Scan(&studentDB)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New(fmt.Sprintf("数据库不存在学生：%s", id))
	}
	return &studentDB, nil
}

func (d *StudentMysqlDao) AddStudentToMysql(tx *gorm.DB, student *model.Student) error {
	err := tx.Exec("insert into student (id,name,gender,class,expiration) values (?,?,?,?,?)",
		student.ID, student.Name, student.Gender, student.Class, student.Expiration).Error
	return err
}

func (d *StudentMysqlDao) AddGradeToMysql(tx *gorm.DB, subject string, score float64, id string) error {
	err := tx.Exec("insert into grade (subject, score, student_id) VALUES (?,?,?)",
		subject, score, id).Error
	return err
}

func (d *StudentMysqlDao) GetGrade(studentId string) ([]model.Grade, error) {
	var grades []model.Grade
	err := d.DB.Raw("select * from grade where student_id = ?", studentId).Scan(&grades).Error
	return grades, err
}

func (d *StudentMysqlDao) UpdateStudent(tx *gorm.DB, student *model.Student) error {
	sqlStmt := `
        UPDATE student
        SET
            name = IF(COALESCE(?, '') != '', ?, name),
            gender = IF(COALESCE(?, '') != '', ?, gender),
            class = IF(COALESCE(?, '') != '', ?, class)
        WHERE id = ?
    `

	err := tx.Exec(sqlStmt,
		student.Name, student.Name,
		student.Gender, student.Gender,
		student.Class, student.Class,
		student.ID).Error
	return err
}

func (d *StudentMysqlDao) UpdateGrade(tx *gorm.DB, subject string, score float64, studentId string) error {
	err := tx.Exec("update grade set score=? where id=? and subject=?", score, studentId, subject).Error
	return err
}

func (d *StudentMysqlDao) DeleteStudent(tx *gorm.DB, id string) error {
	err := tx.Exec("delete from student where id = ?", id).Error
	return err
}

func (d *StudentMysqlDao) DeleteScore(tx *gorm.DB, id string) error {
	err := tx.Exec("delete from grade where student_id = ?", id).Error
	return err
}

func (d *StudentMysqlDao) GetGradeBySubject(id string, subject string) (*model.Grade, error) {
	var grade *model.Grade
	err := d.DB.Raw("select * from grade where subject = ? and student_id = ?", subject, id).Scan(&grade).Error
	return grade, err
}

func (d *StudentMysqlDao) GetAllStudents() ([]model.StudentDB, error) {
	var studentDBs []model.StudentDB
	err := d.DB.Raw("select * from student").Scan(&studentDBs).Error
	if err != nil {
		return nil, err
	}
	return studentDBs, nil
}

func (d *StudentMysqlDao) GetStudentCount(id string) (*model.StudentCount, error) {
	var count model.StudentCount
	result := d.DB.Raw("select * from student_count where student_id = ?", id).Scan(&count)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New(fmt.Sprintf("不存在学生记录：%s", id))
	}
	return &count, nil
}

func (d *StudentMysqlDao) AddStudentCount(id string) error {
	err := d.DB.Exec("insert into student_count (student_id) values (?)", id).Error
	return err
}

func (d *StudentMysqlDao) UpdateStudentCount(record *model.StudentCount) error {
	err := d.DB.Exec("update student_count set count=? where student_id = ?", record.Count, record.StudentId).Error
	return err
}

func (d *StudentMysqlDao) DeleteStudentCount(id string) error {
	err := d.DB.Exec("delete from student_count where student_id = ?", id).Error
	return err
}

func (d *StudentMysqlDao) GetHotStudentCounts() ([]*model.StudentCount, error) {
	var counts []*model.StudentCount
	err := d.DB.Raw("select * from student_count order by count desc limit 10").Scan(&counts).Error
	return counts, err
}
