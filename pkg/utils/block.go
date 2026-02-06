package utils

import (
	"bytes"
	"os"
	"strings"
	"sync"

	"github.com/zenithax-cc/baize/pkg/execute"
)

const (
	sysfsBlock  = "/sys/block"
	devDiskByID = "/dev/disk/by-id"
	lsblk       = "/bin/lsblk"

	wwnPrefix  = "wwn-"
	partSuffix = "-part"
)

type blockCache struct {
	once    sync.Once
	blocks  []string
	wwnMap  map[string]string
	initErr error
}

var capcityCache = &blockCache{}

func (b *blockCache) init() {
	b.once.Do(func() {
		b.blocks = b.loadBlocks()
		b.wwnMap = b.loadWWNMap()
	})
}

func (b *blockCache) loadBlocks() []string {
	devices, err := os.ReadDir(sysfsBlock)
	if err != nil {
		return GetBlockByLsblk()
	}

	blocks := make([]string, 0, len(devices))
	for _, device := range devices {
		name := device.Name()
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "md") {
			continue
		}
		blocks = append(blocks, name)
	}

	return blocks
}

func GetBlockByLsblk() []string {
	output := execute.Command(lsblk, "-d", "-o", "NAME", "-n")
	if output.Err != nil {
		return nil
	}

	lines := bytes.Split(output.Stdout, []byte("\n"))
	blocks := make([]string, 0, len(lines))

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		blocks = append(blocks, string(line))
	}
	return blocks
}

func (b *blockCache) loadWWNMap() map[string]string {
	files, err := os.ReadDir(devDiskByID)
	if err != nil {
		return nil
	}

	wwnMap := make(map[string]string)
	for _, file := range files {
		fn := file.Name()
		if !strings.HasPrefix(fn, wwnPrefix) || strings.Contains(fn, partSuffix) {
			continue
		}

		key := parseWWN(fn)
		if value, err := ReadLinkBase(fn); err == nil {
			wwnMap[key] = value
		}
	}

	return wwnMap
}

func parseWWN(fn string) string {
	if idx := strings.IndexByte(fn, '-'); idx != -1 {
		return fn[idx+1:]
	}
	return fn
}

func GetOneBlock() string {
	capcityCache.init()

	if len(capcityCache.blocks) > 0 {
		return capcityCache.blocks[0]
	}

	return "sda"
}

func GetAllBlock() []string {
	capcityCache.init()

	res := make([]string, 0, len(capcityCache.blocks))
	copy(res, capcityCache.blocks)

	return res
}

func GetBlockByWWN(wwn string) string {
	capcityCache.init()
	if block, ok := capcityCache.wwnMap[wwn]; ok {
		return block
	}
	return ""
}

func RefreshCache() {
	capcityCache = &blockCache{}
	capcityCache.init()
}

func GetOneBlockWithRefresh(refresh bool) string {
	if refresh {
		RefreshCache()
	}
	return GetOneBlock()
}
