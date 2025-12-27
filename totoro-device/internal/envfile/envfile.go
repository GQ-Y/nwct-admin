package envfile

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// 设备端部署时“少命令行”：自动在固定位置生成/加载 .env
//
// - Linux 设备：/etc/nwct/.env
// - 其它平台：二进制同目录 .env（便于本地调试）
func Bootstrap() {
	path := ""
	if runtime.GOOS == "linux" {
		path = "/etc/nwct/.env"
	} else {
		exe, err := os.Executable()
		if err != nil {
			path = filepath.Join(".", ".env")
		} else {
			path = filepath.Join(filepath.Dir(exe), ".env")
		}
	}
	_ = ensureAndLoad(path)
}

func ensureAndLoad(dotenvPath string) error {
	if _, err := os.Stat(dotenvPath); err != nil {
		if os.IsNotExist(err) {
			b := []byte(envExample)
			if len(bytes.TrimSpace(b)) > 0 {
				_ = os.MkdirAll(filepath.Dir(dotenvPath), 0o755)
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

const envExample = `# totoro-device 默认环境变量模板（首次运行会自动写入）
#
# 说明：
# - 桥梁地址优先级：config.json 的 bridge.url > TOTOTO_BRIDGE_URL > 内置 DefaultBridgeURL
# - 未设置时，会默认使用内置：http://192.168.2.32:18090

# 桥梁地址（可选）
TOTOTO_BRIDGE_URL=

# 配置文件路径（可选；默认 Linux: /etc/nwct/config.json）
NWCT_CONFIG_PATH=
`
