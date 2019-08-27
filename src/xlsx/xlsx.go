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
	xlsxToJsonDef "xlsxToJson/src/def"

	"github.com/coderguang/GameEngine_go/sgthread"

	"github.com/coderguang/GameEngine_go/sglog"

	"github.com/360EntSecGroup-Skylar/excelize"
)

func StartGenFile(fileName string) {

	sglog.Info("开始执行配置表 %s 解析到json文件\n", fileName)

	xls, err := excelize.OpenFile(fileName)

	if err != nil {
		sglog.Error("读取文件报错,%s", err)
		sgthread.DelayExit(2)
	}
	StartGetRoot(xls)

	sglog.Info("配置表 %s 全部解析完成!\n=============\n\n", fileName)
}

func StartGetRoot(xls *excelize.File) {
	configList, _ := ReadRootField(xls)

	if len(configList) <= 0 {
		sglog.Error("root 表未配置任何数据")
		sgthread.DelayExit(2)
	}

	for k := range configList {
		sglog.Info("开始执行生成:%s", configList[k].Name)
		StartGenConfig(xls, &configList[k])
		sglog.Info("执行生成 %s 完成\n==================================================================\n\n\n", configList[k].Name)
	}
}

func StartGenConfig(xls *excelize.File, config *xlsxToJsonDef.RootDirStruct) {

	if "" == config.Name {
		sglog.Error("主表为空")
		sgthread.DelayExit(2)
	}

	sglog.Info("主表sheet名称为:%s", config.Name)

	datalist, err := ReadField(xls, config.Name)

	if err != nil {
		sglog.Error("读取字段名称出错，ex=%s", err)
		sgthread.DelayExit(2)
	}

	typeCell, err := GetTypeCellAndCheck(xls, config.Name)

	if err != nil {
		sglog.Error("获取生成配置类型错误，ex=%s", err)
		sgthread.DelayExit(2)
	}

	fileName := config.Name + ".json"
	serverStr := ParseXlxs(xls, config.Name, typeCell, datalist, xlsxToJsonDef.StrategyType_Server)
	CheckJsonValidAndWriteFile(serverStr, config.ServerDir, fileName, "服务器")

	clientStr := ParseXlxs(xls, config.Name, typeCell, datalist, xlsxToJsonDef.StrategyType_Client)
	//CheckJsonValidAndWriteFile(clientStr, config.ClientDir, fileName, "客户端json")
	//CheckJsonValid(clientStr)

	if CheckJsonValid(clientStr) {
		luafileName := config.Name + ".lua"
		clientluaStr := TransformJsonTolua(clientStr, typeCell)
		WriteConfigFile(clientluaStr, config.ClientDir, luafileName, "客户端lua")
	} else {
		sglog.Error("生成lua配置错误，ex=%s", err)
		sgthread.DelayExit(2)
	}
}

func ReadRootField(xls *excelize.File) ([]xlsxToJsonDef.RootDirStruct, error) {

	dataStructList := []xlsxToJsonDef.RootDirStruct{}

	rows, err := xls.Rows(xlsxToJsonDef.TABLE_ROOT_SHEET_Name)
	if err != nil {
		sglog.Error("读取 %s 错误,err=%s", xlsxToJsonDef.TABLE_ROOT_SHEET_Name, err)
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
				sglog.Error("root 表字段名称配置长度不一致 ServerDir rowIndex=%d,colIndex=%d,名称:%s", rowIndex, colIndex, colCell)
				sgthread.DelayExit(2)
			}

			if int(xlsxToJsonDef.TABLE_ROOT_COLUMN_CONFIG_Name) == colIndex {
				tmp := xlsxToJsonDef.NewRootDirStruct()
				tmp.Name = colCell
				dataStructList = append(dataStructList, tmp)
			} else if int(xlsxToJsonDef.TABLE_ROOT_COLUMN_CONFIG_SERVER_DIR) == colIndex {
				dataStructList[dataIndex].ServerDir = colCell
			} else if int(xlsxToJsonDef.TABLE_ROOT_COLUMN_CONFIG_CLIENT_DIR) == colIndex {
				dataStructList[dataIndex].ClientDir = colCell
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
		sglog.Error("读取配置类型 %s 数据报错,type=%d", xlsxToJsonDef.TABLE_FORMAT_TYPECELL_POS, typeCellInt)
		sgthread.DelayExit(2)
	}
	typeCell := xlsxToJsonDef.TableType(typeCellInt)

	if xlsxToJsonDef.TableType_object != typeCell && xlsxToJsonDef.TableType_array != typeCell {
		sglog.Error("配置类型 A1 数据错误,既不是0（数组）也不是1（对象）,当前值为:%d", typeCell)
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
				sglog.Error("字段名称配置长度不一致 rowIndex=%d,colIndex=%d,名称:%s", rowIndex, colIndex, colCell)
				sgthread.DelayExit(2)
			}

			if int(xlsxToJsonDef.TABLE_FORMAT_ROW_DES) == rowIndex {
				tmp := xlsxToJsonDef.NewDataStruct()
				tmp.Desc = colCell
				dataStructList = append(dataStructList, tmp)
			} else if int(xlsxToJsonDef.TABLE_FORMAT_ROW_Name) == rowIndex {
				dataStructList[dataIndex].Name = colCell
			} else if int(xlsxToJsonDef.TABLE_FORMAT_ROW_CONFIG_STRATEGY) == rowIndex {
				StrategyTypeValue, err := strconv.Atoi(colCell)
				if err != nil {
					sglog.Error("策略类型错误,rowIndex=%d,类型:%s", rowIndex, colCell)
					sgthread.DelayExit(2)
				}
				if StrategyTypeValue != int(xlsxToJsonDef.StrategyType_All) && StrategyTypeValue != int(xlsxToJsonDef.StrategyType_Server) && StrategyTypeValue != int(xlsxToJsonDef.StrategyType_Client) {
					sglog.Error("策略类型错误,rowIndex=%d,类型:%d", rowIndex, StrategyTypeValue)
					sgthread.DelayExit(2)
				}
				dataStructList[dataIndex].StrategyTypeValue = xlsxToJsonDef.StrategyType(StrategyTypeValue)
			} else if int(xlsxToJsonDef.TABLE_FORMAT_ROW_DATATYPE) == rowIndex {
				DataTypeValue, err := strconv.Atoi(colCell)
				if err != nil {
					sglog.Error("数据类型错误,rowIndex=%d,类型:%s", rowIndex, colCell)
					sgthread.DelayExit(2)
				}
				if DataTypeValue != int(xlsxToJsonDef.DataType_raw) && DataTypeValue != int(xlsxToJsonDef.DataType_string) && DataTypeValue != int(xlsxToJsonDef.DataType_link) {
					sglog.Error("数据类型错误,rowIndex=%d,类型:%d", rowIndex, DataTypeValue)
					sgthread.DelayExit(2)
				}
				dataStructList[dataIndex].DataTypeValue = xlsxToJsonDef.DataType(DataTypeValue)
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

func GenColCell(xls *excelize.File, sheetName string, dataStruct *xlsxToJsonDef.DataStruct, colCell string, StrategyTypeValue xlsxToJsonDef.StrategyType) string {
	tmpColumnStr := "\"" + dataStruct.Name + "\":"
	switch dataStruct.DataTypeValue {
	case xlsxToJsonDef.DataType_raw:
		{
			if "" == colCell {
				sglog.Error("解析表:%s 错误:数据类型0不允许为空，列 %s", sheetName, dataStruct.Desc)
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
			tmpColumnStr += ParseChildXlxs(xls, dataStruct.Name, colCell, StrategyTypeValue)
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
	}
	return ""
}

func GetInnerStrSuffixByTypeCell(typeCell xlsxToJsonDef.TableType) string {
	if xlsxToJsonDef.TableType_array == typeCell {
		return "}"
	}
	return ""
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
	}
	return true

}

//解析子表，子表的子项必然是{}格式
func ParseChildXlxs(xls *excelize.File, Name string, strlist string, StrategyTypeValue xlsxToJsonDef.StrategyType) string {
	sheetName := "link_" + Name
	//log.Println("开始解析子表:", sheetName, "过滤条件为:", strlist)

	dataStructList, err := ReadField(xls, sheetName)
	if err != nil {
		sglog.Error("解析子表数据结构:%s 错误:%s", sheetName, err)
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

			if IsIgnoreField(StrategyTypeValue, dataStruct.StrategyTypeValue) {
				colIndex++
				continue
			}

			tmpColumnStr := GenColCell(xls, sheetName, &dataStruct, colCell, StrategyTypeValue)
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
		sglog.Error("err=%s", err)
		sgthread.DelayExit(2)
	}

	//log.Println("解析子表:", sheetName, "过滤条件为:", strlist, " 成功")

	return rawStr
}

func ParseXlxs(xls *excelize.File, sheetName string, typeCell xlsxToJsonDef.TableType, dataStructList []xlsxToJsonDef.DataStruct, StrategyTypeValue xlsxToJsonDef.StrategyType) string {
	sglog.Info("开始解析主表:%s", sheetName)
	rows, err := xls.Rows(sheetName)
	if err != nil {
		sglog.Error("解析主表:%s 错误:", sheetName)
		sglog.Error("err:%s", err)
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

			if IsIgnoreField(StrategyTypeValue, dataStruct.StrategyTypeValue) {
				colIndex++
				continue
			}

			tmpColumnStr := GenColCell(xls, sheetName, &dataStruct, colCell, StrategyTypeValue)
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

func WriteConfigFile(js string, dir string, fileName string, Desc string) {
	// check
	if _, err := os.Stat(dir); err == nil {
		//log.Println("文件夹", dir, "已存在,无需创建")
	} else {
		sglog.Info("%s 文件夹 %s 不存在,开始创建文件夹", Desc, dir)
		err := os.MkdirAll(dir, 0711)

		if err != nil {
			sglog.Error("创建 %s 文件夹 %s,失败", Desc, dir)
			sgthread.DelayExit(2)
		}
	}

	// check again
	if _, err := os.Stat(dir); err != nil {
		sglog.Error("创建 %s 文件夹 %s 失败，请联系开发", Desc, dir)
	}

	file, err := os.Create(dir + "/" + fileName)
	if nil != err {
		sglog.Error("创建/打开 %s json文件 %s 失败,err=%s", Desc, fileName, err)
		sgthread.DelayExit(2)
	}

	_, err = io.WriteString(file, string(js))

	if nil != err {
		sglog.Error("写入 %s json文件 %s 失败,err=%s", Desc, fileName, err)
		sgthread.DelayExit(2)
	}
	sglog.Info("写入 %s json文件 %s 到路径 %s 完成!", Desc, fileName, dir)
}

func CheckJsonValid(str string) bool {
	var target interface{}
	if err := json.Unmarshal([]byte(str), &target); err != nil {
		sglog.Error("解析json失败，不是有效的json格式，ex=%s", err)
		return false
	}
	return true
}

func CheckJsonValidAndWriteFile(str string, dir string, fileName string, flag string) {
	sglog.Info("本次配置:%s 解析结果:", flag)
	sglog.Info(str)
	sglog.Info("本次 %s 解析完成,开始检测是否为合法Json格式", flag)

	var target interface{}
	if err := json.Unmarshal([]byte(str), &target); err != nil {
		sglog.Error("解析 %s json失败，不是有效的json格式，ex=%s", flag, err)
		sgthread.DelayExit(2)
	}

	sglog.Info("解析 %s json成功，开始写入配置文件", flag)
	js, _ := json.MarshalIndent(&target, "", "  ")

	WriteConfigFile(string(js), dir, fileName, flag)
}

func TransfromInterfaceTolua(result interface{}, firstRun bool) string {
	transformdata, ok := result.(map[string]interface{})
	rawStr := ""
	if ok {
		rawStr += "{"
		firstInsert := true

		sortkeys := []string{}

		for k := range transformdata {
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
						sglog.Error("lua数组解析失败,数据转换失败,string,k=%d,v=%s ,vtype=%s", k, v, vtype)
						sgthread.DelayExit(2)
					}
					tmpStr = "\"" + value + "\""
				case bool:
					value, ok := v.(bool)
					if !ok {
						sglog.Error("lua数组解析失败,数据转换失败,bool,,k=%d,v=%s", k, v)
						sgthread.DelayExit(2)
					}
					tmpStr = strconv.FormatBool(value)
				case float64:
					sglog.Error("lua数组解析失败,数组内容不应该为float64,,k=%d,v=%s", k, v)
					sgthread.DelayExit(2)
				case int64:
					sglog.Error("lua数组解析失败,数组内容不应该为int64,,k=%d,v=%s", k, v)
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
							sglog.Error("lua数组解析失败,数据转换失败,222unknow,,k=%d,v=%s", k, v)
							sgthread.DelayExit(2)
						} else {
							tmpStr += strconv.FormatFloat(vex, 'f', -1, 64)
						}
					} else {
						tmpStr += strconv.FormatInt(vex, 10)
					}
				default:
					sglog.Error("lua数组解析失败,数据转换失败,unknow,,k=%d,v=%s", k, v)
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
			sglog.Error("TransformJsonTolua json解析失败,str=%s \nerr=%s", str, err)
			sgthread.DelayExit(2)
		}

		luaStr += TransfromInterfaceTolua(result, true)
	} else {
		var result map[string]interface{}
		if err := decoder.Decode(&result); err != nil {
			sglog.Error("TransformJsonTolua json解析失败,str=%s \nerr=%s", str, err)
			sgthread.DelayExit(2)
		}
		luaStr += TransfromInterfaceTolua(result, true)
	}

	return luaStr
}
