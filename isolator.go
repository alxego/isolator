package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/otiai10/copy"
)

type config struct {
	Data    string `json:"data,omitempty"`
	Root    string `json:"root,omitempty"`
	Command string `json:"command,omitempty"`
	Memlim  int    `json:"memlim,omitempty"`
}

func main() {

	if len(os.Args) >= 2 {
		if os.Args[1] == "run" && len(os.Args) >= 4 {
			run(os.Args[2], os.Args[3], os.Args[4:])
			return
		}
		panic("unknow command argunent" + os.Args[1])
	}

	conf, err := parseConfig()
	if err != nil {
		println(err.Error())
		return
	}

	err = execCommand(conf)
	if err != nil {
		println(err.Error())
		return
	}

}

// Parse json from config.json to config struct 
func parseConfig() (conf config, err error) {
	file, err := os.Open("config.json")
	if err != nil {
		return
	}
	defer file.Close()
	dec := json.NewDecoder(file)
	err = dec.Decode(&conf)
	if err != nil {
		return
	}
	return
}

// Create root and copy data to root
func prepare(dataDir, rootDir string) (err error) {
	err = syscall.Mkdir(rootDir, 0766)
	if err != nil {
		return
	}

	err = copy.Copy(dataDir, rootDir)
	if err != nil {
		return
	}

	err = os.Mkdir("/sys/fs/cgroup/memory/group2", 0777)
	if err != nil {
		return
	}

	return
}

// Remove root directory
func clear(rootDir string) (err error) {
	err = os.RemoveAll(rootDir)
	if err != nil {
		return
	}
	err = syscall.Rmdir("/sys/fs/cgroup/memory/group2")
	return
}

// Execute command from config
func execCommand(conf config) (err error) {
	cmd := exec.Command("/proc/self/exe", "run", conf.Root, conf.Command)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdin, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNET,
	}

	err = prepare(conf.Data, conf.Root)
	if err != nil {
		return
	}
	defer clear(conf.Root)

	err = cmd.Start()
	if err != nil {
		return
	}

	err = ioutil.WriteFile("/sys/fs/cgroup/memory/group2/memory.limit_in_bytes", []byte(strconv.Itoa(conf.Memlim)), 0666)
	if err != nil {
		return
	}

	err = ioutil.WriteFile("/sys/fs/cgroup/memory/group2/tasks", []byte(strconv.Itoa(cmd.Process.Pid)), 0666)
	if err != nil {
		return
	}

	err = cmd.Wait()
	if err != nil {
		return
	}

	return
}

// Run command in container
func run(rootDir, command string, args []string) {
	syscall.Chroot(rootDir)
	os.Chdir("/")
	println(os.Getpid())

	cmd := exec.Command(command, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdin, os.Stderr
	err := cmd.Run()
	if err != nil {
		println(err.Error())
	}
}

