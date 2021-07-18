package main

// 后端返回数据结构
// 包含 data 与 type
type backendMsg struct {
	Type string   `json:"type"`
	Data []string `json:"data"`
}

// 前端传递数据结构
type msgInfo struct {
	Type     string `json:"type"`
	Row      uint64 `json:"row"`
	Column   uint64 `json:"column"`
	NewValue string `json:"newValue"`
	FileName string `json:"fileName"`
	UserName string `json:"userName"`
	// OldValue string `json:"oldValue"`
}

// 前端传递数据结构
type msgOldInfo struct {
	Type     string `json:"type"`
	Row      uint64 `json:"row"`
	Column   uint64 `json:"column"`
	NewValue string `json:"newValue"`
	FileName string `json:"fileName"`
	UserName string `json:"userName"`
	OldValue string `json:"oldValue"`
}

// 拿锁错误数据结构
type lockErrMsg struct {
	Type            string `json:"type"`
	Row             uint64 `json:"row"`
	Column          uint64 `json:"column"`
	SuccessUsername string `json:"successUsername"`
	RejectUsername  string `json:"rejectUsername"`
}

// 文件锁信息数据结构
type fileLockMsg struct {
	Filename string `json:"filename"`
	Owner    string `json:"owner"`
	Row      uint64 `json:"row"`
	Column   uint64 `json:"column"`
}

// 文件锁信息返回数据结构
type fileLockRetMsg struct {
	Type string        `json:"type"`
	Data []fileLockMsg `json:"data"`
}

type rc struct {
	Row uint64
	Col uint64
}
