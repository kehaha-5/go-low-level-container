package container

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

type ContainerInfos struct {
	Id         string `json:"id"`         //容器id
	Pid        string `json:"pid"`        //容器init进程在宿主机上的pid
	Name       string `json:"name"`       //容器名称
	Command    string `json:"command"`    //容器init进程执行的命令
	CreateTime string `json:"createTime"` //容器创建时间
	Status     string `json:"status"`     //容器状态
}

type Status string

const (
	Running             Status = "running"
	Stop                Status = "stopped"
	Exit                Status = "exited"
	defaultSavefilepath string = "/workspaces/go-simple-docker/runEnv/container/"
	defautlSavename     string = "config.json"
)

func GetConfigSavePath() string {
	return defaultSavefilepath
}

func (t *ContainerInfos) RecordContainerInfo(pid int, containerName string, command []string) (string, error) {
	createTime := time.Now().Format(time.RFC3339)

	if containerName == "" {
		containerName = t.Id
	}

	t.Pid = strconv.Itoa(pid)
	t.Name = containerName
	t.Command = strings.Join(command, " ")
	t.CreateTime = createTime
	t.Status = string(Running)

	jsonStr, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("json encode %v", err)
	}

	if err := t.recordToJsonfile(string(jsonStr)); err != nil {
		return "", err
	}

	return t.Name, nil

}

func (t *ContainerInfos) DeleteContainerInfo() {
	savefilepath := path.Join(defaultSavefilepath, t.Name)
	if err := os.RemoveAll(savefilepath); err != nil {
		slog.Error("DeleteContainerInfo", "err", err)
	}
}

func (t *ContainerInfos) WirteInfoToTabwriter(w *tabwriter.Writer) {
	// "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n"
	fmt.Fprintf(
		w, "%s\t%s\t%s\t%s\t%s\t%s\n",
		t.Id,
		t.Name,
		t.Pid,
		t.Status,
		t.Command,
		t.CreateTime,
	)
}

func (t *ContainerInfos) RandomContainerId(n int) {
	letterSeed := "0123456789abcde"
	rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letterSeed[rand.Intn(len(letterSeed))]
	}
	t.Id = string(b)
}

func (t *ContainerInfos) recordToJsonfile(jsonStr string) error {
	savefilepath := path.Join(defaultSavefilepath, t.Name)
	if err := os.MkdirAll(savefilepath, 0777); err != nil {
		return err
	}

	file, err := os.Create(path.Join(savefilepath, defautlSavename))

	if err != nil {
		return err
	}

	if _, err := file.WriteString(jsonStr); err != nil {
		return err
	}
	return nil
}

func GetInfoByContainerName(containerName string, data *ContainerInfos) error {
	savefilepath := path.Join(defaultSavefilepath, containerName, defautlSavename)
	info, err := os.ReadFile(savefilepath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(info, data); err != nil {
		return err
	}
	return nil
}
