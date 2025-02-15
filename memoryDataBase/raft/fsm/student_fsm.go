package fsm

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/raft"
	"io"
	"memoryDataBase/interfaces"
	"memoryDataBase/model"
)

// StudentCommand 定义 Raft 日志条目的结构
type StudentCommand struct {
	Operation string         `json:"operation"`
	Student   *model.Student `json:"student,omitempty"`
	Id        string         `json:"id"`
}

// StudentFSM 实现 raft.FSM 接口
type StudentFSM struct {
	service interfaces.StudentServiceInterface
}

// NewStudentFSM 创建一个新的 StudentFSM 实例
func NewStudentFSM(service interfaces.StudentServiceInterface) *StudentFSM {
	return &StudentFSM{
		service: service,
	}
}

// Apply 应用日志条目到状态机
func (fsm *StudentFSM) Apply(log *raft.Log) interface{} {
	var cmd StudentCommand
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return err
	}
	switch cmd.Operation {
	case "add":
		return fsm.service.AddStudentInternal(cmd.Student)
	case "update":
		return fsm.service.UpdateStudentInternal(cmd.Student)
	case "delete":
		return fsm.service.DeleteStudentInternal(cmd.Id)
	case "reloadCacheData":
		fsm.service.ReloadCacheDataInternal()
		return nil
	case "periodicDelete":
		fsm.service.PeriodicDeleteInternal()
		return nil
	default:
		return fmt.Errorf("unknown operation: %s", cmd.Operation)
	}
}

// Snapshot 实现快照功能 还没写
func (fsm *StudentFSM) Snapshot() (raft.FSMSnapshot, error) {
	return nil, nil
}

// Restore 恢复状态机到快照状态 还没写
func (fsm *StudentFSM) Restore(snapshot io.ReadCloser) error {
	defer snapshot.Close()
	fmt.Printf("Restoring snapshot data\n")
	return nil
}
