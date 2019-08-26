package xlsxToJsonXlsx

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/coderguang/GameEngine_go/sgthread"

	"github.com/coderguang/GameEngine_go/sglog"

	"github.com/360EntSecGroup-Skylar/excelize"
)

func StartGenFile(filename string) {

	sglog.Info("开始执行配置表 %s 解析到json文件\n", filename)

	xls, err := excelize.OpenFile(filename)

	if err != nil {
		sglog.Error("读取文件报错,%s", err)
		sgthread.DelayExit(2)
	}
	StartGetRoot(xls)

	sglog.Info("配置表 %s 全部解析完成!\n=============\n\n", filename)
}

func StartGetRoot(xls *excelize.File) {
	configList, _ := ReadRootField(xls)

	if len(configList) <= 0 {
		sglog.Error("root 表未配置任何数据")
		sgthread.DelayExit(2)
	}

	for k, _ := range configList {
		sglog.Info("开始执行生成:%s", configList[k].name)
		StartGenConfig(xls, configList[k])
		sglog.Info("执行生成 %s 完成\n==================================================================\n\n\n", configList[k].name)
	}
}

func StartGenConfig(xls *excelize.File, config *xlsxToJsonDef.RootDirStruct) {

	if "" == config.name {
		sglog.Error("主表为空")
		sgthread.DelayExit(2)
	}

	sglog.Info("主表sheet名称为:%s", config.name)

	datalist, err := ReadField(xls, config.name)

	if err != nil {
		sglog.Error("读取字段名称出错，ex=%s", err)
		sgthread.DelayExit(2)
	}

	typeCell, err := GetTypeCellAndCheck(xls, config.name)

	if err != nil {
		sglog.Error("获取生成配置类型错误，ex=%s", err)
		sgthread.DelayExit(2)
	}

	filename := config.name + ".json"
	serverStr := ParseXlxs(xls, config.name, typeCell, datalist, xlsxToJsonDef.StrategyType_Server)
	CheckJsonValidAndWriteFile(serverStr, config.serverDir, filename, "服务器")

	clientStr := ParseXlxs(xls, config.name, typeCell, datalist, xlsxToJsonDef.StrategyType_Client)
	//CheckJsonValidAndWriteFile(clientStr, config.clientDir, filename, "客户端json")
	//CheckJsonValid(clientStr)

	if CheckJsonValid(clientStr) {
		luafilename := config.name + ".lua"
		clientluaStr := TransformJsonTolua(clientStr, typeCell)
		WriteConfigFile(clientluaStr, config.clientDir, luafilename, "客户端lua")
	} else {
		sglog.Error("生成lua配置错误，ex=%s", err)
		sgthread.DelayExit(2)
	}
}

func ReadRootField(xls *excelize.File) ([]xlsxToJsonDef.RootDirStruct, error) {

	dataStructList := []xlsxToJsonDef.RootDirStruct{}

	rows, err := xls.Rows(xlsxToJsonDef.TABLE_ROOT_SHEET_NAME)
	if err != nil {
		sglog.Error("读取 %s 错误,err=%s", xlsxToJsonDef.TABLE_ROOT_SHEET_NAME, err)
		sgthread.DelayExit(2)
	}

	rowIndex := 1
	for rows.Next() {
		if rowIndex <= int(xlsxToJsonDef.TABLE_ROOT_ROW_DES) {
			rowIndex++
			continue
		}
		row := rows.Columns()
		colIndex := 1
		if len(row) <= 0 {
			break
		}
		for _, colCell := range row {
			if colIndex > int(xlsxToJsonDef.TABLE_ROOT_COLUMN_CONFIG_CLIENT_DIR) {
				break
			}
			if "" == colCell {
				break
			}
			dataIndex := rowIndex - int(xlsxToJsonDef.TABLE_ROOT_ROW_DES) - 1
			if dataIndex > len(dataStructList) {
				sglog.Error("root 表字段名称配置长度不一致 serverDir rowIndex=%s,colIndex=%s,名称:", rowIndex, colIndex, colCell)
				sgthread.DelayExit(2)
			}

			if int(xlsxToJsonDef.TABLE_ROOT_COLUMN_CONFIG_NAME) == colIndex {
				tmp := newRootDirStruct()
				tmp.name = colCell
				dataStructList = append(dataStructList, tmp)
			} else if int(xlsxToJsonDef.TABLE_ROOT_COLUMN_CONFIG_SERVER_DIR) == colIndex {
				dataStructList[dataIndex].serverDir = colCell
			} else if int(xlsxToJsonDef.TABLE_ROOT_COLUMN_CONFIG_CLIENT_DIR) == colIndex {
				dataStructList[dataIndex].clientDir = colCell
			}
			colIndex++
		}
		rowIndex++
	}

	if len(dataStructList) <= 0 {
		sglog.Error("root表为空")
		sgthread.DelayExit(2)
	}

	for _, v := range dataStructList {
		if v.CheckEmpty() {
			sglog.Error("root表读取失败,有字段数据为空")
			v.Show()
			sgthread.DelayExit(2)
		}
	}

	return dataStructList, err
}

func GetTypeCellAndCheck(xls *excelize.File, sheetName string) (xlsxToJsonDef.TableType, error) {

	typeCellStr := xls.GetCellValue(sheetName, xlsxToJsonDef.TABLE_FORMAT_TYPECELL_POS)

	typeCellInt, err := strconv.Atoi(typeCellStr)
	if err != nil {
		sglog.Error("读取配置类型 %s 数据报错,type=%s", xlsxToJsonDef.TABLE_FORMAT_TYPECELL_POS, typeCellInt)
		sgthread.DelayExit(2)
	}
	typeCell := TableType(typeCellInt)

	if xlsxToJsonDef.TableType_object != typeCell && TableType_array != typeCell {
		sglog.Error("配置类型 A1 数据错误,既不是0（数组）也不是1（对象）,当前值为:%s", typeCell)
		sgthread.DelayExit(2)
	}

	if xlsxToJsonDef.TableType_object == typeCell {
		log.Println("该表生成的数据结构为对象")
	} else {
		log.Println("该表生成的数据结果为数组")
	}

	return typeCell, err
}

func ReadField(xls *excelize.File, sheetName string) ([]xlsxToJsonDef.DataStruct, error) {
	dataStructList := []xlsxToJsonDef.DataStruct{}

	rows, err := xls.Rows(sheetName)
	rowIndex := 1
	for rows.Next() {
		if rowIndex > int(xlsxToJsonDef.TABLE_FORMAT_ROW_DATATYPE) {
			break
		}
		row := rows.Columns()
		if len(row) <= 0 {
			break
		}
		colIndex := 1
		for _, colCell := range row {
			if int(xlsxToJsonDef.TABLE_FORMAT_COLUMN_DES) == colIndex {
				colIndex++
				continue
			}
			if "" == colCell {
				break
			}

			dataIndex := colIndex - int(xlsxToJsonDef.TABLE_FORMAT_COLUMN_DES) - 1
			if dataIndex > len(dataStructList) {
				sglog.Error("字段名称配置长度不一致 rowIndex=%s,colIndex=%s,名称:", rowIndex, colIndex, colCell)
				sgthread.DelayExit(2)
			}

			if int(xlsxToJsonDef.TABLE_FORMAT_ROW_DES) == rowIndex {
				tmp := NewDataStruct()
				tmp.desc = colCell
				dataStructList = append(dataStructList, tmp)
			} else if int(xlsxToJsonDef.TABLE_FORMAT_ROW_NAME) == rowIndex {
				dataStructList[dataIndex].name = colCell
			} else if int(xlsxToJsonDef.TABLE_FORMAT_ROW_CONFIG_STRATEGY) == rowIndex {
				strategyType, err := strconv.Atoi(colCell)
				if err != nil {
					sglog.Error("策略类型错误,rowIndex=%s,类型:%s", rowIndex, colCell)
					sgthread.DelayExit(2)
				}
				if strategyType != int(xlsxToJsonDef.StrategyType_All) && strategyType != int(xlsxToJsonDef.StrategyType_Server) && strategyType != int(xlsxToJsonDef.StrategyType_Client) {
					sglog.Error("策略类型错误,rowIndex=%s,类型:%s", rowIndex, strategyType)
					sgthread.DelayExit(2)
				}
				dataStructList[dataIndex].strategyType = StrategyType(strategyType)
			} else if int(xlsxToJsonDef.TABLE_FORMAT_ROW_DATATYPE) == rowIndex {
				dataType, err := strconv.Atoi(colCell)
				if err != nil {
					sglog.Error("数据类型错误,rowIndex=%s,类型:%s", rowIndex, colCell)
					sgthread.DelayExit(2)
				}
				if dataType != int(xlsxToJsonDef.DataType_raw) && dataType != int(xlsxToJsonDef.DataType_string) && dataType != int(xlsxToJsonDef.DataType_link) {
					sglog.Error("数据类型错误,rowIndex=%s,类型:%s", rowIndex, dataType)
					sgthread.DelayExit(2)
				}
				dataStructList[dataIndex].dataType = DataType(dataType)
			}
			colIndex++
		}
		rowIndex++
	}

	if len(dataStructList) <= 0 {
		sglog.Error("数据表为空,请检测:%s", sheetName)
		sgthread.DelayExit(2)
	}

	for _, v := range dataStructList {
		if v.CheckEmpty() {
			sglog.Error("解析表数据结构配置错误,配置为空:表名称 %s", sheetName)
			v.Show()
			sgthread.DelayExit(2)
		}
	}

	return dataStructList, err
}

func GenColCell(xls *excelize.File, sheetName string, dataStruct *xlsxToJsonDef.DataStruct, colCell string, strategyType xlsxToJsonDef.StrategyType) string {
	tmpColumnStr := "\"" + dataStruct.name + "\":"
	switch dataStruct.dataType {
	case xlsxToJsonDef.DataType_raw:
		{
			if "" == colCell {
				sglog.Error("解析表:%s 错误:数据类型0不允许为空，列 %s", sheetName, dataStruct.desc)
				sgthread.DelayExit(2)
			}
			tmpColumnStr += colCell
		}
	case xlsxToJsonDef.DataType_string:
		{
			tmpColumnStr += "\"" + colCell + "\""
		}
	case xlsxToJsonDef.DataType_link:
		{
			tmpColumnStr += ParseChildXlxs(xls, dataStruct.name, colCell, strategyType)
		}
	}
	return tmpColumnStr
}

func GetStrPrefixByTypeCell(typeCell xlsxToJsonDef.TableType) string {
	if xlsxToJsonDef.TableType_array == typeCell {
		return "["
	} else if xlsxToJsonDef.TableType_object == typeCell {
		return "{"
	} else {
		return ""
	}
}

func GetStrSuffixByTypeCell(typeCell xlsxToJsonDef.TableType) string {
	if xlsxToJsonDef.TableType_array == typeCell {
		return "]"
	} else if xlsxToJsonDef.TableType_object == typeCell {
		return "}"
	} else {
		return ""
	}
}

func GetInnerStrPrefixByTypeCell(typeCell xlsxToJsonDef.TableType) string {
	if xlsxToJsonDef.TableType_array == typeCell {
		return "{"
	} else {
		return ""
	}
}

func GetInnerStrSuffixByTypeCell(typeCell xlsxToJsonDef.TableType) string {
	if xlsxToJsonDef.TableType_array == typeCell {
		return "}"
	} else {
		return ""
	}
}

func ConnectTwoString(firstInsert bool, oldStr string, newStr string) (bool, string) {
	flag := firstInsert
	if firstInsert {
		oldStr += newStr
		flag = false
	} else {
		oldStr += "," + newStr
	}
	return flag, oldStr
}

func IsIgnoreField(writeFileType xlsxToJsonDef.StrategyType, configStrategy xlsxToJsonDef.StrategyType) bool {
	if configStrategy == xlsxToJsonDef.StrategyType_All {
		return false
	}
	if writeFileType == configStrategy {
		return false
	} else {
		return true
	}
}

//解析子表，子表的子项必然是{}格式
func ParseChildXlxs(xls *excelize.File, name string, strlist string, strategyType xlsxToJsonDef.StrategyType) string {
	sheetName := "link_" + name
	//log.Println("开始解析子表:", sheetName, "过滤条件为:", strlist)

	dataStructList, err := ReadField(xls, sheetName)
	if err != nil {
		sglog.Error("解析子表数据结构:%s 错误:%s", sheetName)
		log.Println(err)
		sgthread.DelayExit(2)
	}

	rows, err := xls.Rows(sheetName)
	if err != nil {
		sglog.Error("解析子表:%s错误:", sheetName)
		log.Println(err)
		sgthread.DelayExit(2)
	}

	fliter_str := strings.Split(strlist, ",")

	write_flieter_str := []string{}

	rawStr := "["

	firstInsert := true
	rowIndex := 1
	for rows.Next() {
		if rowIndex <= int(xlsxToJsonDef.TABLE_FORMAT_ROW_DATATYPE) {
			rowIndex++
			continue
		}
		tmpStr := "{"
		//每个row都是一个单独的数据
		row := rows.Columns()
		if len(row) <= 0 {
			break
		}

		colIndex := 1
		firstColumInsert := true
		ignore_row := false
		for _, colCell := range row {
			if colIndex <= int(xlsxToJsonDef.TABLE_FORMAT_COLUMN_DES) {
				colIndex++
				continue
			}
			if colIndex-int(xlsxToJsonDef.TABLE_FORMAT_COLUMN_DES) > len(dataStructList) {
				ignore_row = true
				break
			}

			if colIndex == int(xlsxToJsonDef.TABLE_FORMAT_COLUMN_DES)+1 {
				in_flieter := false
				for _, v := range fliter_str {
					if v == colCell {
						in_flieter = true
						break
					}
				}
				if in_flieter == false {
					ignore_row = true
					break
				}
				write_flieter_str = append(write_flieter_str, colCell)
			}
			dataStruct := dataStructList[colIndex-int(xlsxToJsonDef.TABLE_FORMAT_COLUM_REAL_DATA_INDEX)]

			if IsIgnoreField(strategyType, dataStruct.strategyType) {
				colIndex++
				continue
			}

			tmpColumnStr := GenColCell(xls, sheetName, dataStruct, colCell, strategyType)
			firstColumInsert, tmpStr = ConnectTwoString(firstColumInsert, tmpStr, tmpColumnStr)
			colIndex++
		}
		rowIndex++
		tmpStr += "}"

		if ignore_row {
			continue
		}
		firstInsert, rawStr = ConnectTwoString(firstInsert, rawStr, tmpStr)
	}

	rawStr += "]"

	if strlist != "" && (len(fliter_str) != len(write_flieter_str)) {
		sglog.Error("解析子表:%s 错误:声明的子项长度与子表搜索获得的长度不一致", sheetName)
		sglog.Error("主表子项列表:%s", fliter_str)
		sglog.Error("搜索所获得的的列表:%s", write_flieter_str)
		sglog.Error(err)
		sgthread.DelayExit(2)
	}

	//log.Println("解析子表:", sheetName, "过滤条件为:", strlist, " 成功")

	return rawStr
}

func ParseXlxs(xls *excelize.File, sheetName string, typeCell xlsxToJsonDef.TableType, dataStructList []xlsxToJsonDef.DataStruct, strategyType xlsxToJsonDef.StrategyType) string {
	log.Println("开始解析主表:%s", sheetName)
	rows, err := xls.Rows(sheetName)
	if err != nil {
		sglog.Error("解析主表:%s 错误:", sheetName)
		sglog.Error(err)
		sgthread.DelayExit(2)
	}

	rawStr := GetStrPrefixByTypeCell(typeCell)
	firstInsert := true
	rowIndex := 1
	for rows.Next() {
		if rowIndex <= int(xlsxToJsonDef.TABLE_FORMAT_ROW_DATATYPE) {
			rowIndex++
			continue
		}

		if xlsxToJsonDef.TableType_object == typeCell {
			if rowIndex > int(xlsxToJsonDef.TABLE_FORMAT_ROW_DATATYPE)+1 {
				break //obj格式只有第一行数据有效
			}
		}

		//数组格式每个row都是一个单独的数据
		tmpStr := GetInnerStrPrefixByTypeCell(typeCell)
		row := rows.Columns()
		if len(row) <= 0 {
			break
		}
		colIndex := 1
		firstColumInsert := true
		for _, colCell := range row {
			if colIndex <= int(xlsxToJsonDef.TABLE_FORMAT_COLUMN_DES) {
				colIndex++
				continue
			}
			if colIndex-int(xlsxToJsonDef.TABLE_FORMAT_COLUMN_DES) > len(dataStructList) {
				break
			}
			dataStruct := dataStructList[colIndex-int(xlsxToJsonDef.TABLE_FORMAT_COLUM_REAL_DATA_INDEX)]

			if IsIgnoreField(strategyType, dataStruct.strategyType) {
				colIndex++
				continue
			}

			tmpColumnStr := GenColCell(xls, sheetName, dataStruct, colCell, strategyType)
			firstColumInsert, tmpStr = ConnectTwoString(firstColumInsert, tmpStr, tmpColumnStr)
			colIndex++
		}
		rowIndex++
		tmpStr += GetInnerStrSuffixByTypeCell(typeCell)
		firstInsert, rawStr = ConnectTwoString(firstInsert, rawStr, tmpStr)
	}
	rawStr += GetStrSuffixByTypeCell(typeCell)
	return rawStr
}

func WriteConfigFile(js string, dir string, filename string, desc string) {
	// check
	if _, err := os.Stat(dir); err == nil {
		//log.Println("文件夹", dir, "已存在,无需创建")
	} else {
		log.Println("%s 文件夹 %s 不存在,开始创建文件夹", desc, dir)
		err := os.MkdirAll(dir, 0711)

		if err != nil {
			sglog.Error("创建 %s 文件夹 %s,失败", desc, dir)
			sgthread.DelayExit(2)
		}
	}

	// check again
	if _, err := os.Stat(dir); err != nil {
		log.Println("创建 %s 文件夹 %s 失败，请联系开发", desc, dir)
	}

	file, err := os.Create(dir + "/" + filename)
	if nil != err {
		sglog.Error("创建/打开 %s json文件 %s 失败,err=", desc, filename, err)
		sgthread.DelayExit(2)
	}

	_, err = io.WriteString(file, string(js))

	if nil != err {
		sglog.Error("写入 %s json文件 %s 失败,err=%s", desc, filename, err)
		sgthread.DelayExit(2)
	}
	log.Println("写入 %s json文件 %s 到路径 %s 完成!", desc, filename, dir)
}

func CheckJsonValid(str string) bool {
	var target interface{}
	if err := json.Unmarshal([]byte(str), &target); err != nil {
		sglog.Error("解析json失败，不是有效的json格式，ex=%s", err)
		return false
	}
	return true
}

func CheckJsonValidAndWriteFile(str string, dir string, filename string, flag string) {
	log.Println("本次配置:%s 解析结果:", flag)
	log.Println(str)
	log.Println("本次 %s 解析完成,开始检测是否为合法Json格式", flag)

	var target interface{}
	if err := json.Unmarshal([]byte(str), &target); err != nil {
		sglog.Error("解析 %s json失败，不是有效的json格式，ex=%s", flag, err)
		sgthread.DelayExit(2)
	}

	log.Println("解析 %s json成功，开始写入配置文件", flag)
	js, _ := json.MarshalIndent(&target, "", "  ")

	WriteConfigFile(string(js), dir, filename, flag)
}

func TransfromInterfaceTolua(result interface{}, firstRun bool) string {
	transformdata, ok := result.(map[string]interface{})
	rawStr := ""
	if ok {
		rawStr += "{"
		firstInsert := true

		sortkeys := []string{}

		for k, _ := range transformdata {
			sortkeys = append(sortkeys, k)
		}

		sort.Strings(sortkeys)

		for _, k := range sortkeys {
			tmpStr := k + "="
			v := transformdata[k]
			switch vtype := v.(type) {
			case string:
				value, ok := v.(string)
				if !ok {
					sglog.Error("lua数组解析失败,数据转换失败,string,k=%s,v=%s ,vtype=%s", k, v, vtype)
					sgthread.DelayExit(2)
				}
				tmpStr += "\"" + value + "\""
			case bool:
				value, ok := v.(bool)
				if !ok {
					sglog.Error("lua数组解析失败,数据转换失败,bool,k=%s,v=%s", k, v)
					sgthread.DelayExit(2)
				}
				tmpStr += strconv.FormatBool(value)
			case float64:
				value, ok := v.(float64)
				if !ok {
					sglog.Error("lua数组解析失败,数据转换失败,float64,k=%s,v=%s", k, v)
					sgthread.DelayExit(2)
				}
				tmpStr += strconv.FormatFloat(value, 'f', 10, 64)
			case int64:
				value, ok := v.(int64)
				if !ok {
					sglog.Error("lua数组解析失败,数据转换失败,int64,k=%s,v=%s", k, v)
					sgthread.DelayExit(2)
				}
				tmpStr += strconv.FormatInt(value, 10)
			case []interface{}:
				tmpStr += TransfromInterfaceTolua(v, false)
			case map[string]interface{}:
				tmpStr += TransfromInterfaceTolua(v, false)
			case json.Number:
				vex, err := v.(json.Number).Int64()
				if err != nil {
					vex, err := v.(json.Number).Float64()
					if err != nil {
						sglog.Error("lua数组解析失败,数据转换失败,unknow,k=%s,v=%s", k, v)
						sgthread.DelayExit(2)
					} else {
						tmpStr += strconv.FormatFloat(vex, 'f', -1, 64)
					}
				} else {
					tmpStr += strconv.FormatInt(vex, 10)
				}
			default:
				sglog.Error("lua数组解析失败,数据转换失败,unknow,k=%s,v=%s", k, v)
				sgthread.DelayExit(2)
			}

			if firstInsert {
				rawStr += tmpStr
				firstInsert = false
			} else {
				rawStr += ", " + tmpStr
			}
		}
		rawStr += "}"
	} else {
		transformdata, ok := result.([]interface{})
		if ok {
			rawStr += "{"
			if firstRun {
				rawStr += "\n"
			}
			firstInsert := true
			for k, v := range transformdata {
				tmpStr := ""
				switch vtype := v.(type) {
				case string:
					value, ok := v.(string)
					if !ok {
						sglog.Error("lua数组解析失败,数据转换失败,string,k=%s,v=%s ,vtype=%s", k, v, vtype)
						sgthread.DelayExit(2)
					}
					tmpStr = "\"" + value + "\""
				case bool:
					value, ok := v.(bool)
					if !ok {
						sglog.Error("lua数组解析失败,数据转换失败,bool,,k=%s,v=%s", k, v)
						sgthread.DelayExit(2)
					}
					tmpStr = strconv.FormatBool(value)
				case float64:
					sglog.Error("lua数组解析失败,数组内容不应该为float64,,k=%s,v=%s", k, v)
					sgthread.DelayExit(2)
				case int64:
					log.Println("lua数组解析失败,数组内容不应该为int64,,k=%s,v=%s", k, v)
					sgthread.DelayExit(2)
				case []interface{}:
					tmpStr += TransfromInterfaceTolua(v, false)
				case map[string]interface{}:
					tmpStr += TransfromInterfaceTolua(v, false)
				case json.Number:
					tmpStr = ""
					vex, err := v.(json.Number).Int64()
					if err != nil {
						vex, err := v.(json.Number).Float64()
						if err != nil {
							sglog.Error("lua数组解析失败,数据转换失败,222unknow,,k=%s,v=%s", k, v)
							sgthread.DelayExit(2)
						} else {
							tmpStr += strconv.FormatFloat(vex, 'f', -1, 64)
						}
					} else {
						tmpStr += strconv.FormatInt(vex, 10)
					}
				default:
					sglog.Error("lua数组解析失败,数据转换失败,unknow,,k=%s,v=%s", k, v)
					sgthread.DelayExit(2)
				}

				if firstInsert {
					rawStr += tmpStr
					firstInsert = false
				} else {
					if firstRun {
						rawStr += ",\n" + tmpStr
					} else {
						rawStr += "," + tmpStr
					}
				}
			}
			if firstRun {
				rawStr += "\n"
			}
			rawStr += "}"
		}
	}
	return rawStr
}

func TransformJsonTolua(str string, typeCell xlsxToJsonDef.TableType) string {
	luaStr := "return "
	decoder := json.NewDecoder(bytes.NewBufferString(str))
	decoder.UseNumber()
	if xlsxToJsonDef.TableType_array == typeCell {
		var result []interface{}
		if err := decoder.Decode(&result); err != nil {
			sglog.Error("TransformJsonTolua json解析失败,str=%s \nerr=", str, err)
			sgthread.DelayExit(2)
		}

		luaStr += TransfromInterfaceTolua(result, true)
	} else {
		var result map[string]interface{}
		if err := decoder.Decode(&result); err != nil {
			sglog.Error("TransformJsonTolua json解析失败,str=%s \nerr=", str, err)
			sgthread.DelayExit(2)
		}
		luaStr += TransfromInterfaceTolua(result, true)
	}

	return luaStr
}
