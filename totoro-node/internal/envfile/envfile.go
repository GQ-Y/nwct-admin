package envfile

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// envExample 内置模板（避免依赖 go:embed 在某些编辑器/诊断链路下的误报）
const envExample = `# totoro-node 默认环境变量模板（会在首次运行时自动写入到二进制同目录的 .env）
#
# 说明：
# - 你可以直接启动 totoro-node，不必手工设置 node_id/node_key（程序会自动生成并持久化）
# - 如需覆盖默认值，编辑同目录 .env 或在系统环境变量中设置即可

# 节点管理 API 监听地址
TOTOTO_NODE_API_ADDR=:18080

# 节点管理面板/API 鉴权（可留空表示不校验；生产建议设置）
TOTOTO_NODE_ADMIN_KEY=

# 节点本地数据库路径
TOTOTO_NODE_DB=./node.db

# frps 配置文件路径
TOTOTO_FRPS_CONFIG=./frps.toml

# 节点对外公开地址（用于自动补全 endpoints/node_api，生产建议设置公网域名或 IP）
TOTOTO_NODE_PUBLIC_ADDR=127.0.0.1

# 节点管理 API 对外地址（可选；默认用 TOTOTO_NODE_PUBLIC_ADDR + 管理端口拼出来）
TOTOTO_NODE_PUBLIC_API=

# 桥梁地址（可选；未设置时会使用程序内置默认值，或编译时 -ldflags 覆盖的默认值）
TOTOTO_BRIDGE_URL=
`

// Bootstrap 会在 exe 同目录下：
// - 若不存在 .env，则写入内置 env.example（便于“开箱即用”部署）
// - 若存在 .env，则加载其中未被外部环境设置的变量
func Bootstrap() {
	exe, err := os.Executable()
	if err != nil {
		// 退化：使用当前工作目录
		_ = ensureAndLoad(filepath.Join(".", ".env"))
		return
	}
	exeDir := filepath.Dir(exe)
	_ = ensureAndLoad(filepath.Join(exeDir, ".env"))
}

func ensureAndLoad(dotenvPath string) error {
	if _, err := os.Stat(dotenvPath); err != nil {
		if os.IsNotExist(err) {
			// 写入模板
			b := []byte(envExample)
			if len(bytes.TrimSpace(b)) > 0 {
				_ = os.MkdirAll(filepath.Dir(dotenvPath), 0o755)
				// 0644 足够；如果你希望更严格，可改成 0600
				_ = os.WriteFile(dotenvPath, b, 0o644)
			}
		}
	}
	return Load(dotenvPath)
}

// Load 解析 dotenv（KEY=VALUE），只会 set 尚未在外部环境存在的键。
func Load(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 支持 export KEY=VALUE
		line = strings.TrimSpace(strings.TrimPrefix(line, "export"))
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		v = strings.Trim(v, `"'`)
		if k == "" {
			continue
		}
		if _, exists := os.LookupEnv(k); exists {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("setenv %s: %w", k, err)
		}
	}
	return sc.Err()
}
