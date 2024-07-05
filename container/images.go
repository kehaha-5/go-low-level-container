package container

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"text/tabwriter"

	"github.com/kehaha-5/go-low-level-container/common"
	"github.com/pkg/errors"
)

var (
	defaultImageInfoPath = common.ROOTPATH + "/images/"
	defaultImageFileName = "info.json"
)

type imageInfoItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Size       string `json:"size"`
	CreateTime string `json:"createTime"`
}

type imageInfos struct {
	Infos map[string]imageInfoItem `json:"infos"` //filename => imageInfoItem
}

func addImage(image *imageInfoItem) error {
	var infos imageInfos
	if err := infos.load(); err != nil {
		return errors.WithStack(err)
	}
	return infos.add(image)
}

func getSaveFilePath() string {
	return path.Join(defaultImageInfoPath, defaultImageFileName)
}

func (t *imageInfos) load() error {
	isExist, _ := common.PathExist(defaultImageInfoPath)
	if !isExist {
		if err := os.MkdirAll(defaultImageInfoPath, 0644); err != nil {
			return errors.Wrap(err, "fail to mkdir defaultImageInfoPath")
		}
	}
	if !common.FileExist(getSaveFilePath()) {
		t.Infos = make(map[string]imageInfoItem)
		return nil
	}

	jsonStr, err := os.ReadFile(getSaveFilePath())
	if err != nil {
		return errors.Wrap(err, "fail to read form file")
	}
	if err := json.Unmarshal(jsonStr, t); err != nil {
		return errors.Wrapf(err, "fail to unmarshal json %s ", string(jsonStr))
	}
	if len(t.Infos) == 0 {
		t.Infos = make(map[string]imageInfoItem)
	}
	return nil
}

func (t *imageInfos) dump() error {
	jsonStr, err := json.Marshal(t)
	if err != nil {
		return errors.Wrap(err, "fail to marshal image infos to json")
	}
	return recordToJsonfile(jsonStr)
}

func (t *imageInfos) add(image *imageInfoItem) error {
	_, isExist := t.Infos[image.Name]
	if isExist {
		return fmt.Errorf("the images name %s has existed ", image.Name)
	}
	t.Infos[image.Name] = *image
	return t.dump()
}

func recordToJsonfile(jsonStr []byte) error {
	isExist, err := common.PathExist(defaultImageInfoPath)
	if err != nil {
		return errors.Wrapf(err, "fail to judge the file path %s", defaultImageInfoPath)
	}
	if !isExist {
		if err := os.MkdirAll(defaultImageInfoPath, 0644); err != nil {
			return errors.Wrap(err, "fail to mkdir")
		}
	}

	return errors.Wrap(os.WriteFile(getSaveFilePath(), jsonStr, 0644), "fail to write data to file ")
}

func WirteImagesInfoToTabwriter(w *tabwriter.Writer) error {
	var infos imageInfos
	if err := infos.load(); err != nil {
		return errors.WithStack(err)
	}
	for _, item := range infos.Infos {
		// "ID\tNAME\tSize\tCREATED\n"
		fmt.Fprintf(
			w, "%s\t%s\t%s\t%s\n",
			item.ID,
			item.Name,
			item.Size,
			item.CreateTime,
		)
	}
	return nil
}

func DelImageByName(names []string) error {
	var infos imageInfos
	if err := infos.load(); err != nil {
		return errors.WithStack(err)
	}
	for _, name := range names {
		_, isExist := infos.Infos[name]
		if !isExist {
			slog.Info(fmt.Sprintf("delete image name %s not existed", name))
			continue
		}
		delete(infos.Infos, name)
		fmt.Println(name)
	}
	return infos.dump()
}
