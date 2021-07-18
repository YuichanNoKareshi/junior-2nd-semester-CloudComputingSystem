package main

import (
	"strconv"
)

// 获取固定长的的字符串
func getStr(strSize uint64) string {
	ret := ""
	var i uint64
	for i = 0; i < strSize; i++ {
		ret += " "
	}
	return ret
}

// 将 srcNum(int) 转为 aimLen 长度的字符串，空闲位置用 0 补齐
func numExtend(srcNum int, aimLen int) string {
	tempStr := strconv.Itoa(srcNum)
	tempLen := len(tempStr)
	if tempLen >= aimLen {
		// 截取后 aimLen 长度数字结果作为输出(10)
		tempStr := tempStr[tempLen-aimLen : tempLen]
		return tempStr
	}
	zeroStr := ""
	zeroNum := aimLen - tempLen
	for i := 0; i < zeroNum; i++ {
		zeroStr += "0"
	}
	tempStr = zeroStr + tempStr
	return tempStr
}

func traverseFileMap() []string {
	retSlice := make([]string, 0)
	for key, _ := range fileMap {
		retSlice = append(retSlice, key)
	}
	return retSlice
}
