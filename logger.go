package klog

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	Info        *klogger
	Error       *klogger
	logFileName string
	logFlag     int
	logFile     *os.File
	// 日志文件最大size
	maxFileSize int64

	// 日志是否同时输入到stdout中
	stdoutFlag bool

	// 是否需要对Archive进行gzip压缩存储
	compressArchive bool

	// 文件重命名的时候，需要加锁
	mu sync.Mutex // ensures atomic writes; protects the following fields
)

type klogger struct {
	logger       *log.Logger
	stdoutLogger *log.Logger
}

func init() {
	InitKLog("", os.O_CREATE|os.O_APPEND|os.O_RDWR, true)
	SetMaxFileSizeMB(10)
}

func newLogger(out io.Writer, prefix string, flag int, stdoutFlag bool) *klogger {
	logger := new(klogger)

	if nil != out {
		logger.logger = log.New(out, prefix, flag)
	}

	if true == stdoutFlag {
		logger.stdoutLogger = log.New(os.Stdout, prefix, flag)
	}
	return logger
}

// 设置日志文件, 文件名，日志格式，是（同时）否向stdout输出日志
func InitKLog(iFileName string, iLogFlag int, iStdoutFlag bool) {
	if nil != logFile {
		err := logFile.Close()
		if nil != err {
			log.Printf("[Error] Close old log file Error: %v", logFile.Close())
		}
	}

	logFileName = iFileName
	logFileDir, err := filepath.Abs(logFileName)
	if "" == logFileDir || nil != err {
		log.Printf("[Error] Parse klog dir failed: %s, %v", logFileDir, err)
		return
	}
	logFileDir = filepath.Dir(logFileDir)
	// 创建目录
	if _, err := os.Stat(logFileDir); os.IsNotExist(err) {
		err = os.MkdirAll(logFileDir, os.ModePerm)
		if nil != err {
			log.Printf("[Error] Create klog dir failed: %s, %v", logFileDir, err)
			return
		}
	}

	// 默认不允许日志标志位为空
	if 0 != iLogFlag {
		logFlag = iLogFlag
	}
	stdoutFlag = iStdoutFlag

	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0660)
	if err != nil {
		log.Printf("[Error] Failed to open log logFile: %v, logFile: %v", logFileName, logFile)
		logFile = nil
	}
	// 日志文件格式:log包含时间及文件行数
	Info = newLogger(logFile, "[Info]  ", logFlag, stdoutFlag)
	Error = newLogger(logFile, "[Error] ", logFlag, stdoutFlag)
}

// 备份日志文件
// 出于效率考虑，klog默认不支持隔日自动创建新日志文件，自动备份日志的触发条件，只有日志文件大小。
// 如有需求隔日创建，建议使用定时调度程序在00:00:00调用这个方法。
func ArchiveLogFile() {
	// 加锁，对文件进行重命名
	mu.Lock()
	defer mu.Unlock()
	fInfo, err := os.Stat(logFileName)
	if nil != err {
		log.Printf("[Error] %v getLogFileSize Error: %v", time.Now().String(), err)
		return
	}

	// 获取备份文件名
	archiveName := fInfo.ModTime().String()
	archiveName = archiveName[0:strings.Index(fInfo.ModTime().String(), ".")]
	archiveName = strings.ReplaceAll(archiveName, " ", "_")
	archiveName = strings.ReplaceAll(archiveName, ":", "-")
	archiveName = filepath.Dir(logFileName) + archiveName + ".log"

	err = os.Rename(logFileName, archiveName)
	if nil != err {
		if nil != Error.logger {
			Error.logger.Printf("Error for rename overload log file: %v", err)
		}

		if nil != Error.stdoutLogger {
			Error.stdoutLogger.Printf("Error for rename overload log file: %v", err)
		}
		return
	}

	InitKLog(logFileName, logFlag, stdoutFlag)
}

func compressArchiveFile(archiveName string) {
	if false == compressArchive {
		return
	}
	fw, err := os.Create(archiveName + ".gzip") // 创建gzip包文件，返回*io.Writer
	if err != nil {
		log.Fatalln(err)
	}
	defer fw.Close()

	// 实例化心得gzip.Writer
	gw := gzip.NewWriter(fw)
	defer gw.Close()

	// 获取要打包的文件信息
	fr, err := os.Open(archiveName)
	if err != nil {
		log.Fatalln(err)
	}
	defer fr.Close()

	// 获取文件头信息
	fi, err := fr.Stat()
	if err != nil {
		log.Fatalln(err)
	}

	// 创建gzip.Header
	gw.Header.Name = fi.Name()

	// 读取文件数据
	buf := make([]byte, fi.Size())
	_, err = fr.Read(buf)
	if err != nil {
		log.Fatalln(err)
	}

	// 写入数据到zip包
	_, err = gw.Write(buf)
	if err != nil {
		log.Fatalln(err)
	}
	_ = os.Remove(archiveName)
}

// 返回日志文件最大size，单位是byte
func (l *klogger) GetMaxFileSize() int64 {
	return maxFileSize
}

// 以KB为单位设置日志文件最大size
func SetMaxFileSizeKB(size int) {
	maxFileSize = int64(size) * 1024
}

// 以MB为单位设置日志文件最大size
func SetMaxFileSizeMB(size int) {
	maxFileSize = int64(size) * 1024 * 1024
}

// 设置是否对归档日志进行压缩
func (l *klogger) SetCompressArchive(flag bool) {
	compressArchive = flag
}

// 设置日志标志
func (l *klogger) SetLogFlag(flag int) {
	logFlag = flag
}

func (l *klogger) Fatalf(format string, v ...interface{}) {
	checkLogFileSize()

	if nil != l.stdoutLogger {
		_ = l.stdoutLogger.Output(2, fmt.Sprintf(format, v...))
	}
	if nil != l.logger {
		_ = l.logger.Output(2, fmt.Sprintf(format, v...))
		os.Exit(1)
	}
}

func (l *klogger) Printf(format string, v ...interface{}) {
	checkLogFileSize()
	if nil != l.logger {
		_ = l.logger.Output(2, fmt.Sprintf(format, v...))
	}

	if nil != l.stdoutLogger {
		_ = l.stdoutLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

func (l *klogger) Println(v ...interface{}) {
	checkLogFileSize()
	if nil != l.logger {
		_ = l.logger.Output(2, fmt.Sprintln(v...))
	}

	if nil != l.stdoutLogger {
		_ = l.stdoutLogger.Output(2, fmt.Sprintln(v...))
	}
}

func checkLogFileSize() {
	if "" == logFileName {
		return
	}

	fInfo, err := os.Stat(logFileName)
	if nil != err {
		fmt.Printf("getLogFileSize Error: %v", err)
		return
	}

	currSize := fInfo.Size()

	//如果不是同一天，就强制更换日志
	if time.Now().Day() != fInfo.ModTime().Day() {
		currSize = maxFileSize
	}

	if currSize < maxFileSize {
		return
	}

	// 加锁，对文件进行重命名
	mu.Lock()
	defer mu.Unlock()

	// 获取备份文件名
	archiveName := fInfo.ModTime().String()
	archiveName = archiveName[0:strings.Index(fInfo.ModTime().String(), ".")]
	archiveName = strings.ReplaceAll(archiveName, " ", "_")
	archiveName = strings.ReplaceAll(archiveName, ":", "-")
	archiveName = filepath.Join(filepath.Dir(logFileName), archiveName+".log")
	// 加锁后，再获取并检测一次文件大小，防止异步时重新加载
	fInfo, err = os.Stat(logFileName)
	if nil != err {
		log.Printf("getLogFileSize Error: %v", err)
		return
	}
	currSize = fInfo.Size()
	if currSize < maxFileSize {
		return
	}

	err = os.Rename(logFileName, archiveName)
	if nil != err {
		if nil != Error.logger {
			Error.logger.Printf("Error for rename overload log file: %v", err)
		}

		if nil != Error.stdoutLogger {
			Error.stdoutLogger.Printf("Error for rename overload log file: %v", err)
		}
		return
	}

	InitKLog(logFileName, logFlag, stdoutFlag)
	go compressArchiveFile(archiveName)
}
