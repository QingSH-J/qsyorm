package qsysession

import (
	"reflect"
)

// Hook 定义了数据库操作前后的钩子接口
type Hook interface {
	// 查询相关钩子
	BeforeQuery() error
	AfterQuery() error

	// 插入相关钩子
	BeforeInsert() error
	AfterInsert() error

	// 更新相关钩子
	BeforeUpdate() error
	AfterUpdate() error

	// 删除相关钩子
	BeforeDelete() error
	AfterDelete() error
}

// 调用对象的 BeforeQuery 方法（如果实现了 Hook 接口）
func (s *Session) CallMethod(value interface{}, method string) error {
	// 获取对象的反射值
	fm := reflect.ValueOf(value).MethodByName(method)
	if !fm.IsValid() {
		return nil
	}

	// 调用该方法
	values := fm.Call([]reflect.Value{})
	if len(values) > 0 {
		if err, ok := values[0].Interface().(error); ok && err != nil {
			return err
		}
	}

	return nil
}

// 钩子辅助方法

// 调用 BeforeQuery 钩子
func (s *Session) CallBeforeQuery(value interface{}) error {
	return s.CallMethod(value, "BeforeQuery")
}

// 调用 AfterQuery 钩子
func (s *Session) CallAfterQuery(value interface{}) error {
	return s.CallMethod(value, "AfterQuery")
}

// 调用 BeforeInsert 钩子
func (s *Session) CallBeforeInsert(value interface{}) error {
	return s.CallMethod(value, "BeforeInsert")
}

// 调用 AfterInsert 钩子
func (s *Session) CallAfterInsert(value interface{}) error {
	return s.CallMethod(value, "AfterInsert")
}

// 调用 BeforeUpdate 钩子
func (s *Session) CallBeforeUpdate(value interface{}) error {
	return s.CallMethod(value, "BeforeUpdate")
}

// 调用 AfterUpdate 钩子
func (s *Session) CallAfterUpdate(value interface{}) error {
	return s.CallMethod(value, "AfterUpdate")
}

// 调用 BeforeDelete 钩子
func (s *Session) CallBeforeDelete(value interface{}) error {
	return s.CallMethod(value, "BeforeDelete")
}

// 调用 AfterDelete 钩子
func (s *Session) CallAfterDelete(value interface{}) error {
	return s.CallMethod(value, "AfterDelete")
}
