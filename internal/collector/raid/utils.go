package raid

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/common/utils"
)

const (
	sysBlockPath    = "/sys/block"
	devDiskByIDPath = "/dev/disk/by-id"
	lsblk           = "/bin/lsblk"
)

var once sync.Once

// GetBlock 获取系统中其中一个块设备
func GetBlock() string {
	block := "sda"

	once.Do(func() {
		blocks := getBlockByPath()
		if len(blocks) > 0 {
			block = blocks[0]
		}
	})

	return block
}

// GetBlockByPath 获取系统中所有的块设备
func getBlockByPath() []string {
	devices, err := os.ReadDir(sysBlockPath)
	if err != nil {
		return getBlockByLsblk()
	}

	var blocks []string
	// 遍历sys/block目录下的所有文件
	for _, device := range devices {
		name := device.Name()
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "md") {
			continue
		}
		blocks = append(blocks, device.Name())
	}

	return blocks
}

// GetBlockByLsblk 获取系统中所有的块设备
func getBlockByLsblk() []string {
	output, err := utils.Run.Command("bash", "-c", fmt.Sprintf("%s -d -o NAME | grep -v NAME", lsblk))
	blocks := make([]string, 8)
	if err == nil {
		devices := bytes.Split(output, []byte("\n"))
		for _, device := range devices {
			blocks = append(blocks, string(device))
		}
	}

	return blocks
}

// GetBlockByWWN 获取系统中指定WWN的块设备
func GetBlockByWWN(wwn string) string {
	wwnMap := make(map[string]string)

	once.Do(
		func() {
			wwnMap = GetBlockByID()
		},
	)

	if block, ok := wwnMap[wwn]; ok {
		return block
	}

	return ""
}

// GetBlockByID 获取系统中所有的块设备，以wwn为键，块设备名称为值
func GetBlockByID() map[string]string {
	blocks := make(map[string]string)
	files, err := os.ReadDir(devDiskByIDPath)
	if err != nil {
		return blocks
	}

	for _, file := range files {
		fn := file.Name()
		if !strings.HasPrefix(fn, "wwn-") || strings.HasSuffix(fn, `-part*`) {
			continue
		}
		key := parseWWN(fn)
		value, _ := readLink(filepath.Join(devDiskByIDPath, fn))
		blocks[key] = value
	}

	return blocks
}

// parseWWN 解析WWN，返回WWN的后半部分
func parseWWN(s string) string {
	index := strings.IndexByte(s, '-')
	if index == -1 {
		return s
	}
	return s[index+1:]
}

// readLink 读取软链接，返回目标文件的名称
func readLink(path string) (string, error) {
	link, err := os.Readlink(path)
	if err != nil {
		return "Unkown", err
	}

	return filepath.Base(link), nil
}

func GetBlockByDevicesPath(devicesPath string) (string, error) {
	enttries, err := utils.ReadDir(sysBlockPath)
	if err != nil {
		return "", err
	}

	for _, entry := range enttries {
		devPath := filepath.Join(sysBlockPath, entry.Name())

		info, err := os.Readlink(devPath)
		if err != nil {
			continue
		}

		if strings.HasPrefix(info, devicesPath) {
			return DevicePrefix + filepath.Base(info), nil
		}
	}

	return "", fmt.Errorf("not found block device")
}
