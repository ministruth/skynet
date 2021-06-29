package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	plugins "skynet/plugin"
	monitor "skynet/plugin/monitor/shared"
	task "skynet/plugin/task/shared"
	"skynet/sn"
	"skynet/sn/utils"
	"strconv"
	"strings"
	"time"
)

var (
	PlatformNotSupportError = errors.New("Platform not supported")
)

func UninstallTask(ctx context.Context, base int, aid int, tid int) error {
	t := sn.Skynet.SharedData["plugin_c1e81895-1f75-4988-9f10-52786b875ec7"].(task.PluginShared)
	m := sn.Skynet.SharedData["plugin_2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"].(monitor.PluginShared)
	_, output, err := m.RunCMDSync(aid, "bash "+m.GetPluginPath(Config, "linux.sh uninstall"), time.Second*10)
	if err != nil {
		return err
	}
	output = strings.TrimSuffix(output, "\n\nTask exit with code: 0")
	err = t.AppendOutputNewLine(tid, output)
	if err != nil {
		return err
	}
	err = t.AddPercent(tid, 80*base/100)
	if err != nil {
		return err
	}

	err = t.AppendOutputNewLine(tid, "Updating database...")
	if err != nil {
		return err
	}
	err = m.DeleteSetting(aid, plugins.SPWithIDPrefix(Config, "install"))
	if err != nil {
		return err
	}
	err = m.DeleteSetting(aid, plugins.SPWithIDPrefix(Config, "version"))
	if err != nil {
		return err
	}
	err = m.DeleteSetting(aid, plugins.SPWithIDPrefix(Config, "time"))
	if err != nil {
		return err
	}
	err = t.AppendOutputNewLine(tid, "Success")
	if err != nil {
		return err
	}
	err = t.AddPercent(tid, 20*base/100)
	if err != nil {
		return err
	}
	err = t.UpdateStatus(tid, task.TaskSuccess)
	if err != nil {
		return err
	}
	return nil
}

func InstallTask(ctx context.Context, base int, aid int, tid int) error {
	t := sn.Skynet.SharedData["plugin_c1e81895-1f75-4988-9f10-52786b875ec7"].(task.PluginShared)
	m := sn.Skynet.SharedData["plugin_2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"].(monitor.PluginShared)
	err := t.AppendOutputNewLine(tid, "Downloading checksums...")
	if err != nil {
		return err
	}
	agent := m.GetAgents()[aid]
	res, err := getGottyInfo()
	if err != nil {
		return err
	}
	err = utils.DownloadTempFile(ctx, "https://github.com/yudai/gotty/releases/download/"+res["tag_name"].(string)+"/SHA256SUMS",
		plugins.SPWithIDPrefixTempPath(Config, "SHA256SUMS"), "")
	if err != nil {
		return err
	}
	file, err := ioutil.ReadFile(plugins.SPWithIDPrefixTempPath(Config, "SHA256SUMS"))
	if err != nil {
		return err
	}
	fileLine := strings.Split(strings.TrimSpace(string(file)), "\n")
	fileList := make(map[string]string)
	for _, v := range fileLine {
		tmp := strings.Split(v, "  ")
		fileList[tmp[1]] = tmp[0]
	}
	err = t.AddPercent(tid, 20*base/100)
	if err != nil {
		return err
	}

	if agent.Machine == "x86_64" && strings.HasPrefix(agent.System, "Linux") {
		err = t.AppendOutputNewLine(tid, "Checksum loaded\nDownloading gotty_linux_amd64.tar.gz...")
		if err != nil {
			return err
		}
		err = utils.DownloadTempFile(ctx, "https://github.com/yudai/gotty/releases/download/"+res["tag_name"].(string)+"/gotty_linux_amd64.tar.gz",
			plugins.SPWithIDPrefixTempPath(Config, "gotty_linux_amd64.tar.gz"), fileList["gotty_linux_amd64.tar.gz"])
		if err != nil {
			return err
		}
		err = t.AddPercent(tid, 20*base/100)
		if err != nil {
			return err
		}

		err = t.AppendOutputNewLine(tid, "File downloaded\nSending file to agent...")
		if err != nil {
			return err
		}
		err = m.WriteFile(aid, m.GetPluginPath(Config, "linux.sh"), Config.Path+"linux.sh", true, true, 0755, time.Second*3)
		if err != nil {
			return err
		}
		err = m.WriteFile(aid, m.GetPluginPath(Config, "gotty_linux_amd64.tar.gz"), plugins.SPWithIDPrefixTempPath(Config, "gotty_linux_amd64.tar.gz"), true, true, 0755, time.Second*3)
		if err != nil {
			return err
		}
		err = t.AddPercent(tid, 20*base/100)
		if err != nil {
			return err
		}

		err = t.AppendOutputNewLine(tid, "File sended")
		if err != nil {
			return err
		}
		_, output, err := m.RunCMDSync(aid, "bash "+m.GetPluginPath(Config, "linux.sh install"), time.Second*10)
		if err != nil {
			return err
		}
		output = strings.TrimSuffix(output, "\n\nTask exit with code: 0")
		err = t.AppendOutputNewLine(tid, output)
		if err != nil {
			return err
		}
		err = t.AddPercent(tid, 20*base/100)
		if err != nil {
			return err
		}

		err = t.AppendOutputNewLine(tid, "Updating database...")
		if err != nil {
			return err
		}
		err = m.NewSetting(aid, plugins.SPWithIDPrefix(Config, "install"), "1")
		if err != nil {
			return err
		}
		err = m.NewSetting(aid, plugins.SPWithIDPrefix(Config, "version"), res["tag_name"].(string)[1:])
		if err != nil {
			return err
		}
		err = m.NewSetting(aid, plugins.SPWithIDPrefix(Config, "time"), strconv.FormatInt(time.Now().Unix(), 10))
		if err != nil {
			return err
		}
		err = t.AppendOutputNewLine(tid, "Success")
		if err != nil {
			return err
		}
		err = t.AddPercent(tid, 20*base/100)
		if err != nil {
			return err
		}
		err = t.UpdateStatus(tid, task.TaskSuccess)
		if err != nil {
			return err
		}
	} else {
		return PlatformNotSupportError
	}
	return nil
}

func getGottyInfo() (map[string]interface{}, error) {
	resp, err := http.Get("https://api.github.com/repos/yudai/gotty/releases/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func GetVersion() (string, error) {
	res, err := getGottyInfo()
	if err != nil {
		return "", err
	}
	return res["tag_name"].(string), nil
}
