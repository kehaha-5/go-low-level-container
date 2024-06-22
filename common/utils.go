package common

import (
	"math/rand"
	"os"
	"time"
)

// 文件夹是否存在 存在ture 不存在false
func PathExist(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// 文件是否存在 存在ture 不存在false
func FileExist(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func RangeStr(n int) string {
	letterSeed := "0123456789abcde"
	rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letterSeed[rand.Intn(len(letterSeed))]
	}
	return string(b)
}
