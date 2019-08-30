xlsxToJson
===
[![Build Status](https://travis-ci.org/coderguang/xlsxToJson.svg?branch=master)](https://travis-ci.org/coderguang/xlsxToJson)
![](https://img.shields.io/badge/language-golang-orange.svg)
[![codebeat badge](https://codebeat.co/badges/e3189da4-243c-4a00-a052-dd9f2fba708b)](https://codebeat.co/projects/github-com-coderguang-xlsxtojson-master)
[![](https://img.shields.io/badge/wp-@royalchen-blue.svg)](https://www.royalchen.com)

## **a tool to generate json and lua config data by xlsx**
  
## what it can do
  * generate json and lua code by xlsx file
  * can link other sheet
  * can generate file by every column
  * more xlsx file detail please read **template.xlsx** file
  
## how to start
### 1. clone repository 
```shell
git clone git@github.com:coderguang/xlsxToJson.git xlsxToJson
```

### 2. build project
```shell
go build
```

### 2. run script in windows
```shell
call build.bat
```
###  if your are in linux,it should be 
 ```shell
    ./xlsxToJson template.json
```
