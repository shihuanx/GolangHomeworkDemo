package dao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"memoryDataBase/model"
	"strconv"
)

const studentCachePrefix = "student:"

type StudentCacheDao struct {
	client redis.Client
}

func NewStudentCacheDao(client *redis.Client) *StudentCacheDao {
	return &StudentCacheDao{
		client: *client,
	}
}

func (d StudentCacheDao) AddStudent(student *model.Student) error {
	ctx := context.Background()

	key := studentCachePrefix + student.ID

	gradeJSON, err := json.Marshal(student.Grades)
	if err != nil {
		log.Println("将成绩序列化为json时出错")
		return err
	}

	fields := make(map[string]interface{})
	fields["id"] = student.ID
	fields["name"] = student.Name
	fields["gender"] = student.Gender
	fields["class"] = student.Class
	fields["grade"] = gradeJSON
	fields["expiration"] = student.Expiration

	err = d.client.HSet(ctx, key, fields).Err()
	return err
}

func (d StudentCacheDao) GetStudent(id string) (*model.Student, error) {
	ctx := context.Background()
	key := studentCachePrefix + id

	// 从缓存中获取学生的所有字段信息
	result, err := d.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	// 如果结果为空，说明学生信息不存在
	if len(result) == 0 {
		errMsg := fmt.Sprintf("在缓存中查找不到学号为：%s的学生", id)
		return nil, errors.New(errMsg)
	}

	// 创建一个新的 Student 对象
	student := &model.Student{}

	// 填充基本信息
	student.ID = result["id"]
	student.Name = result["name"]
	student.Gender = result["gender"]
	student.Class = result["class"]
	student.Expiration, err = strconv.ParseInt(result["expiration"], 10, 64)

	// 反序列化成绩信息
	gradeJSON := []byte(result["grade"])
	grades := make(map[string]float64)
	err = json.Unmarshal(gradeJSON, &grades)
	if err != nil {
		log.Println("将成绩从json反序列化时出错")
		return nil, err
	}
	student.Grades = grades

	return student, nil
}

func (d StudentCacheDao) DeleteStudent(id string) error {
	ctx := context.Background()
	// 构建缓存键
	key := studentCachePrefix + id

	// 使用 Del 命令删除缓存数据
	err := d.client.Del(ctx, key).Err()
	if err != nil {
		log.Printf("删除学生缓存信息时出错: %v\n", err)
		return err
	}
	return nil
}

func (d StudentCacheDao) ReLoadCacheData(students []*model.Student) error {
	ctx := context.Background()
	// 删除所有 Redis 记录
	err := d.client.FlushDB(ctx).Err()
	if err != nil {
		return fmt.Errorf("删除所有缓存记录时出错: %w", err)
	}
	for _, student := range students {
		err = d.AddStudent(student)
		if err != nil {
			return fmt.Errorf("加载学生时出错：%w", err)
		}
	}
	return nil
}

func (d StudentCacheDao) GetAllStudents() ([]*model.Student, error) {
	ctx := context.Background()
	var students []*model.Student

	keys, err := d.client.Keys(ctx, "student:*").Result()
	if err != nil {
		log.Printf("获取所有学生的键时失败: %v", err)
		return nil, err
	}

	for _, key := range keys {
		studentId, err := d.client.HGet(ctx, key, "id").Result()
		if err != nil {
			log.Printf("通过键获得学生id时失败：%v", err)
			return nil, err
		}
		student, err := d.GetStudent(studentId)
		if err != nil {
			log.Printf("通过学生id：%s获得学生时失败：%v", studentId, err)
		}
		students = append(students, student)
	}
	return students, nil
}
