package main

import (
	"fmt"
	"os"
	"os/exec"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Host     string `yaml:"DBHOST"`
	Port     string `yaml:"DBPORT"`
	Name     string `yaml:"DBNAME"`
	Acct     string `yaml:"DBACCT"`
	Password string `yaml:"DBPSWD"`
}

// SOPSを使って暗号化されたYAMLファイルを復号し、構造体にパースする関数
func loadEncryptedConfig(filePath string) (*Config, error) {
	// sops --decrypt <ファイル名> を実行するコマンドを作成
	cmd := exec.Command("sops", "--decrypt", filePath)

	// 親の環境変数をベースに、必要な変数を確実に上書き/追加する
	env := os.Environ()
	// もし os.Getenv で取れているなら、それを明示的に詰め直す
	// if keyFile := os.Getenv("SOPS_AGE_KEY_FILE"); keyFile != "" {
	keyFile := os.Getenv("SOPS_AGE_KEY_FILE")
	if keyFile == "" {
		env = append(env, "SOPS_AGE_KEY_FILE="+"/home/chouette/.config/age/key.txt")
	}
	cmd.Env = env

	// 実行して標準出力を取得（メモリ上に復号されたYAMLが展開される）
	decryptedData, err := cmd.Output()
	if err != nil {
		// sopsコマンドの実行に失敗した場合、詳細なエラーを取得
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("sops error: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	// 復号されたYAMLデータを構造体にマッピング
	var cfg Config
	if err := yaml.Unmarshal(decryptedData, &cfg); err != nil {
		return nil, fmt.Errorf("yaml unmarshal error: %w", err)
	}

	return &cfg, nil
}
