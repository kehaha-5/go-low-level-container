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
	Id          string   `json:"id"`         //容器id
	Pid         string   `json:"pid"`        //容器init进程在宿主机上的pid
	Name        string   `json:"name"`       //容器名称
	Command     string   `json:"command"`    //容器init进程执行的命令
	CreateTime  string   `json:"createTime"` //容器创建时间
	Status      string   `json:"status"`     //容器状态
	Volume      string   `json:"volume"`
	PortMapping []string `json:"portMapping"`
}

const (
	Running                 string = "running"
	Stop                    string = "stopped"
	Exit                    string = "exited"
	defaultInfoSavefilepath string = "/root/runc/runEnv/info/"
	defaultInfoSavename     string = "config.json"
	defaultIdLen            int    = 10
)

func GetConfigSavePath() string {
	return defaultInfoSavefilepath
}

func (t *ContainerInfos) SetContainerName(containerName string) {
	t.randomContainerId(defaultIdLen)
	if containerName == "" {
		t.Name = t.Id
	} else {
		t.Name = containerName
	}
}

func (t *ContainerInfos) RecordContainerInfo(pid int, command []string, volumeArg string, protMappingStr string) (string, error) {
	tz, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		slog.Error("timezone to Asia/Shanghai", "err", err)
	}
	createTime := time.Now().In(tz).Format(time.RFC3339)

	t.Pid = strconv.Itoa(pid)
	t.Command = strings.Join(command, " ")
	t.CreateTime = createTime
	t.Status = Running
	t.Volume = volumeArg

	protMapping := strings.Split(protMappingStr, " ")
	if protMappingStr != "" && len(protMapping) != 0 {
		t.PortMapping = protMapping
	}

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
	savefilepath := path.Join(defaultInfoSavefilepath, t.Name)
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

func (t *ContainerInfos) randomContainerId(n int) {
	letterSeed := "0123456789abcde"
	rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letterSeed[rand.Intn(len(letterSeed))]
	}
	t.Id = string(b)
}

func (t *ContainerInfos) recordToJsonfile(jsonStr string) error {
	savefilepath := path.Join(defaultInfoSavefilepath, t.Name)
	if err := os.MkdirAll(savefilepath, 0777); err != nil {
		return err
	}

	file, err := os.Create(path.Join(savefilepath, defaultInfoSavename))

	if err != nil {
		return err
	}

	if _, err := file.WriteString(jsonStr); err != nil {
		return err
	}
	return nil
}

func (t *ContainerInfos) del() error {
	savefilepath := path.Join(defaultInfoSavefilepath, t.Name)
	return os.RemoveAll(savefilepath)
}

func GetInfoByContainerName(containerName string, data *ContainerInfos) error {
	savefilepath := path.Join(defaultInfoSavefilepath, containerName, defaultInfoSavename)
	info, err := os.ReadFile(savefilepath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(info, data); err != nil {
		return err
	}
	return nil
}

func getPidByContainerName(name string) (string, error) {
	data := ContainerInfos{}
	if err := GetInfoByContainerName(name, &data); err != nil {
		return "", err
	}
	return data.Pid, nil
}

func modifyContainerStatusToStopByName(name string) error {
	data := ContainerInfos{}
	if err := GetInfoByContainerName(name, &data); err != nil {
		return err
	}
	data.Status = Stop
	jsonStr, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return data.recordToJsonfile(string(jsonStr))
}
