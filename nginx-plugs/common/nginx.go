package common

import (
	"fmt"
	"net"
	"nginx-plugs/config"
	"nginx-plugs/model"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// ReloadAllNginx 通过SSH并发远程重载所有nginx服务器
// 返回每台服务器的reload结果
func ReloadAllNginx() []model.SSHReloadResult {
	targets := config.GetSSHTargets()
	if len(targets) == 0 {
		Logger.Warn("未配置ssh_targets，跳过nginx重载")
		return []model.SSHReloadResult{}
	}

	cmdConf := config.GetNginxCmdConfig()
	reloadCmd := cmdConf.Reload
	if reloadCmd == "" {
		reloadCmd = "nginx -s reload"
	}
	Logger.Infof("[RELOAD] 使用重载命令: %s", reloadCmd)

	// 并发执行多台服务器重载
	results := make([]model.SSHReloadResult, len(targets))
	var wg sync.WaitGroup
	for i, target := range targets {
		wg.Add(1)
		go func(idx int, t config.SSHTarget) {
			defer wg.Done()
			results[idx] = reloadOneServer(t, reloadCmd)
		}(i, target)
	}
	wg.Wait()

	return results
}

// reloadOneServer SSH到单台服务器执行reload
func reloadOneServer(target config.SSHTarget, reloadCmd string) model.SSHReloadResult {
	result := model.SSHReloadResult{
		Name: target.Name,
		Host: target.Host,
	}

	// 构建SSH客户端配置
	Logger.Infof("[SSH] 正在构建 %s(%s) SSH配置...", target.Name, target.Host)
	sshConfig, err := buildSSHConfig(target)
	if err != nil {
		result.Status = "failed"
		result.Error = "构建SSH配置失败: " + err.Error()
		Logger.Errorf("[SSH] %s(%s) 配置失败: %v", target.Name, target.Host, err)
		return result
	}

	// 连接SSH
	port := target.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(target.Host, strconv.Itoa(port))

	Logger.Infof("[SSH] 正在连接 %s(%s) ...", target.Name, addr)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		result.Status = "failed"
		result.Error = "SSH连接失败: " + err.Error()
		Logger.Errorf("[SSH] %s(%s) 连接失败: %v", target.Name, target.Host, err)
		return result
	}
	Logger.Infof("[SSH] %s(%s) 连接成功", target.Name, target.Host)

	// 执行reload命令
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		result.Status = "failed"
		result.Error = "创建SSH会话失败: " + err.Error()
		Logger.Errorf("[SSH] %s(%s) 会话失败: %v", target.Name, target.Host, err)
		return result
	}

	Logger.Infof("[SSH] %s(%s) 执行重载命令: %s", target.Name, target.Host, reloadCmd)
	output, err := session.CombinedOutput(reloadCmd)
	session.Close()
	if err != nil {
		client.Close()
		result.Status = "failed"
		result.Error = fmt.Sprintf("重载命令执行失败: %v, 输出: %s", err, strings.TrimSpace(string(output)))
		Logger.Errorf("[SSH] %s(%s) 重载失败: %v, 输出: %s", target.Name, target.Host, err, strings.TrimSpace(string(output)))
		return result
	}

	Logger.Infof("[SSH] %s(%s) 重载命令执行成功", target.Name, target.Host)

	// 断开SSH连接
	client.Close()
	Logger.Infof("[SSH] %s(%s) 已断开连接", target.Name, target.Host)

	result.Status = "success"
	return result
}

// buildSSHConfig 构建SSH客户端配置（支持密钥和密码两种认证方式）
func buildSSHConfig(target config.SSHTarget) (*ssh.ClientConfig, error) {
	sshConfig := &ssh.ClientConfig{
		User:            target.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	if target.User == "" {
		sshConfig.User = "root"
	}

	// 优先使用密钥认证
	if target.KeyPath != "" {
		key, err := os.ReadFile(target.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("读取SSH私钥失败: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("解析SSH私钥失败: %w", err)
		}
		sshConfig.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	} else if target.Password != "" {
		// 使用密码认证
		sshConfig.Auth = []ssh.AuthMethod{
			ssh.Password(target.Password),
		}
	} else {
		return nil, fmt.Errorf("未配置SSH认证方式（需要key_path或password）")
	}

	return sshConfig, nil
}
