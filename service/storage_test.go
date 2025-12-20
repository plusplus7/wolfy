package service

import (
	"fmt"
	"testing"
)

func Test_WalkPackages(t *testing.T) {
	testPath := "/Users/bytedance/Downloads/Package/"
	fromPackage, err := collectSongInfoFromPackage(testPath)
	if err != nil {
		return
	}
	fmt.Println(fromPackage)
}
