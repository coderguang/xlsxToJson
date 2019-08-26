package main

import (
	"os"
	xlsxToJsonXlsx "xlsxToJson/src/xlsx"

	"github.com/coderguang/GameEngine_go/sglog"
)

func main() {

	arg_num := len(os.Args) - 1
	sglog.Info("本次需生成文件数量为 %d\n", arg_num)
	for i := 1; i <= arg_num; i++ {
		xlsxToJsonXlsx.StartGenFile(os.Args[i])
	}
	sglog.Info("全部文件生成完毕,文件数量为:%d", arg_num)
}
