package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"strconv"
	"strings"
	"time"
)

// --------- handlers begin ----------------

// 将 logData 按行写入 log 文件中
func writeLog(logData string) {
	logLenStr := read("gDocLog", 0, 8)
	logLen, _ := strconv.Atoi(logLenStr)
	newLogLen := logLen + len(logData)
	write("gDocLog", 0, numExtend(newLogLen, 8))
	write("gDocLog", uint64(logLen), logData)
}

// 将文件内容全部读出，发送给连接者
// 返回的格式为所有单元格内容组成的 string 数组
func handleOpen(currFileName string, conn *websocket.Conn, messageType int, userName string) {
	// 读取整个文件
	// 将参与的文件内容发回前端
	currFileSize, ok := fileMap[currFileName]

	// 若打开文件失败，说明文件列表需要被更新
	// 将现在的文件列表重新发回
	if !ok {
		fmt.Println("[OPEN] error: try to open non-exists file", currFileName)
		// 向连接发送错误信息
		var retMsg backendMsg
		retMsg.Type = "OpenError"
		retMsg.Data = traverseFileMap()
		bytes, e := json.Marshal(retMsg)
		if e != nil {
			fmt.Println("[OPEN] Error in marshal")
		} else {
			fmt.Println("[OPEN] Send back: ", string(bytes))
			err := conn.WriteMessage(1, bytes)
			if err != nil {
				fmt.Println("[OPEN] error in send back")
			}
		}
		return
	}

	fmt.Println("[OPEN] Try to open file: ", currFileName)
	backMsg := read(currFileName, 0, currFileSize)

	// 将 backMsg 转化成 string 数组
	backMsgSlice := make([]string, 0)
	checkAlign := len(backMsg) % int(blockSize)
	sliceTime := len(backMsg) / int(blockSize)
	if checkAlign == 0 {
		sliceTime -= 1
	}

	start := 0
	end := int(blockSize)
	for i := 0; i < sliceTime; i++ {
		// 空字符检查，若为空则用""替换
		emptyFlag := getStr(blockSize)
		currMsg := backMsg[start:end]
		if strings.Compare(emptyFlag, currMsg) == 0 {
			// fmt.Println("[OPEN] change the emnpty flag")
			currMsg = ""
		}
		backMsgSlice = append(backMsgSlice, currMsg)
		start += int(blockSize)
		end += int(blockSize)
	}
	backMsgSlice = append(backMsgSlice, backMsg[start:])

	// 若文件不满，则主动添加剩余的string
	currLen := len(backMsgSlice)
	lastLen := int(colNum*rowNum) - currLen
	for i := 0; i < lastLen; i++ {
		backMsgSlice = append(backMsgSlice, " ")
	}

	// fmt.Println("[OPEN] read result:", backMsgSlice)

	// 将返回数据结构改为 backendMsg 形式
	retMsg := backendMsg{"FileDataSlice", backMsgSlice}
	bytes, e := json.Marshal(retMsg)
	if e != nil {
		fmt.Println("Error")
	} else {
		// 1 是 messageType 不一定对
		fmt.Println("[OPEN] Send back the data of file:", currFileName)
		// fmt.Println(bytes)
		conn.WriteMessage(messageType, bytes)
	}

	// if err := conn.WriteMessage(messageType, []byte(backMsg)); err != nil {
	// 	log.Println("OPEN ERROR: fail to send message back!")
	// 	log.Println(err)
	// }

	// 写 log 文件
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	logData := "[OPEN] " + timeStr + " " + userName + " open the file " + currFileName + "\t"
	writeLog(logData)

	// 将是否有其他人正在编辑文件本文件的信息读出并返回
	// 锁结构：/filename-row-col
	fileLockSlice := make([]fileLockMsg, 0)
	for k, _ := range lockMap {
		lockHead := "/" + currFileName
		lockHeadLen := len(lockHead)
		if k[:lockHeadLen] == lockHead {
			lockInfo := k
			sp := strings.Split(lockInfo, "-")
			rowStr := sp[1]
			colStr := sp[2]
			r, _ := strconv.Atoi(rowStr)
			c, _ := strconv.Atoi(colStr)
			owner := lockMap[k]
			currFL := fileLockMsg{currFileName, owner, uint64(r), uint64(c)}
			fileLockSlice = append(fileLockSlice, currFL)
		}
	}
	flRet := fileLockRetMsg{"FILELOCKINFO", fileLockSlice}
	bytes2, _ := json.Marshal(flRet)
	err2 := conn.WriteMessage(1, bytes2)
	if err2 != nil {
		fmt.Println("[OPEN] error in send back file lock message")
	} else {
		fmt.Println("[OPEN] send the file lock message:", string(bytes2))
	}

}

// 创建新文件
func handleCreate(createFileName string, conn *websocket.Conn, userName string) {
	fmt.Println("[CREATE] try to create file: ", createFileName)

	// 首先在 trashMap 中查找文件，若存在同名文件，则认为执行 recover 操作
	_, ok := trashMap[createFileName]
	if ok {
		fmt.Println("[CREATE] the creating file", createFileName, "is already in trash bin")
		recoverFile(createFileName, userName)
		return
	}

	// fileSystem api
	defaultSize := uint64(rowNum * colNum * blockSize)
	createCheck := create(createFileName)
	if createCheck == false {
		fmt.Println("[CREATE] create exists file: ", createFileName)
		var retMsgErr backendMsg
		retMsgErr.Type = "CEF"
		bytes3, _ := json.Marshal(retMsgErr)
		error := conn.WriteMessage(1, bytes3)
		if error != nil {
			fmt.Println("[CREATE] error in send back exisits msg")
		} else {
			fmt.Println("[CREATE]:", string(bytes3))
		}
		return
	}

	// 用空字符初始化新建的文件
	write(createFileName, 0, getStr(defaultSize))

	// 在 fileMap 中添加记录
	fileMap[createFileName] = uint64(defaultSize)

	// 修改 config file 中的文件数
	fileNumStr := read("config", 0, 8)
	fileNum, _ := strconv.Atoi(fileNumStr)
	fileNumStr2 := numExtend(fileNum+1, 8)
	write("config", 0, fileNumStr2)

	// 在 config file 中增加新增文件的记录
	totalLenStr := read("config", 8, 8)
	totalLen, _ := strconv.Atoi(totalLenStr)
	// filenameLen filename filesize validBit

	str1 := numExtend(len(createFileName), 8)
	str3 := numExtend(int(defaultSize), 8)
	validBit := "1"

	strWrite := str1 + createFileName + str3 + validBit
	write("config", uint64(totalLen), strWrite)

	// 更新 "config" 中的 totalLen
	totalLen += len(strWrite)
	str4 := numExtend(totalLen, 8)
	write("config", 8, str4)

	// 返回创建完成的信息
	var retMsg backendMsg
	retMsg.Type = "CREATESUCCESS"
	data := make([]string, 0)
	data = append(data, createFileName)
	retMsg.Data = data
	bytes, e := json.Marshal(retMsg)
	if e != nil {
		fmt.Println("[CREATE] error in marshal message")
		return
	} else {
		err := conn.WriteMessage(1, bytes)
		if err != nil {
			fmt.Println("[CREATE] error in write back message")
			return
		}
	}

	// allConnMap
	// 给其他前端返回信息
	var retMsg2 backendMsg
	retMsg2.Type = "CREATESUCCESS2"
	for k, _ := range allConnMap {
		if k == conn {
			continue
		} else {
			bytes2, _ := json.Marshal(retMsg2)
			err := k.WriteMessage(1, bytes2)
			fmt.Println("[SBBBBBBB]:", string(bytes2))
			if err != nil {
				fmt.Println("[CREATE] error in send back message to other frontends")
			} else {
				fmt.Println("[CREATE] success in send back msg")
			}
		}
	}

	// 写 log 文件
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	logData := "[CREATE] " + timeStr + " " + userName + " create the file " + createFileName + "\t"
	writeLog(logData)
}

// 暂时没用
func handleDelete(delFileName string, conn *websocket.Conn) {
	fmt.Println("[DELETE] try to delete file: ", delFileName)

	// 首先检查是否有其他的conn在编辑文件
	existConnSlice, ok := connMap[delFileName]
	if ok {
		// 有文件信息，需要检查是否有连接
		if len(existConnSlice) != 0 {
			// 不能删除
			var retMsg backendMsg
			retMsg.Type = "Fail"
			bytes, e := json.Marshal(retMsg)
			if e != nil {
				fmt.Println("[DELETE] Error in send back")
			} else {
				fmt.Println("Send back: ", string(bytes))
				conn.WriteMessage(1, bytes)
			}
		}
	}

	// 删除 fileMap 中的记录
	delete(fileMap, delFileName)

	// fileSystem api
	// 删除存储的文件
	deletee(delFileName)

	// 修改 config 中的文件数
	fileNumStr := read("config", 0, 8)
	fileNum, _ := strconv.Atoi(fileNumStr)
	// fileNumStr2 := strconv.Itoa(fileNum - 1)
	fileNumStr2 := numExtend(fileNum-1, 8)
	write("config", 0, fileNumStr2)

	// 在 config 文件中记录本次删除
	// code

	// 返回成功信息
	var retMsg backendMsg
	retMsg.Type = "Success"
	bytes, e := json.Marshal(retMsg)
	if e != nil {
		fmt.Println("[DELETE] Error in send back")
	} else {
		fmt.Println("[DELETE] Send back: ", string(bytes))
		conn.WriteMessage(1, bytes)
	}
}

// 处理 update 操作，将更新写入文件并向所有参与的前端发送信息
func handleUpdate(receiveData msgInfo, messageType int, p []byte, conn *websocket.Conn) {
	// fileSystem api
	// 计算需要进行更改的 offset
	currRow := receiveData.Row
	currCol := receiveData.Column

	currPosition := uint64(currRow*colNum*blockSize) + uint64(currCol*blockSize)
	writeValue := receiveData.NewValue
	writeFileName := receiveData.FileName

	// 从文件中读取oldValue
	oldValue := read(writeFileName, currPosition, blockSize)
	slicePos := strings.Index(oldValue, "     ")
	fmt.Println("[UPDATE] the old value end is:", slicePos)
	if slicePos == 0 {
		oldValue = " "
	} else {
		oldValue = oldValue[:slicePos]
	}
	fmt.Println("[UPDATE] the old value is:", oldValue)

	// 对 writeValue 进行长度的检查，如有必要则进行截断
	valueLen := len(writeValue)
	if valueLen > int(blockSize) {
		writeValue = writeValue[0:blockSize]
		fmt.Println("[UPDATE] the value is too long, cut the value as:", writeValue)
	}

	fmt.Println("[UPDATE] try to update:", writeValue, "in Row:", currRow, "Col:", currCol, "in file:", writeFileName)
	fmt.Println("[UPDATE] the offset of value", writeValue, "is", currPosition)

	write(writeFileName, currPosition, writeValue)

	// LOCK
	// 放锁
	lockIndex := "/" + writeFileName + "-" + strconv.Itoa(int(currRow)) + "-" + strconv.Itoa(int(currCol))
	currNode, ok := nodeMap[conn]
	if ok {
		// 从 lockMap 中移除对应 lockIndex
		delete(lockMap, lockIndex)
		fmt.Println("[LOCKMAP]", lockMap)

		unlock(currNode, lockIndex)
		fmt.Println("[UPDATE] unlock the lock:", lockIndex)
	} else {
		fmt.Println("[UPDATE] fail to load the node info")
	}
	// LOCK END

	// 将修改后的内容原样转发给所有参与文件链接的前端
	// 发送格式待定
	currConnSlice := connMap[writeFileName]
	for _, value := range currConnSlice {
		fmt.Println("[UPDATE] write back to a connection")
		if err := value.WriteMessage(messageType, p); err != nil {
			fmt.Println("[UPDATE] err in write back")
			log.Println(err)
			// return
		} else {
			fmt.Println("[UPDATE] success write back message", string(p))
		}
	}

	// // 获取旧值并存入log
	// oldValue = "receiveData.OldValue"
	// 写 log 文件
	userName := receiveData.UserName
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	logData := "[UPDATE] " + timeStr + " user:" + userName + " set row:" + strconv.Itoa(int(currRow)) + " col:" + strconv.Itoa(int(currCol)) + " from:" + oldValue + " to:" + writeValue + " in file:" + writeFileName + "\t"
	writeLog(logData)
}

// 将链接信息存储到 connMap 中
func createConnect(connFileName string, conn *websocket.Conn) {
	// LOCK
	// 为当前链接创建一个 node
	currNode := connect()
	nodeMap[conn] = currNode
	fmt.Println("[CONNECT] create node for the connection")

	// 查看当前文件是否有其他人在编辑
	// 若有则只更新对应slice即可
	// 否则创建新的键值对
	currConnSlice, ok := connMap[connFileName]
	if ok {
		currConnSlice = append(currConnSlice, conn)
		connMap[connFileName] = currConnSlice
		fmt.Println("[CONNECT] file:", connFileName, "have ", len(currConnSlice), "editor now")
		return
	} else {
		currConnSlice = make([]*websocket.Conn, 0)
		currConnSlice = append(currConnSlice, conn)
		connMap[connFileName] = currConnSlice
		fmt.Println("[CONNECT] file:", connFileName, "have 1 editor now")
		return
	}
}

// 将所有 fileName 转为 json 并发回前端
func sendBakcMsg(conn *websocket.Conn) {
	allFileName := make([]string, 0)
	for k, _ := range fileMap {
		allFileName = append(allFileName, k)
	}
	// 将返回数据组织成backendMsg形式
	retMsg := backendMsg{"FileNameSlice", allFileName}
	bytes, e := json.Marshal(retMsg)
	if e != nil {
		fmt.Println("Error")
	} else {
		// 1 是 messageType 不一定对
		fmt.Println("[CONNECT] Send back: ", string(bytes))
		conn.WriteMessage(1, bytes)
	}
}

// 处理 close 操作，将参与的前端从记录文件列表中移除
// 只是当前的处理信息被删除，websocket 的链接依然保留
func handleClose(closeFileName string, conn *websocket.Conn) {
	currConnSlice, ok := connMap[closeFileName]
	if ok {
		newConnSlice := make([]*websocket.Conn, 0)
		for _, val := range currConnSlice {
			if val == conn {
				continue
			} else {
				newConnSlice = append(newConnSlice, val)
			}
		}
		// 若删除的是最后一个，则将fileName从map中移除
		// 否则更新map
		if len(newConnSlice) == 0 {
			delete(connMap, closeFileName)
			fmt.Println("[CLOSE] file:", closeFileName, " have no editor now")
		} else {
			connMap[closeFileName] = newConnSlice
			fmt.Println("[CLOSE] file:", closeFileName, "have ", len(newConnSlice), " editor now")
		}
	} else {
		return
	}
}

// 将链接加入allConnMap中
func commonConnect(conn *websocket.Conn) {
	allConnMap[conn] = 0
}

// 当连接断开时被调用
// 遍历所有connMap, 将conn全部移除
func removeConn(conn *websocket.Conn) {
	// LOCK
	// 将 conn 与对应 node 从 nodeMap 中删除
	// 关闭 currNode 后，持有的锁会被自动释放
	currNode, ok := nodeMap[conn]
	if ok {
		close(currNode)
		delete(nodeMap, conn)
		fmt.Println("[EXIT] delete the conn successfully")
	} else {
		fmt.Println("[EXIT] conn not exists")
	}

	for k, _ := range connMap {
		handleClose(k, conn)
	}

	// 将 conn 从 allConnMap 中移除
	delete(allConnMap, conn)
	fmt.Println("[ALLCONNMAP] the map len is:", len(allConnMap))
	fmt.Println("[ALLCONNMAP]", allConnMap)

	// 关闭 websocket
	conn.Close()
}

// 处理 edit 操作，向所有参与文件修改的前端发送信息
func handleEdit(editFile msgInfo, messageType int, p []byte, conn *websocket.Conn) {
	editFileName := editFile.FileName
	editRow := editFile.Row
	editCol := editFile.Column
	fmt.Println("[Edit] try to edit file: ", editFileName)

	// LOCK
	// 在 edit 阶段尝试拿锁，如果失败则将信息输出给当前前端，否则继续执行
	lockIndex := "/" + editFileName + "-" + strconv.Itoa(int(editRow)) + "-" + strconv.Itoa(int(editCol))
	fmt.Println("[EDIT] try to get lock:", lockIndex)
	currNode, ok := nodeMap[conn]
	if ok {
		getLock := locknotblock(currNode, lockIndex)
		if getLock {
			// 向 lockMap 中存储信息
			lockMap[lockIndex] = editFile.UserName
			fmt.Println("[LOCKMAP]", lockMap)
			fmt.Println("[EDIT] get the lock for:", lockIndex)
		} else {
			fmt.Println("[EDIT] fail to get the lock for:", lockIndex)
			var retMsg lockErrMsg
			retMsg.Type = "FAILOCK"
			retMsg.Row = editRow
			retMsg.Column = editCol
			retMsg.SuccessUsername = lockMap[lockIndex]
			retMsg.RejectUsername = editFile.UserName
			bytes, _ := json.Marshal(retMsg)
			conn.WriteMessage(messageType, bytes)
			fmt.Println("[EDIT] write back:", string(bytes))
			return
		}
	} else {
		fmt.Println("[EDIT] fail to load node info")
	}
	// END LOCK

	currConnSlice := connMap[editFileName]
	for _, value := range currConnSlice {
		if err := value.WriteMessage(messageType, p); err != nil {
			fmt.Println("[EDIT] err in write back")
			log.Println(err)
			return
		}
	}
}

// 将 fileName 的 validBit 修改为输入值
func changeValidBit(fileName string, validBit int) {
	var v string
	if validBit == 1 {
		v = "1"
	} else {
		v = "0"
	}
	totalLenStr := read("config", 8, 8)
	totalLen, _ := strconv.Atoi(totalLenStr)
	configStr := read("config", 0, uint64(totalLen))
	fmt.Println("[CHANGE BIT] config file data:", configStr)
	fileNamePos := strings.Index(configStr, fileName)
	fmt.Println("[CHANGE BIT] current filename:", fileName, "index is:", fileNamePos)
	validBitPos := fileNamePos + len(fileName) + 8
	fmt.Println("[CHANGE BIT] the position for validbit is:", validBitPos)
	write("config", uint64(validBitPos), v)
}

// 将文件放入回收站
// 首先进行检查，只有当文件没有链接时才能进行删除操作，即 connMap 中没有对应 fileName 的key
// 否则删除失败，返回错误信息
// 将文件从 fileMap 中移除，同时更新 config 文件中的 validBit，将文件添加到 trashMsp 中
// 将更新后的 fileMap 转发给前端
func moveToTrash(mttFilename string, conn *websocket.Conn, userName string) {
	fmt.Println("[MTT] try to move file", mttFilename, "to trash")
	_, ok := connMap[mttFilename]
	if ok {
		// 有其他前端在编辑文件
		// 返回错误
		var retMsg backendMsg
		retMsg.Type = "MTTERROR"
		bytes, e := json.Marshal(retMsg)
		if e != nil {
			fmt.Println("[MTT] err in marshal")
			return
		} else {
			err := conn.WriteMessage(1, bytes)
			if err != nil {
				fmt.Println("[MTT] err in send back msg")
			}
			return
		}
		return
	}

	// 更新 trashMap 与 fileMap
	trashMap[mttFilename] = fileMap[mttFilename]
	delete(fileMap, mttFilename)

	// 更新 config 文件中的 validBit
	changeValidBit(mttFilename, 0)

	// 将更新后的 fileMap 发回前端
	var retMsg backendMsg
	retMsg.Type = "MTTSUCCESS"
	retMsg.Data = traverseFileMap()
	bytes, e := json.Marshal(retMsg)
	if e != nil {
		fmt.Println("[MTTSUCCESS] err in marshal")
		// return
	} else {
		err := conn.WriteMessage(1, bytes)
		if err != nil {
			fmt.Println("[MTTSUCCESS] err in send back msg")
		}
		// return
	}

	// allConnMap
	// 将信息发回其他的前端
	for k, _ := range allConnMap {
		if k == conn {
			continue
		} else {
			err := k.WriteMessage(1, bytes)
			if err != nil {
				fmt.Println("[MTT] err in send msg to other frontends")
			} else {
				fmt.Println("[MTT] success in send msg to other frontends")
			}
		}
	}

	// 写 log 文件
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	logData := "[MTT] " + timeStr + " " + userName + " move the file " + mttFilename + " to trash\t"
	writeLog(logData)

	return
}

// 从回收站中将文件移除
// 将文件从 trashMap 中移除并添加到 fileMap 中
// 更新 config 文件
func recoverFile(recoverFilename string, userName string) {
	fmt.Println("[RECOVER] try to recover file:", recoverFilename)
	// 更新 fileMap 与 trashMap
	fileMap[recoverFilename] = trashMap[recoverFilename]
	delete(trashMap, recoverFilename)

	// 更新 config 文件中的 validBit
	changeValidBit(recoverFilename, 1)

	// 写 log 文件
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	logData := "[RECOVER] " + timeStr + " " + userName + " recover the file " + recoverFilename + "\t"
	writeLog(logData)

}

func recoverWithRet(recoverFilename string, userName string, conn *websocket.Conn) {
	fmt.Println("[RECOVER] try to recover file:", recoverFilename)
	// 更新 fileMap 与 trashMap
	fileMap[recoverFilename] = trashMap[recoverFilename]
	delete(trashMap, recoverFilename)

	// 更新 config 文件中的 validBit
	changeValidBit(recoverFilename, 1)

	// 写 log 文件
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	logData := "[RECOVER] " + timeStr + " " + userName + " recover the file " + recoverFilename + "\t"
	writeLog(logData)

	var retMsg backendMsg
	retMsg.Type = "RECOVERET"
	bytes, _ := json.Marshal(retMsg)
	conn.WriteMessage(1, bytes)

	// allConnMap
	for k, _ := range allConnMap {
		if k == conn {
			continue
		} else {
			k.WriteMessage(1, bytes)
		}
	}
}

// 将文件彻底删除
// 从回收站中移除
// 将 config 中的记录删除
func clearTrash(fileName string, userName string, conn *websocket.Conn) {
	fmt.Println("[CLEAR TRASH] try to delete file:", fileName)
	_, ok := trashMap[fileName]
	if ok {
		delete(trashMap, fileName)
	} else {
		fmt.Println("[CLEAR TRASH] error:", fileName, "is not a trash file")
		return
	}

	// 在 config 中将文件记录彻底删除
	// config 中文件结构为
	// filenameLen(uint64) filename filesize(uint64) validBit
	delLen := 8 + len(fileName) + 8 + 1

	// 在 config file 中读出整体并进行修改
	totalLenStr := read("config", 8, 8)
	totalLen, _ := strconv.Atoi(totalLenStr)

	configStr := read("config", 0, uint64(totalLen))
	fmt.Println("[CLEAR TRASH] the config file data is:", configStr)
	fileNumStr := configStr[0:8]
	fileLenStr := configStr[8:16]
	fileNum, _ := strconv.Atoi(fileNumStr)
	fileLen, _ := strconv.Atoi(fileLenStr)
	fmt.Println("[CLEAR TRASH] the filenum and filelen is:", fileNum, fileLen)
	newFileNum := numExtend(fileNum-1, 8)
	newFileLen := numExtend(fileLen-delLen, 8)
	startPos := strings.Index(configStr, fileName) - 8
	endPos := startPos + delLen
	frontStr := configStr[16:startPos]
	backStr := configStr[endPos:]
	newConfigStr := newFileNum + newFileLen + frontStr + backStr
	fmt.Println("[CLEAR TRASH] the new config file data is:", newConfigStr)

	// 将新的 config file 写入
	write("config", 0, newConfigStr)

	// 写 log 文件
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	logData := "[CLEAR TRASH] " + timeStr + " " + userName + " really delete the file " + fileName + "\t"
	writeLog(logData)

	// 在文件系统中删除文件
	deletee(fileName)

	// 返回信息
	var retMsg backendMsg
	retMsg.Type = "CTRET"
	bytes, _ := json.Marshal(retMsg)
	conn.WriteMessage(1, bytes)

	// allConnMap
	for k, _ := range allConnMap {
		if k == conn {
			continue
		} else {
			k.WriteMessage(1, bytes)
		}
	}
}

// 展示 log
func showLog(conn *websocket.Conn) {
	totalLenStr := read("gDocLog", 0, 8)
	totalLen, _ := strconv.Atoi(totalLenStr)
	logStr := read("gDocLog", 0, uint64(totalLen))
	logStr = logStr[9:]
	// 按照 \t 进行分离
	strSlice := strings.Split(logStr, "\t")
	fmt.Println("[LOG] the log is:", strSlice)
	retMsg := backendMsg{"LOG", strSlice}
	bytes, _ := json.Marshal(retMsg)
	conn.WriteMessage(1, bytes)
}

// 回滚一格
// 当用户在某个文件进行编辑时允许此操作
// 需要用户名，文件名来确定回滚的条目
func handleRollBack(username string, filename string, conn *websocket.Conn) {
	// 通过log读出文件编辑情况
	totalLenStr := read("gDocLog", 0, 8)
	totalLen, _ := strconv.Atoi(totalLenStr)
	logStr := read("gDocLog", 0, uint64(totalLen))
	logStr = logStr[9:]
	// 按照 \t 进行分离
	strSlice := strings.Split(logStr, "\t")
	fmt.Println("[ROLLBACK] the log slice is:", strSlice)

	// 对 log 进行反向使得后编辑的在前
	// 构造 updateSlice
	strLen := len(strSlice)
	fmt.Println("[ROLLBACK] totalLen is:", strLen)
	if strLen <= 1 {
		return
	}
	updateSlice := make([]msgOldInfo, 0)

	// record跳过
	skipNum := 0

	var aimMsg msgOldInfo
	aimMsg.Type = "nil"

	rcSlice := make([]rc, 0)

	for i := strLen - 2; i >= 0; i-- {
		currStr := strSlice[i]
		fmt.Println("[ROLLBACK] log index is:", i)
		fmt.Println("[ROLLBACK] current log is:", currStr)

		if currStr[0:8] == "[UPDATE]" {
			// 处理当前str信息
			var currInfo msgOldInfo

			// [UPDATE] user: xx set row:1 col:1 from:old to:new in file:file1
			userBegin := strings.Index(currStr, "user:") + 5
			rowBegin := strings.Index(currStr, "row:") + 4
			colBegin := strings.Index(currStr, "col:") + 4
			oldBegin := strings.Index(currStr, "from:") + 5
			newBegin := strings.Index(currStr, "to:") + 3
			fileBegin := strings.Index(currStr, "file:") + 5

			userEnd := rowBegin - 9
			rowEnd := colBegin - 5
			colEnd := oldBegin - 6
			oldEnd := newBegin - 4
			newEnd := fileBegin - 9
			fileEnd := len(currStr)

			userStr := currStr[userBegin:userEnd]
			rowStr := currStr[rowBegin:rowEnd]
			rowNum, _ := strconv.Atoi(rowStr)
			colStr := currStr[colBegin:colEnd]
			colNum, _ := strconv.Atoi(colStr)
			oldStr := currStr[oldBegin:oldEnd]
			newStr := currStr[newBegin:newEnd]
			fileStr := currStr[fileBegin:fileEnd]

			currInfo.UserName = userStr
			currInfo.Row = uint64(rowNum)
			currInfo.Column = uint64(colNum)
			currInfo.OldValue = oldStr
			currInfo.NewValue = newStr
			currInfo.FileName = fileStr
			currInfo.Type = "ROLLBACK"

			updateSlice = append(updateSlice, currInfo)

			if userStr == username && fileStr == filename {
				if skipNum == 0 {
					aimMsg = currInfo
					break
				} else {
					skipNum--
					continue
				}
			}

			// 若为其他用户编辑相同文件，先将col与row存储
			if userStr != username && fileStr == filename {
				currRC := rc{currInfo.Row, currInfo.Column}
				rcSlice = append(rcSlice, currRC)
			}
		}

		// [ROLLBACK] user:name1 file:filename"
		if currStr[0:10] == "[ROLLBACK]" {
			us := strings.Index(currStr, "user:") + 5
			fs := strings.Index(currStr, "file:") + 5
			ue := fs - 6
			fe := len(currStr)

			u := currStr[us:ue]
			f := currStr[fs:fe]

			if u == username && f == filename {
				skipNum++
				fmt.Println("[ROLLBACK]", u, f)
				fmt.Println("[ROLLBACK] rollback time is:", skipNum)
				continue
			}
		}
	}

	fmt.Println("[ROLLBACK] updateSlice:", updateSlice)

	// aimMsg即为更新结构
	if aimMsg.Type == "nil" {
		// 没有找到回滚项
		fmt.Println("[ROLLBACK] fail to find rollback info")
		var m1 backendMsg
		m1.Type = "ROLLBACKEMPTY"
		b1, _ := json.Marshal(m1)
		conn.WriteMessage(1, b1)
		return
	}

	// 回滚结果存入
	currRow := aimMsg.Row
	currCol := aimMsg.Column

	fmt.Println("[ROLLBACK] check the rcSlice:", rcSlice)
	// 判断是否有其他用户写
	for j := 0; j < len(rcSlice); j++ {
		preR := rcSlice[j].Row
		preC := rcSlice[j].Col
		if preR == currRow && preC == currCol {
			fmt.Println("[ROLLBACK] fail, someone edited the same position before")
			var m3 backendMsg
			m3.Type = "ROLLBACKERR"
			b3, _ := json.Marshal(m3)
			conn.WriteMessage(1, b3)
			return
		}
	}

	currValue := aimMsg.OldValue
	currFN := aimMsg.FileName

	// 判断要写的地方是否有锁
	lockIndex := "/" + currFN + "-" + strconv.Itoa(int(currRow)) + "-" + strconv.Itoa(int(currCol))
	fmt.Println("[ROLLBACK] try to get lock:", lockIndex)
	_, ok := lockMap[lockIndex]
	if ok {
		fmt.Println("[ROLLBACK] fail, the position is locked")
		var m4 backendMsg
		m4.Type = "ROLLBACKLOCK"
		b4, _ := json.Marshal(m4)
		conn.WriteMessage(1, b4)
		return
	}

	currPosition := uint64(currRow*colNum*blockSize) + uint64(currCol*blockSize)
	nv := aimMsg.NewValue
	if len(currValue) < len(nv) {
		currValue = currValue + getStr(uint64(len(nv)-len(currValue)))
	}
	write(currFN, currPosition, currValue)
	fmt.Println("[ROLLBACK] check:", aimMsg.UserName, currFN, currValue)

	// 将结果转发给其他参与编辑的前端
	currConnSlice := connMap[currFN]
	p, _ := json.Marshal(aimMsg)
	for _, value := range currConnSlice {
		fmt.Println("[ROLLBACK] write back to a connection:", string(p))
		if err := value.WriteMessage(1, p); err != nil {
			fmt.Println("[ROLLBACK] err in write back")
			log.Println(err)
			// return
		} else {
			fmt.Println("[ROLLBACK] success write back message", string(p))
		}
	}

	// 写log
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	logData := "[ROLLBACK] " + timeStr + " user:" + aimMsg.UserName + " file:" + aimMsg.FileName + "\t"
	writeLog(logData)
}

func getTrashList(conn *websocket.Conn) {
	retStrSlice := make([]string, 0)
	for k, _ := range trashMap {
		retStrSlice = append(retStrSlice, k)
	}
	fmt.Println("[TRASHLIST]:", retStrSlice)
	retMsg := backendMsg{"TRASHLIST", retStrSlice}
	bytes, _ := json.Marshal(retMsg)
	conn.WriteMessage(1, bytes)
}

// -------------  handlers end ----------------
