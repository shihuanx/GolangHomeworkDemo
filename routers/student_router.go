package routers

import (
	"github.com/gin-gonic/gin"
	"memoryDataBase/controller"
)

func SetUpStudentRouter(studentController *controller.StudentController) *gin.Engine {
	r := gin.Default()
	studentGroup := r.Group("/student")

	studentGroup.POST("", studentController.AddStudent)
	studentGroup.GET("/:id", studentController.GetStudent)
	studentGroup.PUT("", studentController.UpdateStudent)
	studentGroup.DELETE("/:id", studentController.DeleteStudent)

	return r

}
