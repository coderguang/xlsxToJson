package xlsxToJsonDef

import (
	"github.com/coderguang/GameEngine_go/sglog"
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
	StrategyType_NoGen  StrategyType = 3
)

type TableType int32

const (
	TableType_array     TableType = iota
	TableType_object    TableType = iota
	TableType_file_list TableType = iota
	TableType_end       TableType = iota
)

type DataStruct struct {
	Desc              string
	Name              string
	DataTypeValue     DataType
	StrategyTypeValue StrategyType
}

func (v *DataStruct) Show() {
	sglog.Info("表结构内容:Desc:%s,Name:%s,DataTypeValue:%d,StrategyTypeValue:%d", v.Desc, v.Name, int(v.DataTypeValue), int(v.StrategyTypeValue))
}

func (v *DataStruct) CheckEmpty() bool {
	if "" == v.Desc || "" == v.Name || DataType(-1) == v.DataTypeValue || StrategyType(-1) == v.StrategyTypeValue {
		return true
	}
	return false
}

func NewDataStruct() DataStruct {
	return DataStruct{
		Desc:              "",
		Name:              "",
		DataTypeValue:     DataType(-1),
		StrategyTypeValue: StrategyType(-1),
	}
}

type RootDirStruct struct {
	ServerDir string
	ClientDir string
	Name      string
}

func (v *RootDirStruct) Show() {
	sglog.Info("root表内容:Name:%s,ClientDir:%s,ServerDir:%s", v.Name, v.ClientDir, v.ServerDir)
}

func (v *RootDirStruct) CheckEmpty() bool {
	if "" == v.Name || "" == v.ServerDir || "" == v.ClientDir {
		return true
	}
	return false
}

func NewRootDirStruct() RootDirStruct {
	return RootDirStruct{
		ServerDir: "",
		ClientDir: "",
		Name:      "",
	}
}

const TABLE_FORMAT_ROW_DES int = 1
const TABLE_FORMAT_ROW_Name int = 2
const TABLE_FORMAT_ROW_CONFIG_STRATEGY int = 3
const TABLE_FORMAT_ROW_DATATYPE int = 4

const TABLE_FORMAT_COLUMN_DES int = 1
const TABLE_FORMAT_COLUM_REAL_DATA_INDEX int = 2

const TABLE_FORMAT_TYPECELL_POS string = "A1"

const TABLE_ROOT_SHEET_Name string = "root"
const TABLE_ROOT_COLUMN_CONFIG_Name int = 1
const TABLE_ROOT_COLUMN_CONFIG_SERVER_DIR int = 2
const TABLE_ROOT_COLUMN_CONFIG_CLIENT_DIR int = 3
const TABLE_ROOT_ROW_DES int = 1
