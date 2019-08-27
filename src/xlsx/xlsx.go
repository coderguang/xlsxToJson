package xlsxToJsonXlsx

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/Luxurioust/excelize"
	"github.com/coderguang/GameEngine_go/sglog"
)

func GetTypeCellAndCheck(xls *excelize.File, sheetName string) (xlsxToJsonDef.TableType, error) {

	typeCellStr, err := xls.GetCellValue(sheetName, xlsxToJsonDef.TABLE_FORMAT_TYPECELL_POS)

	if err != nil {
		sglog.Info("读取配置类型 %s 数据报错,str=%s", xlsxToJsonDef.TABLE_FORMAT_TYPECELL_POS, typeCellStr)
		sglog.Info(err)
		os.Exit(1)
	}

	typeCellInt, err := strconv.Atoi(typeCellStr)
	if err != nil {
		sglog.Info("读取配置类型 ", xlsxToJsonDef.TABLE_FORMAT_TYPECELL_POS, " 数据报错,type=", typeCellInt)
		sglog.Info(err)
		os.Exit(1)
	}
	typeCell := xlsxToJsonDef.TableType(typeCellInt)

	if typeCell >= xlsxToJsonDef.TableType_end {
		sglog.Info("配置类型 A1 数据错误,当前值为:", typeCell)
		os.Exit(1)
	}

	if xlsxToJsonDef.TableType_object == typeCell {
		sglog.Info("该表生成的数据结构为对象")
	} else if xlsxToJsonDef.TableType_array == typeCell {
		sglog.Info("该表生成的数据结果为数组")
	} else if xlsxToJsonDef.TableType_file_list == typeCell {
		sglog.Info("该表生成多个对象文件")
	}

	return typeCell, err
}

func ReadField(xls *excelize.File, sheetName string) ([]xlsxToJsonDef.DataStruct, error) {
	dataStructList := []xlsxToJsonDef.DataStruct{}

	rows, err := xls.Rows(sheetName)
	rowIndex := 1
	for rows.Next() {
		if rowIndex > int(TABLE_FORMAT_ROW_DATATYPE) {
			break
		}
		row, _ := rows.Columns()
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
				sglog.Info("字段名称配置长度不一致 rowIndex=", rowIndex, ",colIndex=", colIndex, ",名称:", colCell)
				os.Exit(1)
			}

			if int(xlsxToJsonDef.TABLE_FORMAT_ROW_DES) == rowIndex {
				tmp := NewDataStruct()
				tmp.Desc = colCell
				dataStructList = append(dataStructList, tmp)
			} else if int(TABLE_FORMAT_ROW_NAME) == rowIndex {
				dataStructList[dataIndex].Name = colCell
			} else if int(xlsxToJsonDef.TABLE_FORMAT_ROW_CONFIG_STRATEGY) == rowIndex {
				strategyType, err := strconv.Atoi(colCell)
				if err != nil {
					sglog.Info("策略类型错误,rowIndex=", rowIndex, ",类型:", colCell)
					os.Exit(1)
				}
				if strategyType != int(xlsxToJsonDef.StrategyType_All) && strategyType != int(xlsxToJsonDef.StrategyType_Server) && strategyType != int(xlsxToJsonDef.StrategyType_Client) && strategyType != int(xlsxToJsonDef.StrategyType_NoGen) {
					sglog.Info("策略类型错误,rowIndex=", rowIndex, ",类型:", strategyType)
					os.Exit(1)
				}
				dataStructList[dataIndex].strategyType = xlsxToJsonDef.StrategyType(strategyType)
			} else if int(TABLE_FORMAT_ROW_DATATYPE) == rowIndex {
				dataType, err := strconv.Atoi(colCell)
				if err != nil {
					sglog.Info("数据类型错误,rowIndex=", rowIndex, ",类型:", colCell)
					os.Exit(1)
				}
				if dataType != int(xlsxToJsonDef.DataType_raw) && dataType != int(xlsxToJsonDef.DataType_string) && dataType != int(xlsxToJsonDef.DataType_link) {
					sglog.Info("数据类型错误,rowIndex=", rowIndex, ",类型:", dataType)
					os.Exit(1)
				}
				dataStructList[dataIndex].dataType = xlsxToJsonDef.DataType(dataType)
			}
			colIndex++
		}
		rowIndex++
	}

	if len(dataStructList) <= 0 {
		sglog.Info("数据表为空,请检测:", sheetName)
		os.Exit(1)
	}

	for _, v := range dataStructList {
		if v.CheckEmpty() {
			sglog.Info("解析表数据结构配置错误,配置为空:表名称", sheetName)
			v.Show()
			os.Exit(1)
		}
	}

	return dataStructList, err
}

func GenColCell(xls *excelize.File, sheetName string, dataStruct xlsxToJsonDef.DataStruct, colCell string, strategyType xlsxToJsonDef.StrategyType) string {
	tmpColumnStr := "\"" + dataStruct.Name + "\":"
	switch dataStruct.dataType {
	case xlsxToJsonDef.DataType_raw:
		{
			if "" == colCell {
				sglog.Info("解析表:", sheetName, "错误:数据类型0不允许为空，列", dataStruct.Desc)
				os.Exit(1)
			}
			tmpColumnStr += colCell
		}
	case xlsxToJsonDef.DataType_string:
		{
			tmpColumnStr += "\"" + colCell + "\""
		}
	case xlsxToJsonDef.DataType_link:
		{
			tmpColumnStr += ParseChildXlxs(xls, dataStruct.Name, colCell, strategyType)
		}
	}
	return tmpColumnStr
}

func GetStrPrefixByTypeCell(typeCell xlsxToJsonDef.TableType) string {
	if xlsxToJsonDef.TableType_array == typeCell {
		return "["
	} else if xlsxToJsonDef.TableType_object == typeCell || xlsxToJsonDef.TableType_file_list == typeCell {
		return "{"
	} else {
		return ""
	}
}

func GetStrSuffixByTypeCell(typeCell xlsxToJsonDef.TableType) string {
	if xlsxToJsonDef.TableType_array == typeCell {
		return "]"
	} else if xlsxToJsonDef.TableType_object == typeCell || xlsxToJsonDef.TableType_file_list == typeCell {
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
	if configStrategy == xlsxToJsonDef.StrategyType_NoGen {
		return true
	}

	if writeFileType == configStrategy {
		return false
	} else {
		return true
	}
}

//解析子表，子表的子项必然是{}格式
func ParseChildXlxs(xls *excelize.File, Name string, strlist string, strategyType xlsxToJsonDef.StrategyType) string {
	sheetName := "link_" + Name
	//sglog.Info("开始解析子表:", sheetName, "过滤条件为:", strlist)

	dataStructList, err := ReadField(xls, sheetName)
	if err != nil {
		sglog.Info("解析子表数据结构:", sheetName, "错误:")
		sglog.Info(err)
		os.Exit(1)
	}

	rows, err := xls.Rows(sheetName)
	if err != nil {
		sglog.Info("解析子表:", sheetName, "错误:")
		sglog.Info(err)
		os.Exit(1)
	}

	fliter_str := strings.Split(strlist, ",")

	write_flieter_str := []string{}

	rawStr := "["

	firstInsert := true
	rowIndex := 1
	for rows.Next() {
		if rowIndex <= int(TABLE_FORMAT_ROW_DATATYPE) {
			rowIndex++
			continue
		}
		tmpStr := "{"
		//每个row都是一个单独的数据
		row, _ := rows.Columns()
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
			if colIndex-int(xlsxToJsonDef.TABLE_FORMAT_COLUMN_DES) == len(dataStructList) {
				break
			}

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
		sglog.Info("解析子表:", sheetName, "错误:声明的子项长度与子表搜索获得的长度不一致")
		sglog.Info("主表子项列表:", fliter_str)
		sglog.Info("搜索所获得的的列表:", write_flieter_str)
		sglog.Info(err)
		os.Exit(1)
	}

	//sglog.Info("解析子表:", sheetName, "过滤条件为:", strlist, " 成功")

	return rawStr
}

func ParseXlxs(xls *excelize.File, config RootDirStruct, typeCell xlsxToJsonDef.TableType, dataStructList []xlsxToJsonDef.DataStruct, strategyType xlsxToJsonDef.StrategyType) string {
	sheetName := config.Name
	sglog.Info("开始解析主表:", sheetName)
	rows, err := xls.Rows(sheetName)
	if err != nil {
		sglog.Info("解析主表:", sheetName, "错误:")
		sglog.Info(err)
		os.Exit(1)
	}

	rawStr := GetStrPrefixByTypeCell(typeCell)
	firstInsert := true
	rowIndex := 1
	for rows.Next() {
		if rowIndex <= int(TABLE_FORMAT_ROW_DATATYPE) {
			rowIndex++
			continue
		}

		if xlsxToJsonDef.TableType_object == typeCell {
			if rowIndex > int(TABLE_FORMAT_ROW_DATATYPE)+1 {
				break //obj格式只有第一行数据有效
			}
		}

		//数组格式每个row都是一个单独的数据
		tmpStr := GetInnerStrPrefixByTypeCell(typeCell)
		row, _ := rows.Columns()
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
		if typeCell == xlsxToJsonDef.TableType_file_list {
			rawStr += GetStrSuffixByTypeCell(typeCell)
			flag := "服务器"
			if strategyType == xlsxToJsonDef.StrategyType_Client {
				flag = "客户端json"
			}
			CheckJsonValidAndWriteFile(rawStr, config, typeCell, flag, row[1])
			rawStr = GetStrPrefixByTypeCell(typeCell)
			firstInsert = true
			continue
		}
	}
	if typeCell == xlsxToJsonDef.TableType_file_list {
		return ""
	}
	rawStr += GetStrSuffixByTypeCell(typeCell)
	return rawStr
}

func WriteConfigFile(js string, dir string, filename string, Desc string) {
	// check
	if _, err := os.Stat(dir); err == nil {
		//sglog.Info("文件夹", dir, "已存在,无需创建")
	} else {
		sglog.Info(Desc, "文件夹", dir, "，不存在,开始创建文件夹")
		err := os.MkdirAll(dir, 0711)

		if err != nil {
			sglog.Info("创建", Desc, "件夹", dir, ",失败")
			sglog.Info(err)
			os.Exit(1)
		}
	}

	// check again
	if _, err := os.Stat(dir); err != nil {
		sglog.Info("创建", Desc, "文件夹", dir, "失败，请联系开发")
	}

	file, err := os.Create(dir + "/" + filename)
	if nil != err {
		sglog.Info("创建/打开", Desc, "json文件 ", filename, " 失败,err=", err)
		os.Exit(1)
	}

	_, err = io.WriteString(file, string(js))

	if nil != err {
		sglog.Info("写入", Desc, "json文件 ", filename, " 失败,err=", err)
		os.Exit(1)
	}
	sglog.Info("写入", Desc, "json文件 ", filename, "到路径", dir, " 完成!")
}

func CheckJsonValid(str string) bool {
	var target interface{}
	if err := json.Unmarshal([]byte(str), &target); err != nil {
		sglog.Info("解析json失败，不是有效的json格式，ex=", err)
		return false
	}
	return true
}

func CheckJsonValidAndWriteFile(str string, config RootDirStruct, typeCell xlsxToJsonDef.TableType, flag string, file_id string) {

	// sglog.Info("本次配置", flag, "解析结果:")
	// sglog.Info(str)
	sglog.Info("本次", flag, "解析完成,开始检测是否为合法Json格式")

	var target interface{}
	if err := json.Unmarshal([]byte(str), &target); err != nil {
		sglog.Info("解析", flag, "json失败，不是有效的json格式，ex=", err)
		os.Exit(1)
	}

	filename := config.Name + ".json"
	if typeCell == xlsxToJsonDef.TableType_file_list {
		filename = config.Name + file_id + ".json"
	}
	sglog.Info("解析", flag, "json成功，开始写入配置文件")
	js, _ := json.MarshalIndent(&target, "", "  ")

	if flag == "服务器" {
		WriteConfigFile(string(js), config.ServerDir, filename, flag)
		svr_build_file_sum++
	} else {
		// WriteConfigFile(string(js), config.ClientDir, filename, flag)
		// cli_build_file_sum++

		if CheckJsonValid(str) {
			clientluaStr := TransformJsonTolua(str, typeCell)
			luafilename := config.Name + ".lua"
			if typeCell == xlsxToJsonDef.TableType_file_list {
				luafilename += file_id + ".lua"
			}
			WriteConfigFile(clientluaStr, config.ClientDir, luafilename, "客户端lua")
			cli_build_file_sum++
		} else {
			sglog.Info("生成lua配置错误，ex=")
			os.Exit(1)
		}
	}

	// if flag == "服务器" {
	// } else {
	// 	if CheckJsonValid(str) {
	// 		clientluaStr := TransformJsonTolua(str, typeCell)
	// 		luafilename := config.Name + ".lua"
	// 		if typeCell == xlsxToJsonDef.TableType_file_list {
	// 			luafilename += file_id + ".lua"
	// 		}
	// 		WriteConfigFile(clientluaStr, config.ClientDir, luafilename, "客户端lua")
	// 		cli_build_file_sum++
	// 	} else {
	// 		sglog.Info("生成lua配置错误，ex=")
	// 		os.Exit(1)
	// 	}
	// }

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
					sglog.Info("lua数组解析失败,数据转换失败,string,k=", k, ",v=", v, "vtype=", vtype)
					os.Exit(1)
				}
				tmpStr += "\"" + value + "\""
			case bool:
				value, ok := v.(bool)
				if !ok {
					sglog.Info("lua数组解析失败,数据转换失败,bool,k=", k, ",v=", v)
					os.Exit(1)
				}
				tmpStr += strconv.FormatBool(value)
			case float64:
				value, ok := v.(float64)
				if !ok {
					sglog.Info("lua数组解析失败,数据转换失败,float64,k=", k, ",v=", v)
					os.Exit(1)
				}
				tmpStr += strconv.FormatFloat(value, 'f', 10, 64)
			case int64:
				value, ok := v.(int64)
				if !ok {
					sglog.Info("lua数组解析失败,数据转换失败,int64,k=", k, ",v=", v)
					os.Exit(1)
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
						sglog.Info("lua数组解析失败,数据转换失败,unknow,k=", k, ",v=", v)
						os.Exit(1)
					} else {
						tmpStr += strconv.FormatFloat(vex, 'f', -1, 64)
					}
				} else {
					tmpStr += strconv.FormatInt(vex, 10)
				}
			default:
				sglog.Info("lua数组解析失败,数据转换失败,unknow,k=", k, ",v=", v)
				os.Exit(1)
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
						sglog.Info("lua数组解析失败,数据转换失败,string,k=", k, ",v=", v, "vtype=", vtype)
						os.Exit(1)
					}
					tmpStr = "\"" + value + "\""
				case bool:
					value, ok := v.(bool)
					if !ok {
						sglog.Info("lua数组解析失败,数据转换失败,bool,k=", k, ",v=", v)
						os.Exit(1)
					}
					tmpStr = strconv.FormatBool(value)
				case float64:
					sglog.Info("lua数组解析失败,数组内容不应该为float64,k=", k, ",v=", v)
					os.Exit(1)
				case int64:
					sglog.Info("lua数组解析失败,数组内容不应该为int64,k=", k, ",v=", v)
					os.Exit(1)
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
							sglog.Info("lua数组解析失败,数据转换失败,222unknow,k=", k, ",v=", v)
							os.Exit(1)
						} else {
							tmpStr += strconv.FormatFloat(vex, 'f', -1, 64)
						}
					} else {
						tmpStr += strconv.FormatInt(vex, 10)
					}
				default:
					sglog.Info("lua数组解析失败,数据转换失败,unknow,k=", k, ",v=", v)
					os.Exit(1)
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
			sglog.Info("TransformJsonTolua json解析失败,str=", str, "\nerr=", err)
			os.Exit(1)
		}

		luaStr += TransfromInterfaceTolua(result, true)
	} else {
		var result map[string]interface{}
		if err := decoder.Decode(&result); err != nil {
			sglog.Info("TransformJsonTolua json解析失败,str=", str, "\nerr=", err)
			os.Exit(1)
		}
		luaStr += TransfromInterfaceTolua(result, true)
	}

	return luaStr
}

func StartGenConfig(xls *excelize.File, config RootDirStruct) {

	if "" == config.Name {
		sglog.Info("主表为空")
		os.Exit(1)
	}

	sglog.Info("主表sheet名称为:", config.Name)

	datalist, err := ReadField(xls, config.Name)

	if err != nil {
		sglog.Info("读取字段名称出错，ex=", err)
		os.Exit(1)
	}

	typeCell, err := GetTypeCellAndCheck(xls, config.Name)

	if err != nil {
		sglog.Info("获取生成配置类型错误，ex=", err)
		os.Exit(1)
	}

	serverStr := ParseXlxs(xls, config, typeCell, datalist, xlsxToJsonDef.StrategyType_Server)
	if typeCell != xlsxToJsonDef.TableType_file_list {
		CheckJsonValidAndWriteFile(serverStr, config, typeCell, "服务器", "")
	}

	clientStr := ParseXlxs(xls, config, typeCell, datalist, xlsxToJsonDef.StrategyType_Client)
	if typeCell != xlsxToJsonDef.TableType_file_list {
		CheckJsonValidAndWriteFile(clientStr, config, typeCell, "客户端json", "")
	}
}

func ReadRootField(xls *excelize.File) ([]RootDirStruct, error) {

	dataStructList := []RootDirStruct{}

	rows, err := xls.Rows(TABLE_ROOT_SHEET_NAME)
	if err != nil {
		sglog.Info("读取", TABLE_ROOT_SHEET_NAME, "错误,err=", err)
		os.Exit(1)
	}

	rowIndex := 1
	for rows.Next() {
		if rowIndex <= int(xlsxToJsonDef.TABLE_ROOT_ROW_DES) {
			rowIndex++
			continue
		}
		row, rowErr := rows.Columns()
		if rowErr != nil {
			sglog.Info("解析错误", TABLE_ROOT_SHEET_NAME, "错误,err=", rowErr)
			os.Exit(1)
		}

		colIndex := 1
		if len(row) <= 0 {
			break
		}
		isBreak := false
		for _, colCell := range row {
			if colIndex > int(xlsxToJsonDef.TABLE_ROOT_COLUMN_CONFIG_CLIENT_DIR) {
				break
			}
			if "" == colCell {
				isBreak = true
				break
			}
			dataIndex := rowIndex - int(xlsxToJsonDef.TABLE_ROOT_ROW_DES) - 1
			if dataIndex > len(dataStructList) {
				sglog.Info("root 表字段名称配置长度不一致 ServerDir rowIndex=", rowIndex, ",colIndex=", colIndex, ",名称:", colCell)
				os.Exit(1)
			}

			if int(TABLE_ROOT_COLUMN_CONFIG_NAME) == colIndex {
				tmp := newRootDirStruct()
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
		if isBreak {
			break
		}
	}

	if len(dataStructList) <= 0 {
		sglog.Info("root表为空")
		os.Exit(1)
	}

	for _, v := range dataStructList {
		if v.CheckEmpty() {
			sglog.Info("root表读取失败,有字段数据为空")
			v.Show()
			os.Exit(1)
		}
	}

	return dataStructList, err
}

func StartGetRoot(xls *excelize.File) {
	configList, _ := ReadRootField(xls)

	if len(configList) <= 0 {
		sglog.Info("root 表未配置任何数据")
		os.Exit(1)
	}

	for k, _ := range configList {
		sglog.Info("开始执行生成 ", configList[k].Name)
		StartGenConfig(xls, configList[k])
		sglog.Info("执行生成 ", configList[k].Name, " 完成\n==================================================================\n\n\n")
	}
}

func StartGenFile(filename string) {

	sglog.Info("开始执行配置表 ", filename, " 解析到json文件\n")

	xls, err := excelize.OpenFile(filename)

	if err != nil {
		sglog.Info("读取文件报错")
		sglog.Info(err)
		os.Exit(1)
	}
	StartGetRoot(xls)

	sglog.Info("配置表 ", filename, " 全部解析完成!\n=============\n\n")
}
