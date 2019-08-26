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

func (v *DataStruct) Show() {
	sglog.Info("表结构内容:desc:%s,name:%s,dataType:%d,strategyType:%d", v.desc, v.name, int(v.dataType), int(v.strategyType))
}

func (v *DataStruct) CheckEmpty() bool {
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

func (v *RootDirStruct) Show() {
	sglog.Info("root表内容:name:%s,clientDir:%s,serverDir:%s", v.name, v.clientDir, v.serverDir)
}

func (v *RootDirStruct) CheckEmpty() bool {
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
