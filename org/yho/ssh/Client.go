package ssh

import (
	"bufio"
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

var currentPath, _ = os.Executable()
var currentRunPath = path.Dir(currentPath)                                           //运行路径
var currentConfPath = currentRunPath + string(os.PathSeparator) + "conf"             //配置路径
var defaultSshConfigPath = currentConfPath + string(os.PathSeparator) + "ssh.config" //默认ssh连接配置路径

type Cli struct {
	IP         string      //IP地址
	Username   string      //用户名
	Password   string      //密码
	Port       int         //端口号
	client     *ssh.Client //ssh客户端
	LastResult string      //最近一次Run的结果
	Remark     string      //备注
}

func New(ip string, username string, password string, port ...int) *Cli {
	cli := new(Cli)
	cli.IP = ip
	cli.Username = username
	cli.Password = password
	if len(port) <= 0 {
		//默认22
		cli.Port = 22
	} else {
		cli.Port = port[0]
	}
	return cli
}

func (c *Cli) connect() error {
	config := ssh.ClientConfig{
		User: c.Username,
		Auth: []ssh.AuthMethod{ssh.Password(c.Password)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 10 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", c.IP, c.Port)
	sshClient, err := ssh.Dial("tcp", addr, &config)
	if err != nil {
		return err
	}
	c.client = sshClient
	return nil
}

func (c *Cli) RunTerminal(stdout, stderr io.Writer) error {
	if c.client == nil {
		if err := c.connect(); err != nil {
			return err
		}
	}
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(fd, oldState)

	session.Stdout = stdout
	session.Stderr = stderr
	session.Stdin = os.Stdin

	termWidth, termHeight, err := terminal.GetSize(fd)
	if err != nil {
		panic(err)
	}
	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// Request pseudo terminal
	if err := session.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
		return err
	}
	session.Shell()
	session.Wait()

	return nil
}

const ShellToUse = "bash"

func ShellOut(command string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(ShellToUse, "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err != nil {
		log.Printf("error: %v\n", err)
	}
	//fmt.Println("--- stdout ---")
	fmt.Println(stdout.String())
	//fmt.Println("--- stderr ---")
	fmt.Println(stderr.String())
}
func ShellIn() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		//ShellOut("ls -ltr")
		ShellOut(text)
	}
}
func TestClient() {
	cli := New("127.0.0.1", "yho", "1217", 22)

	err := cli.RunTerminal(os.Stdout, os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
}
func ReadSshConfig(filePath string) []Cli {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	sshArrStr := string(data)
	strArr := strings.Fields(sshArrStr)
	var sshConfigs = make([]Cli, len(strArr))

	for i, v := range strArr {
		if v == "" || len(v) == 0 {
			break
		}
		sshConfigArr := strings.Split(v, ",")
		ip := replace(strings.Split(sshConfigArr[0], "=")[1])
		port, _ := strconv.Atoi(replace(strings.Split(sshConfigArr[1], "=")[1]))
		username := replace(strings.Split(sshConfigArr[2], "=")[1])
		password := replace(strings.Split(sshConfigArr[3], "=")[1])
		remark := replace(strings.Split(sshConfigArr[4], "=")[1])
		var c Cli
		c.IP = ip
		c.Port = port
		c.Username = username
		c.Password = password
		c.Remark = remark
		sshConfigs[i] = c
		fmt.Println(i, "ip="+c.IP, "remark="+remark)
	}
	return sshConfigs
}
func replace(str string) string {
	// 去除空格
	str = strings.Replace(str, " ", "", -1)
	// 去除换行符
	str = strings.Replace(str, "\n", "", -1)
	return str
}

// ReadUserWrite 读取用户键盘输入
func ReadUserWrite(sshConfigs []Cli) int {
	//接收键盘录入的服务器编号
	var number int
	fmt.Println("请输入要连接的机器编号：")
	fmt.Scanln(&number)
	length := len(sshConfigs) - 1
	if !(number >= 0 && number <= length) {
		fmt.Println("请输入指定机器编号区间[", 0, ",", length, "]")
		return ReadUserWrite(sshConfigs)
	}
	sshConfig := sshConfigs[number]
	fmt.Println("正在连接...", sshConfig.IP)
	return number
}
func Conn(ip string, port int, username string, password string) {
	cli := New(ip, username, password, port)

	err := cli.RunTerminal(os.Stdout, os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
}

func Run(filePath string) {
	if len(filePath) == 0 || filePath == "" {
		filePath = defaultSshConfigPath
	}
	sshConfigs := ReadSshConfig(filePath)
	number := ReadUserWrite(sshConfigs)
	sshConfig := sshConfigs[number]
	Conn(sshConfig.IP, sshConfig.Port, sshConfig.Username, sshConfig.Password)
}
