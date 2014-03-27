package main

import (
	"config"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const PATH_SEPERATOR = string(os.PathSeparator)

// one day = 24 * 60 * 60
var oneDayAgo = time.Unix(time.Now().Unix()-86400, 0).UTC().Format("2006-01-02 15:04:05")

type LogInfo struct {
	AuthorName         string
	AuthorEmail        string
	CommitterName      string
	CommitterEmail     string
	DateTime           int
	FileListWithStatus [][2]string
}

type OutputToStore struct {
	Stakeholders []string
	Code         string
	Result       string
}

var fileTypeToCheck = []string{
	"js",
	"php",
	"py",
	//"css",
}

var gjslint_ignore_list = []string{
	// 文件名不支持正则表达式
	".+.min.js",
}

var flake8_ignore_list = []string{
	"E501",
	"W292",
}

var gjslint_ignoring = "-x " + strings.Join(gjslint_ignore_list, ",")
var flake8_ignoring = "--ignore=" + strings.Join(flake8_ignore_list, ",")

var typeMapCmd = map[string][]string{
	"js":  []string{"/usr/local/bin/gjslint", "--nojsdoc", gjslint_ignoring},          // Google JavaScript Closure Linter
	"php": []string{"/usr/local/bin/phpcs", "-n", "--standard=Zend", "--tab-width=4"}, // PHP_CodeSniffer
	"py":  []string{"/usr/local/bin/flake8", flake8_ignoring},                         // http://flake8.readthedocs.org/en/2.0/
	//"css": []string{"/usr/local/bin/phpcs", phpcs_ignoring, "--standard=Squiz"},
}

func inArray(array []string, me string) bool {
	for _, v := range array {
		if v == me {
			return true
		}
	}
	return false
}

// 更新版本库工作目录
func updateReposWorkingDir() {
	cmd := exec.Command("git", "pull", "origin", "master")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
	fmt.Println(string(output))
}

func countCommitInLastDay(conf config.ConfigInfo) int {
	db, err := sql.Open(conf.DBDriverName, conf.DBDataSourceName)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(oneDayAgo)
	row := db.QueryRow("SELECT count(*) FROM events WHERE project_id=9 AND updated_at > ?", oneDayAgo)
	var count int
	row.Scan(&count)
	fmt.Printf("count: %d\n", count)
	return count
}

// 解析Git日志，抽取需要的信息
func parseGitLog(conf config.ConfigInfo) []LogInfo {
	count := countCommitInLastDay(conf)
	maxNum := fmt.Sprintf("-n %d", count)
	args := []string{"log", "master", "--pretty=format:'%an %ae %cn %ce %at %b%n'", maxNum, "--reverse", "--name-status", "--diff-filter='A|M|D'"}
	cmd := exec.Command("git", args...)
	fmt.Println(cmd.Args)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}
	if len(output) == 0 {
		return nil
	}
	substrings := strings.Split(string(output), "\n")

	logInfoList := make([]LogInfo, 0, 50)

	index := 0
	lineNum := len(substrings)

	for index < lineNum {
		value := substrings[index]
		if strings.HasPrefix(value, "'") && len(value) > 1 {
			commitInfo := strings.Split(value[1:], " ")

			authorName := commitInfo[0]
			authorEmail := commitInfo[1]
			committerName := commitInfo[2]
			committerEmail := commitInfo[3]
			dateTimeStr := commitInfo[4]
			dateTime, err := strconv.Atoi(dateTimeStr)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
			}

			index += 1

			filePathWithStatusList := make([][2]string, 0, 30)
			for {
				index += 1
				value = substrings[index]
				if value != "" {
					var filePathWithStatus [2]string
					filePathWithStatus[0] = value[:1]
					filePathWithStatus[1] = strings.Trim(value[1:], "\t")
					filePathWithStatusList = append(filePathWithStatusList, filePathWithStatus)
				} else {
					index += 1
					break
				}
			}
			logInfo := LogInfo{
				AuthorName:         authorName,
				AuthorEmail:        authorEmail,
				CommitterName:      committerName,
				CommitterEmail:     committerEmail,
				DateTime:           dateTime,
				FileListWithStatus: filePathWithStatusList,
			}
			logInfoList = append(logInfoList, logInfo)
		}
	}
	return logInfoList
}

func isStakeholderExist(stakeholders []string, stakeholder string) bool {
	for _, value := range stakeholders {
		if stakeholder == value {
			return true
		}
	}
	return false
}

// 过滤解析得到的日志信息
func customFilter(LogInfoList []LogInfo) map[string][]string {
	filePathMapStakeholders := make(map[string][]string)
	for _, logInfo := range LogInfoList {
		for _, filePathWithStatus := range logInfo.FileListWithStatus {
			status := filePathWithStatus[0]
			filePath := filePathWithStatus[1]
			if status == "D" {
				delete(filePathMapStakeholders, filePath)
			} else {
				if _, ok := filePathMapStakeholders[filePath]; ok {
					if !isStakeholderExist(filePathMapStakeholders[filePath], logInfo.AuthorEmail) {
						filePathMapStakeholders[filePath] = append(filePathMapStakeholders[filePath], logInfo.AuthorEmail)
					}
					if !isStakeholderExist(filePathMapStakeholders[filePath], logInfo.CommitterEmail) {
						filePathMapStakeholders[filePath] = append(filePathMapStakeholders[filePath], logInfo.CommitterEmail)
					}
				} else {
					stakeholders := make([]string, 0, 10)
					stakeholders = append(stakeholders, logInfo.AuthorEmail)
					if !isStakeholderExist(stakeholders, logInfo.CommitterEmail) {
						stakeholders = append(stakeholders, logInfo.CommitterEmail)
					}
					filePathMapStakeholders[filePath] = stakeholders
				}
			}
		}
	}
	return filePathMapStakeholders
}

// 为代码添加行号
func addLineNumForCode(code string) string {
	lines := strings.Split(code, "\n")
	width := len(strconv.Itoa(len(lines)))
	for index, value := range lines {
		lines[index] = fmt.Sprintf("%.*d. %s", width, index+1, value)
	}
	return strings.Join(lines, "\n")
}

// 提交静态分析结果
func commitResultToRepos(basePath string, analysisResultReposName string, repos string, subdir string) {
	_ = os.Chdir(filepath.Join(basePath, analysisResultReposName))
	addCmd := exec.Command("git", "add", strings.Replace(filepath.Join(repos, subdir), PATH_SEPERATOR, "/", -1))
	addCmd.Output()
	commitCmd := exec.Command("git", "commit", "-m", "'add "+subdir+"'")
	commitCmd.Output()
	pushCmd := exec.Command("git", "push", "origin", "master")
	pushCmd.Output()
}

func sendMail(mailScript string, sender string, receiver string, title string, content string) {
	cmd := exec.Command(mailScript, "-s", sender, "-r", receiver, "-t", title, "-c", content)
	//fmt.Println(cmd.Args)
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("In sendMail: %v\n", err)
	}
	fmt.Println(string(out))
}

// 发送邮件通知
func mailStakeholders(conf config.ConfigInfo, repos string, stakeholderMapInfo map[string][]string, date string) {
	for key, value := range stakeholderMapInfo {
		//fmt.Printf("key: %v, value: %v\n", key, value)
		title := date + " 代码库" + repos + " 与您相关的代码静态分析结果"
		content := fmt.Sprintf("请查看代码静态分析结果，并尽可能纠正其中的错误和警告：\n%s\n谢谢！", strings.Join(value, "\n"))
		//key = "yongfengxia@tencent.com"
		sendMail(conf.MailScript, conf.MailSender, key, title, content)
	}
}

func main() {

	// 启用CPU性能测试
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			fmt.Println(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// 获取应用配置
	// 相对路径以当前执行“编译好的可执行程序”的路径为标准
	conf, err := config.ParseConfig("conf/app_conf.json")
	if err != nil {
		fmt.Println(err)
	}
	// fmt.Println(conf)
	for _, repos := range conf.TargetReposName {
		err = os.Chdir(filepath.Join(conf.BasePath, repos))
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return
		}
		// 更新版本库工作区
		updateReposWorkingDir()
		// 解析Git log
		logInfoList := parseGitLog(conf)
		if logInfoList == nil || len(logInfoList) == 0 {
			return
		}

		// 将解析的结果写到json文件中，调试用
		dataJson, err := json.MarshalIndent(logInfoList, "", "    ")
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		}
		f, err := os.OpenFile(filepath.Join(conf.BasePath, "codelintset", "logInfo.json"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
		}
		defer f.Close()
		f.Write(dataJson)
		f.Sync()
		
		// 创建分析结果存储目录
		date := time.Now().Format("2006-01-02")
		resultPath := filepath.Join(conf.BasePath, conf.AnalysisResultReposName, repos, date)
		resultUrl := conf.AnalysisResultReposUrl + "/" + repos + "/" + date
		if len(logInfoList) > 0 {
			err = os.MkdirAll(resultPath, 0777)
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		// git log解析结果再处理，过滤已被删除的文件
		filePathList := customFilter(logInfoList)

		stakeholderMapMailContent := make(map[string][]string)

		for key, value := range filePathList {
			fileName := key
			fileNameParts := strings.Split(fileName, ".")
			partsNum := len(fileNameParts)
			fi, err := os.Stat(fileName)
			if err != nil {
				fmt.Println(err)
			}
			fileSize := fi.Size()
			if (partsNum > 1) && (fileSize < 51200) {
				fileType := strings.ToLower(fileNameParts[partsNum-1])
				if inArray(fileTypeToCheck, fileType) {
					cmdWithArgs := typeMapCmd[fileType]
					fileName = strings.Replace(fileName, "/", PATH_SEPERATOR, -1)
					argList := append(cmdWithArgs[1:], fileName)
					cmdObj := exec.Command(cmdWithArgs[0], argList...)
					out, _ := cmdObj.Output()
					result := string(out)
					gjslintSkippingOut := "Skipping 1 file(s).\n0 files checked, no errors found.\n"
					if cmdWithArgs[0] == "gjslint" && result == gjslintSkippingOut {
						result = ""
					}
					if result != "" {
						fmt.Println(fileName)
						resultFileName := strings.Replace(fileName, PATH_SEPERATOR, "~", -1) + ".md"
						resultFilePath := filepath.Join(resultPath, resultFileName)
						f, err := os.OpenFile(resultFilePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
						if err != nil {
							fmt.Println(err)
							break
						}
						bytes, _ := ioutil.ReadFile(fileName)
						code := string(bytes)
						code = addLineNumForCode(code)
						t, err := template.ParseFiles(filepath.Join(conf.BasePath, "codelintset", "template.tmpl"))
						if err != nil {
							fmt.Println(err)
							break
						}
						t.ExecuteTemplate(f, "lintresult", OutputToStore{Code: code, Result: result, Stakeholders: value})
						for _, person := range value {
							resultFileUrl := resultUrl + "/" + resultFileName
							if _, ok := stakeholderMapMailContent[person]; ok {
								stakeholderMapMailContent[person] = append(stakeholderMapMailContent[person], resultFileUrl)
							} else {
								stakeholderMapMailContent[person] = []string{resultFileUrl}
							}
						}
					}
				}
			}
		}
		commitResultToRepos(conf.BasePath, conf.AnalysisResultReposName, repos, date)
		mailStakeholders(conf, repos, stakeholderMapMailContent, date)
	}
}
