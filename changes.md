# 变更日志

## 2026-10-24

### 新增 `--check` 连通性检查

**文件**: `ssh/check.go`(新)、`ssh/connect.go`、`ssh/run.go`、`ssh/auth.go`、`sshlib/connect.go`、`app/ssh.go`

#### 用法

```bash
bssh --check                         # 检查全部登记主机，输出可达列表
bssh --check -H host-1,host-2        # 只检查指定主机
bssh --check --check-all             # 不可达主机也一并输出
bssh --check --check-timeout 1       # 单台超时（默认 3s）
bssh --check --check-parallel 50     # 并发度（默认 32）
bssh --check --check-mode tcp        # 只探测端口，不做 SSH 认证
bssh --check --check-strict          # 存在不可达主机时退出码为 1
```

结果表列出 `SERVER NAME / CONNECT INFO / STATUS / LATENCY / REASON / NOTE`，
末尾给出可直接复制的 `bssh -H <可达主机...> <command>`。

#### 实现要点

1. **并发探测**：`Run.CheckHosts` 用 `WaitGroup` + 信号量并发，结果按输入顺序输出。
   认证方式（`CreateAuthMethodMap`）与 ssh-agent（`SetupSSHAgent`）都在并发前准备好，避免 data race。
2. **只做连接**：不创建 session、不执行远程命令，成功后立即关闭连接。
3. **超时必须双覆盖**：`ssh.ClientConfig.Timeout` 只作用于 `ssh.Dial`，`NewClientConn` 路径下不约束握手，
   因此在 `sshlib.Connect` 上新增 `HandshakeTimeout`，握手期间设置 conn deadline，成功后清除。
   同时 `--check-timeout` 覆盖 TCP 拨号超时（原固定 10s）。
4. **无副作用**：`{Prompt}` 密码在 check 模式下不再交互式输入；`autoEncryptPwd` 在 check 模式下不回写配置文件。

## 2026-01-12

### 重构 IP2 环境变量处理逻辑

**文件**: `ssh/connect.go`

#### 变更内容

1. **代码重构**
   - 将原来位于 `CreateSSHConnect` 函数中的 IP2 环境变量处理逻辑（L88-L104）提取为独立函数
   - 新增函数：
     - `resolveIP2Override(addr, port string) (string, string)` - 处理 IP2 环境变量覆盖逻辑
     - `mergePartialIP(originalAddr, partialIP string) string` - 合并部分 IP 地址

2. **新增功能：支持部分 IP**
   - 之前只支持完整 IP 地址替换
   - 现在支持部分 IP 段替换，方便快速修改 IP 的某几段
   
   **示例**：
   - 原始地址：`192.168.230.33`
   - `IP2=34` → 结果：`192.168.230.34`（替换最后1段）
   - `IP2=230.34` → 结果：`192.168.230.34`（替换最后2段）
   - `IP2=168.230.34` → 结果：`192.168.230.34`（替换最后3段）

3. **支持的所有格式**
   - `:port` - 仅覆盖端口（如 `:2222`）
   - `addr:port` - 覆盖地址和端口（如 `192.168.1.100:2222`）
   - `full.ip.address` - 完整 IP 地址，覆盖地址（如 `192.168.1.100`）
   - `partial.ip` - **新增**：部分 IP，与原始地址合并（如 `34` 或 `230.34`）

4. **代码优化**
   - 简化了条件分支逻辑
   - 统一了日志输出格式：`replace {原地址}:{原端口} by $IP2: {新地址}:{新端口}`
   - 提高了代码可读性和可维护性

#### 应用场景

主要用于应对金良小主机 IP 经常发生变化的场景。通过设置 `IP2` 环境变量，可以快速覆盖配置文件中的 IP 地址和端口，无需修改配置文件。

#### 技术细节

部分 IP 的合并采用**从右向左**的替换策略：
- 将部分 IP 按 `.` 分割后，从右向左依次替换原始 IP 的对应段
- 例如：`partialIP="230.34"` 会替换原始地址的倒数第2段和最后1段
