package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	"time"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

const (
	FILEBUFFER int = 4096
)

var ReqArr []string = []string{"GET", "POST", "PUT", "DELETE"}

var IP string
var PORT string

var HOST string = "http://"

var ROOTLOC string
var NOTFOUNDLOC string

var DB *sql.DB

type Request struct {
	request     int
	requestData string
	host        string
	accepts     []string
	connection  string
	useragent   string
	refer       string
	otherReq    []string
}

func main() {
	if len(os.Args) > 3 {
		fmt.Printf("Argument Error. %d Argument Writed.(Max Arg:3)", len(os.Args))
		return
	} else if len(os.Args) < 3 {
		fmt.Printf("Argument Error. Need 3 Argument Get %d.", len(os.Args))
		return
	} else if ((isIp(os.Args[1]) != 0) != (isIp(os.Args[2]) != 0)) == false {
		fmt.Printf("Arguments Wrong.( [Server Loc] [IP] [PORT] )")
		return
	}

	IP = ""
	PORT = ""

	if isIp(os.Args[1]) == 0 {
		IP = os.Args[1]
		PORT = os.Args[2]
	} else {
		IP = os.Args[2]
		PORT = os.Args[1]
	}

	if PORT == "80" {
		HOST += IP
	} else {
		HOST += IP + ":" + PORT
	}

	fmt.Printf("Starting Socket On %s:%s\n", IP, PORT)

	sock, err := net.Listen("tcp", IP+":"+PORT)

	if err != nil {
		HandleErr(errors.New("SockListen!" + err.Error()))
	}

	RootLoc, err := os.Getwd()

	if runtime.GOOS == "windows" {
		ROOTLOC = RootLoc + "\\"
	} else {
		ROOTLOC = RootLoc + "/"
	}

	fmt.Printf("Serve Location: %s\n", RootLoc)
	NOTFOUNDLOC = ROOTLOC + "404\\"

	if _, err := os.Stat(RootLoc[:strings.LastIndex(RootLoc, "\\")] + "\\sqlite-database.db"); os.IsNotExist(err) {
		HandleErr(errors.New("DatabaseRead!" + err.Error()))
	}

	DB, err = sql.Open("sqlite3", "..\\sqlite-database.db")

	fmt.Printf("Database Location: %s\n", RootLoc[:strings.LastIndex(RootLoc, "\\")]+"\\sqlite-database.db")

	if err != nil {
		HandleErr(errors.New("DatabaseRead!" + err.Error()))
	}

	for {
		conn, err := sock.Accept()
		if err != nil {
			HandleErr(errors.New("SockAccept!" + err.Error()))
		}

		go handleConn(conn)
	}

}

func listenConn(conn net.Conn) (string, error) {

	reader := bufio.NewReader(conn)
	str := ""
	//ar req Request

	for {
		data, err := reader.ReadByte()

		if (len(str) > 4) && (string(str[len(str)-3:]) == "\r\n\r") {
			str += string(data)
			return str, io.EOF
		}
		if err == io.EOF {
			str += string(data)

			return str, err
		}
		if err != nil {
			conn.Close()
			HandleErr(errors.New("SockListen!" + err.Error()))
			return str, err
		}

		str += string(data)
	}
}

func handleConn(conn net.Conn) {
	fmt.Printf("Connection Initilized\n")
	fmt.Printf("Connection Adreess: %s, %s\n", conn.RemoteAddr(), conn.LocalAddr())

	req, err := listenConn(conn)

	var requset Request
	if err == io.EOF {

		arr := strings.Split(req, "\r\n")

		requset.request = CheckRequest(arr[0])

		if requset.request < 0 {
			HandleErr(errors.New("InvalidRequest!" + err.Error()))
			conn.Close()
			return
		}

		requset.requestData = strings.Split(arr[0], " ")[1]

		requset.accepts = []string{}
		requset.otherReq = []string{}

		for i := 1; i < len(arr); i++ {
			oz := strings.Split(arr[i], ":")
			if requset.request < 0 {
				HandleErr(errors.New("InvalidRequest!" + err.Error()))
				conn.Close()
				return
			}

			oz1 := strings.ToUpper(oz[0])
			switch oz1 {
			case "HOST":
				oz2 := strings.Split(arr[i], ":")[1]
				if oz2[:7] == "http:\\\\" {
					requset.host = strings.Split(arr[i], ":")[1]
					fmt.Println("1")
				} else {
					fmt.Println("2")
					requset.host = "http:\\\\" + oz2[1:]
				}

			case "CONNECTION":
				requset.connection = strings.Split(arr[i], ":")[1]
			case "ACCEPT":
				requset.accepts = strings.Split(strings.Split(arr[i], ":")[1], ",")
			case "USER-AGENT":
				requset.useragent = strings.Split(arr[i], ":")[1]
			case "REFERER":
				requset.refer = strings.Join(strings.Split(arr[i], ":")[1:], "")[1:]

				requset.refer = requset.refer[strings.Index(requset.refer[7:], "/")+7:]
			}
		}

		if requset.refer == "" {
			requset.refer += "\\"
		}

	} else {
		HandleErr(errors.New("Listening!" + err.Error()))
		conn.Close()
		return
	}

	fmt.Printf("Request: %s %s\n", ReqArr[requset.request], requset.requestData)

	switch requset.request {
	case 0:
		RequestGet(conn, requset)
		break
	case 1:
		break
	}

}

func RequestGet(conn net.Conn, req Request) error {
	var response string = ""
	var path string = ""
	var code int = 200

	if req.requestData[len(req.requestData)-1] == '/' {
		req.requestData += "index.html"
	}

	if req.requestData[len(req.requestData)-5:] != ".html" {
		req.requestData = ConCatPath(req.refer, req.requestData, string(OsSep(runtime.GOOS)))
	}

	req.requestData = ConvertPath(req.requestData, runtime.GOOS)
	path = ConCatPath(ROOTLOC, req.requestData, string(OsSep(runtime.GOOS)))

	filestat, err := os.Stat(path)
	if err != nil {
		code = 404
		path = NOTFOUNDLOC + "index.html"

		filestat, err = os.Stat(path)

		if err != nil {
			HandleErr(errors.New("FileStat!" + err.Error()))
		}
	}

	if filestat.IsDir() && req.requestData != "\\" {
		req.requestData += "\\"
	}
	if req.requestData[len(req.requestData)-1] == '\\' {
		req.requestData += "index.html"
	}

	path = ConCatPath(ROOTLOC, req.requestData, string(OsSep(runtime.GOOS)))

	filestat, err = os.Stat(path)
	if err != nil {
		code = 404
		path = NOTFOUNDLOC + "index.html"

		filestat, err = os.Stat(path)
		if err != nil {
			HandleErr(errors.New("FileStat!" + err.Error()))
			return err
		}
	}

	stype := path[strings.LastIndex(path, ".")+1:]

	i := filestat.Size()

	res, err := DB.Query("SELECT RESPONSE FROM RESPONSE WHERE CODE = ?;", code)
	res.Next()
	res.Scan(&response)
	res.Close()

	response = strings.ReplaceAll(response, "\\r\\n", "\r\n")
	response = strings.Replace(response, "{0}", time.Now().Format(time.RFC1123), 1)
	response = strings.Replace(response, "{1}", "text/"+stype, 1)
	response = strings.Replace(response, "{2}", strconv.FormatInt(i, 10), 1)

	conn.Write([]byte(response))

	WriteFileToConn(conn, path)

	conn.Write([]byte("\r\n\r\n"))
	conn.Close()

	InsertToREQUESTS(req)

	return err
}

func WriteFileToConn(conn net.Conn, path string) error {
	file, err := os.Open(path)
	if err != nil {
		HandleErr(errors.New("FileRead!" + err.Error()))
		return err
	}

	stat, err := file.Stat()
	if err != nil {
		HandleErr(errors.New("FileStat!" + err.Error()))
		return err
	}

	i := int(stat.Size())
	b1 := make([]byte, FILEBUFFER)

	for ; i > FILEBUFFER; i -= FILEBUFFER {
		file.Read(b1)
		conn.Write(b1)
	}

	b1 = make([]byte, i)

	file.Read(b1)
	file.Close()

	conn.Write(b1)

	return err
}

func CheckRequest(str string) int {
	index := strings.Index(str, " ")
	if index == -1 {
		return -1
	}

	slicedStr := str[:index]
	if len(slicedStr) > 4 || len(slicedStr) < 0 {
		return -2
	}

	for i := 0; i < len(ReqArr); i++ {
		if slicedStr == ReqArr[i] {
			return i
		}
	}
	return -3

}

func isIp(str string) int {
	arr := strings.Split(str, ".")

	if len(arr) != 4 {
		return 1
	}

	for i := 0; i < 4; i++ {
		tmp, err := strconv.Atoi(arr[i])
		if err != nil {
			return 3
		}

		if tmp > 255 || tmp < 0 {
			return 2
		}
	}

	return 0
}

func InsertToREQUESTS(req Request) error {
	row, err := DB.Query("SELECT ID FROM HOSTS WHERE IP=? AND PORT=?;", req.host, "0")
	var stm *sql.Stmt

	if err != nil {
		HandleErr(errors.New("SQLSelect!" + err.Error()))
		return err
	}

	if !row.Next() {
		stm, err = DB.Prepare("INSERT INTO HOSTS(IP,PORT) VALUES (?,?)")
		if err != nil {
			HandleErr(errors.New("SQLPrepare!" + err.Error()))
			return err
		}
		stm.Exec(req.host, "0")

		row, err = DB.Query("SELECT ID FROM HOSTS WHERE IP=? AND PORT=?;", req.host, "0")
		if err != nil {
			HandleErr(errors.New("SQLSelect!" + err.Error()))
			return err
		}
	}

	var id int = -1

	row.Scan(&id)

	row.Close()

	ti := time.Now()

	stm, err = DB.Prepare("INSERT INTO REQUESTS (TYPE, DATA, HOST, ACCEPTS, CONNECTION, USERAGENT, DATE ) VALUES (?,?,?,?,?,?,?)")
	if err != nil {
		HandleErr(errors.New("SQLPrepare!" + err.Error()))
		return err
	}

	_, err = stm.Exec(req.request, req.requestData, id, strings.Join(req.accepts, ", "), req.connection, req.useragent, ti.Format("2006-01-02 15:04:05.000000000"))
	if err != nil {
		HandleErr(errors.New("SQLInsert!" + err.Error()))
		return err
	}

	return err
}

func HandleErr(err error) {
	switch strings.Split(err.Error(), "!")[0] {
	case "SockListen":
		break
	case "SockAccept":
		break
	case "DatabaseRead":
		break
	case "InvalidRequest":
		break
	case "Listening":
		break
	case "FileRead":
		break
	case "FileStat":
		break
	case "SQLSelect":
		break
	case "SQLPrepare":
		break
	default:
		break
	}
}

//***************************************************

func ConvertPath(path string, toOs string) string {
	switch toOs {
	case "windows":
		if len(path) > 2 && path[1] == ':' {
			path = path[1:]
		}
		return strings.ReplaceAll(path, "/", "\\")
	default:
		return strings.ReplaceAll(path, "\\", "/")
	}
}

func ConCatPath(path1 string, path2 string, sep string) string {
	if path1[len(path1)-1] == '\\' || path1[len(path1)-1] == '/' {
		if path2[0] == '\\' || path2[0] == '/' {
			return path1[:len(path1)-1] + path2 //[1:]Dont Use
		}

		return path1 + path2
	}
	if path2[0] == '\\' || path2[0] == '/' {
		return path1 + path2
	}

	return path1 + sep + path2
}

func OsSep(os string) byte {
	switch os {
	case "windows":
		return '\\'
	case "unix":
		return '/'
	default:
		return '\n'
	}
}
