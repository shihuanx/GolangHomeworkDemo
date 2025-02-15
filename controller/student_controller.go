package controller

import (
	"github.com/gin-gonic/gin"
	"log"
	"memoryDataBase/model"
	"memoryDataBase/response"
	"memoryDataBase/service"
	"net/http"
)

type StudentController struct {
	studentService *service.StudentService
}

func NewStudentController(studentService *service.StudentService) *StudentController {
	return &StudentController{
		studentService: studentService,
	}
}

func (sc *StudentController) AddStudent(c *gin.Context) {
	var student model.Student
	if err := c.BindJSON(&student); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
	} else if err = sc.studentService.AddStudentInternal(&student); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
	} else {
		log.Printf("添加学号为：%s的学生", student.ID)
		c.JSON(http.StatusOK, response.SuccessWithoutData())
	}
}

func (sc *StudentController) GetStudent(c *gin.Context) {
	studentId := c.Param("id")
	resp, err := sc.studentService.GetStudent(studentId)
	if err != nil {
		c.JSON(500, response.Error(err.Error()))
	} else {
		log.Printf("查询学号为：%s的学生", studentId)
		c.JSON(http.StatusOK, response.Success(resp))
	}
}

func (sc *StudentController) UpdateStudent(c *gin.Context) {
	var student model.Student
	if err := c.BindJSON(&student); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
		return
	}
	err := sc.studentService.UpdateStudentInternal(&student)
	if err != nil {
		c.JSON(http.StatusNotFound, response.Error(err.Error()))
	} else {
		log.Printf("修改学生：%s", student.ID)
		c.JSON(http.StatusOK, response.Success(nil))
	}
}

func (sc *StudentController) DeleteStudent(c *gin.Context) {
	studentId := c.Param("id")
	err := sc.studentService.DeleteStudentInternal(studentId)
	if err != nil {
		c.JSON(http.StatusNotFound, response.Error(err.Error()))
	} else {
		log.Printf("删除学号为：%s的学生", studentId)
		c.JSON(http.StatusOK, response.Success(nil))
	}
}
