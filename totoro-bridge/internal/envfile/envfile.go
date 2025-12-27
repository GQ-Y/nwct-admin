package envfile

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed env.example
var embedded embed.FS

// Bootstrap 会在 exe 同目录下：
// - 若不存在 .env，则写入内置 env.example
// - 若存在 .env，则加载其中未被外部环境设置的变量
func Bootstrap() {
	exe, err := os.Executable()
	if err != nil {
		_ = ensureAndLoad(filepath.Join(".", ".env"))
		return
	}
	exeDir := filepath.Dir(exe)
	_ = ensureAndLoad(filepath.Join(exeDir, ".env"))
}

func ensureAndLoad(dotenvPath string) error {
	if _, err := os.Stat(dotenvPath); err != nil {
		if os.IsNotExist(err) {
			b, rerr := embedded.ReadFile("env.example")
			if rerr == nil && len(bytes.TrimSpace(b)) > 0 {
				_ = os.MkdirAll(filepath.Dir(dotenvPath), 0o755)
				_ = os.WriteFile(dotenvPath, b, 0o644)
			}
		}
	}
	return Load(dotenvPath)
}

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


