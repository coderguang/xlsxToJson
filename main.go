package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/coderguang/GameEngine_go"
)

type DataType int32

const (
	DataType_raw    DataType = 0
	DataType_string DataType = 1
	DataType_link   DataType = 2
)

type StrategyType int32

const (
	StrategyType_All    StrategyType = 0
	StrategyType_Server StrategyType = 1
	StrategyType_Client StrategyType = 2
)

type TableType int32

const (
	TableType_array  TableType = 0
	TableType_object TableType = 1
)

type DataStruct struct {
	desc         string
	name         string
	dataType     DataType
	strategyType StrategyType
}

func (v DataStruct) Show() {
	log.Println("表结构内容:desc:", v.desc, ",name:", v.name, ",dataType:", int(v.dataType), ",strategyType:", int(v.strategyType))
	sglog.Info("hi")
}

func (v DataStruct) CheckEmpty() bool {
	if "" == v.desc || "" == v.name || DataType(-1) == v.dataType || StrategyType(-1) == v.strategyType {
		return true
	}
	return false
}

func NewDataStruct() DataStruct {
	return DataStruct{
		desc:         "",
		name:         "",
		dataType:     DataType(-1),
		strategyType: StrategyType(-1),
	}
}

type RootDirStruct struct {
	serverDir string
	clientDir string
	name      string
}

func (v RootDirStruct) Show() {
	log.Println("root表内容:name:", v.name, ",clientDir:", v.clientDir, ",serverDir:", v.serverDir)
}

func (v RootDirStruct) CheckEmpty() bool {
	if "" == v.name || "" == v.serverDir || "" == v.clientDir {
		return true
	}
	return false
}

func newRootDirStruct() RootDirStruct {
	return RootDirStruct{
		serverDir: "",
		clientDir: "",
		name:      "",
	}
}

const TABLE_FORMAT_ROW_DES int = 1
const TABLE_FORMAT_ROW_NAME int = 2
const TABLE_FORMAT_ROW_CONFIG_STRATEGY int = 3
const TABLE_FORMAT_ROW_DATATYPE int = 4

const TABLE_FORMAT_COLUMN_DES int = 1
const TABLE_FORMAT_COLUM_REAL_DATA_INDEX int = 2

const TABLE_FORMAT_TYPECELL_POS string = "A1"

const TABLE_ROOT_SHEET_NAME string = "root"
const TABLE_ROOT_COLUMN_CONFIG_NAME int = 1
const TABLE_ROOT_COLUMN_CONFIG_SERVER_DIR int = 2
const TABLE_ROOT_COLUMN_CONFIG_CLIENT_DIR int = 3
const TABLE_ROOT_ROW_DES int = 1

func GetTypeCellAndCheck(xls *excelize.File, sheetName string) (TableType, error) {

	typeCellStr := xls.GetCellValue(sheetName, TABLE_FORMAT_TYPECELL_POS)

	typeCellInt, err := strconv.Atoi(typeCellStr)
	if err != nil {
		log.Println("读取配置类型 ", TABLE_FORMAT_TYPECELL_POS, " 数据报错,type=", typeCellInt)
		log.Println(err)
		os.Exit(1)
	}
	typeCell := TableType(typeCellInt)

	if TableType_object != typeCell && TableType_array != typeCell {
		log.Println("配置类型 A1 数据错误,既不是0（数组）也不是1（对象）,当前值为:", typeCell)
		os.Exit(1)
	}

	if TableType_object == typeCell {
		log.Println("该表生成的数据结构为对象")
	} else {
		log.Println("该表生成的数据结果为数组")
	}

	return typeCell, err
}

func ReadField(xls *excelize.File, sheetName string) ([]DataStruct, error) {
	dataStructList := []DataStruct{}

	rows, err := xls.Rows(sheetName)
	rowIndex := 1
	for rows.Next() {
		if rowIndex > int(TABLE_FORMAT_ROW_DATATYPE) {
			break
		}
		row := rows.Columns()
		if len(row) <= 0 {
			break
		}
		colIndex := 1
		for _, colCell := range row {
			if int(TABLE_FORMAT_COLUMN_DES) == colIndex {
				colIndex++
				continue
			}
			if "" == colCell {
				break
			}

			dataIndex := colIndex - int(TABLE_FORMAT_COLUMN_DES) - 1
			if dataIndex > len(dataStructList) {
				log.Println("字段名称配置长度不一致 rowIndex=", rowIndex, ",colIndex=", colIndex, ",名称:", colCell)
				os.Exit(1)
			}

			if int(TABLE_FORMAT_ROW_DES) == rowIndex {
				tmp := NewDataStruct()
				tmp.desc = colCell
				dataStructList = append(dataStructList, tmp)
			} else if int(TABLE_FORMAT_ROW_NAME) == rowIndex {
				dataStructList[dataIndex].name = colCell
			} else if int(TABLE_FORMAT_ROW_CONFIG_STRATEGY) == rowIndex {
				strategyType, err := strconv.Atoi(colCell)
				if err != nil {
					log.Println("策略类型错误,rowIndex=", rowIndex, ",类型:", colCell)
					os.Exit(1)
				}
				if strategyType != int(StrategyType_All) && strategyType != int(StrategyType_Server) && strategyType != int(StrategyType_Client) {
					log.Println("策略类型错误,rowIndex=", rowIndex, ",类型:", strategyType)
					os.Exit(1)
				}
				dataStructList[dataIndex].strategyType = StrategyType(strategyType)
			} else if int(TABLE_FORMAT_ROW_DATATYPE) == rowIndex {
				dataType, err := strconv.Atoi(colCell)
				if err != nil {
					log.Println("数据类型错误,rowIndex=", rowIndex, ",类型:", colCell)
					os.Exit(1)
				}
				if dataType != int(DataType_raw) && dataType != int(DataType_string) && dataType != int(DataType_link) {
					log.Println("数据类型错误,rowIndex=", rowIndex, ",类型:", dataType)
					os.Exit(1)
				}
				dataStructList[dataIndex].dataType = DataType(dataType)
			}
			colIndex++
		}
		rowIndex++
	}

	if len(dataStructList) <= 0 {
		log.Println("数据表为空,请检测:", sheetName)
		os.Exit(1)
	}

	for _, v := range dataStructList {
		if v.CheckEmpty() {
			log.Println("解析表数据结构配置错误,配置为空:表名称", sheetName)
			v.Show()
			os.Exit(1)
		}
	}

	return dataStructList, err
}

func GenColCell(xls *excelize.File, sheetName string, dataStruct DataStruct, colCell string, strategyType StrategyType) string {
	tmpColumnStr := "\"" + dataStruct.name + "\":"
	switch dataStruct.dataType {
	case DataType_raw:
		{
			if "" == colCell {
				log.Println("解析表:", sheetName, "错误:数据类型0不允许为空，列", dataStruct.desc)
				os.Exit(1)
			}
			tmpColumnStr += colCell
		}
	case DataType_string:
		{
			tmpColumnStr += "\"" + colCell + "\""
		}
	case DataType_link:
		{
			tmpColumnStr += ParseChildXlxs(xls, dataStruct.name, colCell, strategyType)
		}
	}
	return tmpColumnStr
}

func GetStrPrefixByTypeCell(typeCell TableType) string {
	if TableType_array == typeCell {
		return "["
	} else if TableType_object == typeCell {
		return "{"
	} else {
		return ""
	}
}

func GetStrSuffixByTypeCell(typeCell TableType) string {
	if TableType_array == typeCell {
		return "]"
	} else if TableType_object == typeCell {
		return "}"
	} else {
		return ""
	}
}

func GetInnerStrPrefixByTypeCell(typeCell TableType) string {
	if TableType_array == typeCell {
		return "{"
	} else {
		return ""
	}
}

func GetInnerStrSuffixByTypeCell(typeCell TableType) string {
	if TableType_array == typeCell {
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

func IsIgnoreField(writeFileType StrategyType, configStrategy StrategyType) bool {
	if configStrategy == StrategyType_All {
		return false
	}
	if writeFileType == configStrategy {
		return false
	} else {
		return true
	}
}

//解析子表，子表的子项必然是{}格式
func ParseChildXlxs(xls *excelize.File, name string, strlist string, strategyType StrategyType) string {
	sheetName := "link_" + name
	//log.Println("开始解析子表:", sheetName, "过滤条件为:", strlist)

	dataStructList, err := ReadField(xls, sheetName)
	if err != nil {
		log.Println("解析子表数据结构:", sheetName, "错误:")
		log.Println(err)
		os.Exit(1)
	}

	rows, err := xls.Rows(sheetName)
	if err != nil {
		log.Println("解析子表:", sheetName, "错误:")
		log.Println(err)
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
		row := rows.Columns()
		if len(row) <= 0 {
			break
		}

		colIndex := 1
		firstColumInsert := true
		ignore_row := false
		for _, colCell := range row {
			if colIndex <= int(TABLE_FORMAT_COLUMN_DES) {
				colIndex++
				continue
			}
			if colIndex-int(TABLE_FORMAT_COLUMN_DES) > len(dataStructList) {
				ignore_row = true
				break
			}

			if colIndex == int(TABLE_FORMAT_COLUMN_DES)+1 {
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
			dataStruct := dataStructList[colIndex-int(TABLE_FORMAT_COLUM_REAL_DATA_INDEX)]

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
		log.Println("解析子表:", sheetName, "错误:声明的子项长度与子表搜索获得的长度不一致")
		log.Println("主表子项列表:", fliter_str)
		log.Println("搜索所获得的的列表:", write_flieter_str)
		log.Println(err)
		os.Exit(1)
	}

	//log.Println("解析子表:", sheetName, "过滤条件为:", strlist, " 成功")

	return rawStr
}

func ParseXlxs(xls *excelize.File, sheetName string, typeCell TableType, dataStructList []DataStruct, strategyType StrategyType) string {
	log.Println("开始解析主表:", sheetName)
	rows, err := xls.Rows(sheetName)
	if err != nil {
		log.Println("解析主表:", sheetName, "错误:")
		log.Println(err)
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

		if TableType_object == typeCell {
			if rowIndex > int(TABLE_FORMAT_ROW_DATATYPE)+1 {
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
			if colIndex <= int(TABLE_FORMAT_COLUMN_DES) {
				colIndex++
				continue
			}
			if colIndex-int(TABLE_FORMAT_COLUMN_DES) > len(dataStructList) {
				break
			}
			dataStruct := dataStructList[colIndex-int(TABLE_FORMAT_COLUM_REAL_DATA_INDEX)]

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
		log.Println(desc, "文件夹", dir, "，不存在,开始创建文件夹")
		err := os.MkdirAll(dir, 0711)

		if err != nil {
			log.Println("创建", desc, "件夹", dir, ",失败")
			log.Println(err)
			os.Exit(1)
		}
	}

	// check again
	if _, err := os.Stat(dir); err != nil {
		log.Println("创建", desc, "文件夹", dir, "失败，请联系开发")
	}

	file, err := os.Create(dir + "/" + filename)
	if nil != err {
		log.Println("创建/打开", desc, "json文件 ", filename, " 失败,err=", err)
		os.Exit(1)
	}

	_, err = io.WriteString(file, string(js))

	if nil != err {
		log.Println("写入", desc, "json文件 ", filename, " 失败,err=", err)
		os.Exit(1)
	}
	log.Println("写入", desc, "json文件 ", filename, "到路径", dir, " 完成!")
}

func CheckJsonValid(str string) bool {
	var target interface{}
	if err := json.Unmarshal([]byte(str), &target); err != nil {
		log.Println("解析json失败，不是有效的json格式，ex=", err)
		return false
	}
	return true
}

func CheckJsonValidAndWriteFile(str string, dir string, filename string, flag string) {
	log.Println("本次配置", flag, "解析结果:")
	log.Println(str)
	log.Println("本次", flag, "解析完成,开始检测是否为合法Json格式")

	var target interface{}
	if err := json.Unmarshal([]byte(str), &target); err != nil {
		log.Println("解析", flag, "json失败，不是有效的json格式，ex=", err)
		os.Exit(1)
	}

	log.Println("解析", flag, "json成功，开始写入配置文件")
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
					log.Println("lua数组解析失败,数据转换失败,string,k=", k, ",v=", v, "vtype=", vtype)
					os.Exit(1)
				}
				tmpStr += "\"" + value + "\""
			case bool:
				value, ok := v.(bool)
				if !ok {
					log.Println("lua数组解析失败,数据转换失败,bool,k=", k, ",v=", v)
					os.Exit(1)
				}
				tmpStr += strconv.FormatBool(value)
			case float64:
				value, ok := v.(float64)
				if !ok {
					log.Println("lua数组解析失败,数据转换失败,float64,k=", k, ",v=", v)
					os.Exit(1)
				}
				tmpStr += strconv.FormatFloat(value, 'f', 10, 64)
			case int64:
				value, ok := v.(int64)
				if !ok {
					log.Println("lua数组解析失败,数据转换失败,int64,k=", k, ",v=", v)
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
						log.Println("lua数组解析失败,数据转换失败,unknow,k=", k, ",v=", v)
						os.Exit(1)
					} else {
						tmpStr += strconv.FormatFloat(vex, 'f', -1, 64)
					}
				} else {
					tmpStr += strconv.FormatInt(vex, 10)
				}
			default:
				log.Println("lua数组解析失败,数据转换失败,unknow,k=", k, ",v=", v)
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
						log.Println("lua数组解析失败,数据转换失败,string,k=", k, ",v=", v, "vtype=", vtype)
						os.Exit(1)
					}
					tmpStr = "\"" + value + "\""
				case bool:
					value, ok := v.(bool)
					if !ok {
						log.Println("lua数组解析失败,数据转换失败,bool,k=", k, ",v=", v)
						os.Exit(1)
					}
					tmpStr = strconv.FormatBool(value)
				case float64:
					log.Println("lua数组解析失败,数组内容不应该为float64,k=", k, ",v=", v)
					os.Exit(1)
				case int64:
					log.Println("lua数组解析失败,数组内容不应该为int64,k=", k, ",v=", v)
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
							log.Println("lua数组解析失败,数据转换失败,222unknow,k=", k, ",v=", v)
							os.Exit(1)
						} else {
							tmpStr += strconv.FormatFloat(vex, 'f', -1, 64)
						}
					} else {
						tmpStr += strconv.FormatInt(vex, 10)
					}
				default:
					log.Println("lua数组解析失败,数据转换失败,unknow,k=", k, ",v=", v)
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

func TransformJsonTolua(str string, typeCell TableType) string {
	luaStr := "return "
	decoder := json.NewDecoder(bytes.NewBufferString(str))
	decoder.UseNumber()
	if TableType_array == typeCell {
		var result []interface{}
		if err := decoder.Decode(&result); err != nil {
			log.Println("TransformJsonTolua json解析失败,str=", str, "\nerr=", err)
			os.Exit(1)
		}

		luaStr += TransfromInterfaceTolua(result, true)
	} else {
		var result map[string]interface{}
		if err := decoder.Decode(&result); err != nil {
			log.Println("TransformJsonTolua json解析失败,str=", str, "\nerr=", err)
			os.Exit(1)
		}
		luaStr += TransfromInterfaceTolua(result, true)
	}

	return luaStr
}

func StartGenConfig(xls *excelize.File, config RootDirStruct) {

	if "" == config.name {
		log.Println("主表为空")
		os.Exit(1)
	}

	log.Println("主表sheet名称为:", config.name)

	datalist, err := ReadField(xls, config.name)

	if err != nil {
		log.Println("读取字段名称出错，ex=", err)
		os.Exit(1)
	}

	typeCell, err := GetTypeCellAndCheck(xls, config.name)

	if err != nil {
		log.Println("获取生成配置类型错误，ex=", err)
		os.Exit(1)
	}

	filename := config.name + ".json"
	serverStr := ParseXlxs(xls, config.name, typeCell, datalist, StrategyType_Server)
	CheckJsonValidAndWriteFile(serverStr, config.serverDir, filename, "服务器")

	clientStr := ParseXlxs(xls, config.name, typeCell, datalist, StrategyType_Client)
	//CheckJsonValidAndWriteFile(clientStr, config.clientDir, filename, "客户端json")
	//CheckJsonValid(clientStr)

	if CheckJsonValid(clientStr) {
		luafilename := config.name + ".lua"
		clientluaStr := TransformJsonTolua(clientStr, typeCell)
		WriteConfigFile(clientluaStr, config.clientDir, luafilename, "客户端lua")
	} else {
		log.Println("生成lua配置错误，ex=", err)
		os.Exit(1)
	}
}

func ReadRootField(xls *excelize.File) ([]RootDirStruct, error) {

	dataStructList := []RootDirStruct{}

	rows, err := xls.Rows(TABLE_ROOT_SHEET_NAME)
	if err != nil {
		log.Println("读取", TABLE_ROOT_SHEET_NAME, "错误,err=", err)
		os.Exit(1)
	}

	rowIndex := 1
	for rows.Next() {
		if rowIndex <= int(TABLE_ROOT_ROW_DES) {
			rowIndex++
			continue
		}
		row := rows.Columns()
		colIndex := 1
		if len(row) <= 0 {
			break
		}
		for _, colCell := range row {
			if colIndex > int(TABLE_ROOT_COLUMN_CONFIG_CLIENT_DIR) {
				break
			}
			if "" == colCell {
				break
			}
			dataIndex := rowIndex - int(TABLE_ROOT_ROW_DES) - 1
			if dataIndex > len(dataStructList) {
				log.Println("root 表字段名称配置长度不一致 serverDir rowIndex=", rowIndex, ",colIndex=", colIndex, ",名称:", colCell)
				os.Exit(1)
			}

			if int(TABLE_ROOT_COLUMN_CONFIG_NAME) == colIndex {
				tmp := newRootDirStruct()
				tmp.name = colCell
				dataStructList = append(dataStructList, tmp)
			} else if int(TABLE_ROOT_COLUMN_CONFIG_SERVER_DIR) == colIndex {
				dataStructList[dataIndex].serverDir = colCell
			} else if int(TABLE_ROOT_COLUMN_CONFIG_CLIENT_DIR) == colIndex {
				dataStructList[dataIndex].clientDir = colCell
			}
			colIndex++
		}
		rowIndex++
	}

	if len(dataStructList) <= 0 {
		log.Println("root表为空")
		os.Exit(1)
	}

	for _, v := range dataStructList {
		if v.CheckEmpty() {
			log.Println("root表读取失败,有字段数据为空")
			v.Show()
			os.Exit(1)
		}
	}

	return dataStructList, err
}

func StartGetRoot(xls *excelize.File) {
	configList, _ := ReadRootField(xls)

	if len(configList) <= 0 {
		log.Println("root 表未配置任何数据")
		os.Exit(1)
	}

	for k, _ := range configList {
		log.Println("开始执行生成 ", configList[k].name)
		StartGenConfig(xls, configList[k])
		log.Println("执行生成 ", configList[k].name, " 完成\n==================================================================\n\n\n")
	}
}

func StartGenFile(filename string) {

	log.Println("开始执行配置表 ", filename, " 解析到json文件\n")

	xls, err := excelize.OpenFile(filename)

	if err != nil {
		log.Println("读取文件报错")
		log.Println(err)
		os.Exit(1)
	}
	StartGetRoot(xls)

	log.Println("配置表 ", filename, " 全部解析完成!\n=============\n\n")
}

func main() {

	arg_num := len(os.Args) - 1
	log.Printf("本次需生成文件数量为 %d\n", arg_num)
	for i := 1; i <= arg_num; i++ {
		StartGenFile(os.Args[i])
	}
	log.Println("全部文件生成完毕,文件数量为:", arg_num)
}
