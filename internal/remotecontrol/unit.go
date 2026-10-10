package remotecontrol

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var notAllowedInName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func unitName(repo, name string) (string, error) {
	if name == "" {
		name = filepath.Base(filepath.Clean(repo))
	}
	cleaned := strings.Trim(notAllowedInName.ReplaceAllString(name, "-"), "-")
	if cleaned == "" {
		return "", errors.New("no usable name, pick one with -name")
	}
	return cleaned, nil
}

func unitFileName(name string) string {
	return unitPrefix + name + unitSuffix
}

func unitFileContent(repo, name, claudePath string) string {
	repo = escapePercent(repo)
	claudePath = escapePercent(claudePath)
	return fmt.Sprintf(`[Unit]
Description=Claude Remote Control for %[2]s (%[1]s)

[Service]
WorkingDirectory=%[1]s
ExecStart=%[3]s remote-control --name %%H-%[2]s --permission-mode default
Restart=always
RestartSec=60

[Install]
WantedBy=default.target
`, repo, name, claudePath)
}

func escapePercent(s string) string {
	return strings.ReplaceAll(s, "%", "%%")
}
