# Baize — Hardware Information Collector

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-green)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux-lightgrey)]()

**Baize（白泽）** 是一款面向 Linux 服务器的轻量级硬件信息采集工具，能够并发采集 CPU、内存、RAID 控制器、网络接口、GPU 等多维度硬件信息，支持终端格式化输出和 JSON 输出两种模式。

---

## 目录

- [功能特性](#功能特性)
- [系统要求](#系统要求)
- [安装](#安装)
- [快速开始](#快速开始)
- [命令行参数](#命令行参数)
- [采集模块](#采集模块)
- [输出示例](#输出示例)
- [项目结构](#项目结构)
- [架构设计](#架构设计)
- [外部依赖](#外部依赖)
- [开发指南](#开发指南)
- [License](#license)

---

## 功能特性

- 🚀 **并发采集**：所有模块通过 goroutine 并发执行，总采集时间 ≤ 2 秒（受限于最慢的单个命令）
- 🖥️ **多维度覆盖**：CPU、内存、RAID 控制器、NVMe、网络接口、Bond、GPU、服务器基本信息、硬件健康状态
- 📊 **双输出模式**：终端彩色格式化（简要/详细）+ JSON 机器可读输出
- 🔌 **SMBIOS 原生解析**：直接读取 `/sys/firmware/dmi/tables`，无需依赖 `dmidecode` 二进制（也支持 `dmidecode` 回退）
- 🗂️ **多厂商 RAID 支持**：LSI/Broadcom（MegaRAID）、HPE（SmartArray）、Adaptec、Intel VROC
- 🔍 **SMART 健康检测**：通过 `smartctl` 获取物理磁盘 SMART 属性，自动诊断故障风险
- 🌐 **LLDP 拓扑感知**：采集上联交换机端口信息，辅助网络拓扑可视化

---

## 系统要求

| 项目 | 要求 |
|------|------|
| 操作系统 | Linux（推荐 CentOS 8 / RHEL 8 / Ubuntu 20.04+） |
| Go 版本 | 1.24 及以上（编译时） |
| 运行权限 | **root** 或具有相应硬件访问权限的用户 |
| 内核版本 | 4.15+（EDAC、hwmon sysfs 接口） |

---

## 安装

### 从源码编译

```bash
git clone https://github.com/zenithax-cc/baize.git
cd baize
go build -o baize ./cmd/terminal
```

### 直接运行（无需安装）

```bash
go run ./cmd/terminal [flags]
```

---

## 快速开始

```bash
# 采集全部模块，终端简要输出（默认）
sudo ./baize

# 采集全部模块，终端详细输出
sudo ./baize -d

# 采集全部模块，JSON 输出
sudo ./baize -j

# 只采集 CPU 模块
sudo ./baize -m cpu

# 只采集内存，输出 JSON
sudo ./baize -m memory -j
```

---

## 命令行参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-m` | string | `all` | 指定采集模块名称，`all` 表示全部模块 |
| `-j` | bool | `false` | 以 JSON 格式输出结果 |
| `-d` | bool | `false` | 输出详细视图（包含每条 DIMM / 每个线程等） |

### 可用模块名称

| 模块名 | 说明 |
|--------|------|
| `product` | 服务器基本信息（厂商、型号、序列号、OS） |
| `cpu` | CPU 频率、温度、功耗、超线程状态 |
| `memory` | 内存容量、DIMM 详情、EDAC 错误计数 |
| `raid` | RAID 控制器、逻辑盘、物理盘、NVMe |
| `network` | 网络接口、驱动、速率、IPv4、LLDP |
| `bond` | Bond 聚合接口配置及成员状态 |
| `gpu` | GPU 设备信息 |
| `ipmi` | BMC 信息、传感器、电源、系统事件日志 |
| `health` | 硬件健康状态汇总 |

---

## 采集模块

### product — 服务器基本信息

- **数据来源**：SMBIOS Type 1/2/3（系统、主板、机箱）、`/etc/os-release`
- **采集内容**：服务器厂商、产品名称、序列号、UUID、BIOS 版本、操作系统版本

### cpu — 处理器

- **数据来源**：`lscpu`、SMBIOS Type 4、`turbostat`、`/sys/class/hwmon`
- **采集内容**：
  - 型号、架构、Socket 数、核心数、线程数、超线程状态
  - 基础频率、最大/最小实时频率（turbostat 1 秒采样）
  - 封装温度（℃）、封装功耗（W）
  - 电源状态（Performance / Powersave）
  - 缓存大小（L1d / L1i / L2 / L3）
  - 每核心详细线程信息（详细模式）

### memory — 内存

- **数据来源**：SMBIOS Type 17、`/proc/meminfo`、`/sys/bus/edac`
- **采集内容**：
  - 物理内存总量、插槽数（最大/已用）
  - 系统内存使用情况（Total / Free / Available / Buffer / Cache）
  - Swap 配置
  - 每条 DIMM 信息（型号、序列号、速率、电压、容量）
  - EDAC 可纠正/不可纠正错误计数

### raid — 存储控制器

- **数据来源**：`storcli`（LSI/Broadcom）、`hpssacli`/`ssacli`（HPE）、`arcconf`（Adaptec）、`ipmitool`（Intel VROC）、`smartctl`
- **采集内容**：
  - 控制器型号、固件版本、缓存大小、健康状态
  - 逻辑盘（RAID 级别、容量、状态、缓存策略）
  - 物理盘（厂商、型号、SN、容量、接口速率、SMART 状态）
  - BBU / CacheVault 电池状态
  - NVMe 设备独立采集

### network — 网络

- **数据来源**：`/sys/class/net`、`ethtool`、`lldpctl`、`/proc/net`、PCI 设备扫描
- **采集内容**：
  - 所有网口基本信息（MAC、驱动版本、固件版本、速率、双工、MTU、Link 状态）
  - IPv4 地址、网关
  - 环形缓冲区（Ring Buffer）当前/最大配置
  - 网卡队列（Channel）配置
  - LLDP 上联交换机信息（ToR MAC、主机名、管理 IP、端口、VLAN）
  - PCI 设备详情

### bond — 聚合链路

- **数据来源**：`/proc/net/bonding`
- **采集内容**：Bond 模式、LACP 速率、哈希策略、MII 状态、成员接口状态及错误计数

### ipmi — 带外管理（BMC）

- **数据来源**：`ipmitool bmc info`、`ipmitool lan print`、`ipmitool sensor`、`ipmitool sdr`、`ipmitool dcmi`、`ipmitool sel elist`
- **采集内容**：
  - BMC 设备 ID、固件版本、IPMI 规范版本
  - BMC 管理网络接口（IP、MAC、网关、子网掩码）
  - 全部传感器读数，按类型分组（温度 / 电压 / 风扇转速 / 电流）
  - 电源模块（PSU）在位状态及 DCMI 系统瞬时功耗
  - 系统事件日志（SEL）过滤：仅保留含 `error/critical/fault` 等关键字的告警条目，并自动打 Critical / Warning / Info 级别标签
  - 自动诊断：综合 SEL 告警、异常传感器、PSU 故障，输出 `OK` 或 `WARNING + 详情`

---

## 输出示例

### 终端简要模式

```
[Product]
  Vendor        : Dell Inc.
  Model         : PowerEdge R750
  Serial        : XXXXXXX
  OS            : Red Hat Enterprise Linux 8.6

[CPU]
  Model         : Intel(R) Xeon(R) Gold 6338 CPU @ 2.00GHz
  Vendor        : Intel
  Socket(s)     : 2
  Cores Per Soket: 32
  Threads Per Core: 2
  Hyper Threading: Enabled
  Power State   : Performance
  Frequency     : 2000 MHz
  Temperature   : 42 °C
  Watt          : 185.32 W

[Memory]
  Physical Memory: 512 GiB
  Slot Max      : 32
  Slot Used     : 16
  System Memory : 515396 MB
  ...
```

### JSON 模式（`-j`）

```json
{
  "product": {
    "vendor": "Dell Inc.",
    "model": "PowerEdge R750",
    "serial_number": "XXXXXXX"
  },
  "cpu": {
    "model_name": "Intel(R) Xeon(R) Gold 6338 CPU @ 2.00GHz",
    "sockets": "2",
    "power_state": "Performance",
    "based_freq_mhz": "2000 MHz",
    "temperature_celsius": "42 °C",
    "watt": "185.32 W"
  },
  ...
}
```

---

## 项目结构

```
baize/
├── cmd/
│   └── terminal/          # CLI 入口（main.go）
├── internal/
│   └── collector/
│       ├── cpu/           # CPU 采集（lscpu + SMBIOS + turbostat + hwmon）
│       ├── memory/        # 内存采集（SMBIOS + /proc/meminfo + EDAC）
│       ├── network/       # 网络采集（sysfs + ethtool + LLDP + Bond）
│       ├── raid/          # RAID 采集（LSI / HPE / Adaptec / Intel / NVMe）
│       ├── gpu/           # GPU 采集
│       ├── ipmi/          # IPMI 采集（BMC / 传感器 / 电源 / SEL）
│       ├── health/        # 健康状态汇总
│       ├── pci/           # PCI 设备扫描（lspci + pci.ids）
│       ├── product/       # 服务器基本信息
│       └── smbios/        # SMBIOS 原生二进制解析
├── pkg/
│   ├── collector/         # Manager 编排层（并发调度 + 输出控制）
│   ├── execute/           # 外部命令执行封装
│   └── utils/             # 通用工具函数
├── go.mod
├── go.sum
└── README.md
```

---

## 架构设计

### Collector 接口

所有硬件模块均实现统一的 `Collector` 接口：

```go
type Collector interface {
    Name()          string          // 模块名称
    Collect(context.Context) error  // 数据采集
    BriefPrintln()                  // 终端简要输出
    DetailPrintln()                 // 终端详细输出
    JSON() error                    // JSON 输出
}
```

### 并发采集流程

```
NewManager()
    │
    ├── SetModule()          — 按 -m 参数注册模块
    │
    └── Collect()
          │
          ├── goroutine: product.Collect()  ─┐
          ├── goroutine: cpu.Collect()      ─┤
          ├── goroutine: memory.Collect()   ─┼─► resultsCh (buffered channel)
          ├── goroutine: raid.Collect()     ─┤
          ├── goroutine: network.Collect()  ─┤
          └── goroutine: gpu.Collect()      ─┘
                │
                ▼ (wg.Wait → close channel)
          顺序输出（按注册顺序保证一致性）
```

所有模块并发执行，执行完毕后按固定注册顺序输出，确保输出的确定性。

---

## 外部依赖

| 工具 | 用途 | 必需 |
|------|------|------|
| `turbostat` | CPU 实时频率 / 功耗采集 | 建议（无则跳过） |
| `lscpu` | CPU 架构信息 | 建议 |
| `storcli` / `storcli64` | LSI/Broadcom RAID 管理 | 可选 |
| `ssacli` / `hpssacli` | HPE SmartArray RAID 管理 | 可选 |
| `arcconf` | Adaptec RAID 管理 | 可选 |
| `ipmitool` | Intel VROC / BMC 信息 | 可选 |
| `smartctl` | 磁盘 SMART 数据 | 可选 |
| `ethtool` | 网卡硬件参数 | 建议 |
| `lldpctl` | LLDP 邻居发现 | 可选 |

> **注意**：工具缺失时对应模块会跳过采集并记录警告日志，不影响其他模块正常运行。

---

## 开发指南

### 新增采集模块

1. 在 `internal/collector/<module>/` 下创建模块目录
2. 实现 `Collector` 接口的全部方法
3. 在 `pkg/collector/collect.go` 的 `supportedModules` 中注册新模块：

```go
var supportedModules = []struct {
    module    moduleType
    collector Collector
}{
    // ... 已有模块 ...
    {ModuleTypeMyModule, mymodule.New()},
}
```

### 运行测试

```bash
go test ./...
```

### 代码规范

- 所有导出函数和类型须包含英文 doc comment
- 错误处理遵循 `fmt.Errorf("context: %w", err)` 惯例
- 外部命令调用统一使用 `pkg/execute` 封装，支持 context 超时控制

---

## License

本项目使用 [Apache License 2.0](LICENSE) 开源协议。