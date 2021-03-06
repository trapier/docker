// +build linux

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/docker/docker/pkg/integration/checker"
	"github.com/docker/docker/pkg/mount"
	"github.com/go-check/check"
)

// TestDaemonRestartWithPluginEnabled tests state restore for an enabled plugin
func (s *DockerDaemonSuite) TestDaemonRestartWithPluginEnabled(c *check.C) {
	testRequires(c, IsAmd64, Network)

	s.d.Start(c)

	if out, err := s.d.Cmd("plugin", "install", "--grant-all-permissions", pName); err != nil {
		c.Fatalf("Could not install plugin: %v %s", err, out)
	}

	defer func() {
		if out, err := s.d.Cmd("plugin", "disable", pName); err != nil {
			c.Fatalf("Could not disable plugin: %v %s", err, out)
		}
		if out, err := s.d.Cmd("plugin", "remove", pName); err != nil {
			c.Fatalf("Could not remove plugin: %v %s", err, out)
		}
	}()

	s.d.Restart(c)

	out, err := s.d.Cmd("plugin", "ls")
	if err != nil {
		c.Fatalf("Could not list plugins: %v %s", err, out)
	}
	c.Assert(out, checker.Contains, pName)
	c.Assert(out, checker.Contains, "true")
}

// TestDaemonRestartWithPluginDisabled tests state restore for a disabled plugin
func (s *DockerDaemonSuite) TestDaemonRestartWithPluginDisabled(c *check.C) {
	testRequires(c, IsAmd64, Network)

	s.d.Start(c)

	if out, err := s.d.Cmd("plugin", "install", "--grant-all-permissions", pName, "--disable"); err != nil {
		c.Fatalf("Could not install plugin: %v %s", err, out)
	}

	defer func() {
		if out, err := s.d.Cmd("plugin", "remove", pName); err != nil {
			c.Fatalf("Could not remove plugin: %v %s", err, out)
		}
	}()

	s.d.Restart(c)

	out, err := s.d.Cmd("plugin", "ls")
	if err != nil {
		c.Fatalf("Could not list plugins: %v %s", err, out)
	}
	c.Assert(out, checker.Contains, pName)
	c.Assert(out, checker.Contains, "false")
}

// TestDaemonKillLiveRestoreWithPlugins SIGKILLs daemon started with --live-restore.
// Plugins should continue to run.
func (s *DockerDaemonSuite) TestDaemonKillLiveRestoreWithPlugins(c *check.C) {
	testRequires(c, IsAmd64, Network)

	s.d.Start(c, "--live-restore")
	if out, err := s.d.Cmd("plugin", "install", "--grant-all-permissions", pName); err != nil {
		c.Fatalf("Could not install plugin: %v %s", err, out)
	}
	defer func() {
		s.d.Restart(c, "--live-restore")
		if out, err := s.d.Cmd("plugin", "disable", pName); err != nil {
			c.Fatalf("Could not disable plugin: %v %s", err, out)
		}
		if out, err := s.d.Cmd("plugin", "remove", pName); err != nil {
			c.Fatalf("Could not remove plugin: %v %s", err, out)
		}
	}()

	if err := s.d.Kill(); err != nil {
		c.Fatalf("Could not kill daemon: %v", err)
	}

	cmd := exec.Command("pgrep", "-f", pluginProcessName)
	if out, ec, err := runCommandWithOutput(cmd); ec != 0 {
		c.Fatalf("Expected exit code '0', got %d err: %v output: %s ", ec, err, out)
	}
}

// TestDaemonShutdownLiveRestoreWithPlugins SIGTERMs daemon started with --live-restore.
// Plugins should continue to run.
func (s *DockerDaemonSuite) TestDaemonShutdownLiveRestoreWithPlugins(c *check.C) {
	testRequires(c, IsAmd64, Network)

	s.d.Start(c, "--live-restore")
	if out, err := s.d.Cmd("plugin", "install", "--grant-all-permissions", pName); err != nil {
		c.Fatalf("Could not install plugin: %v %s", err, out)
	}
	defer func() {
		s.d.Restart(c, "--live-restore")
		if out, err := s.d.Cmd("plugin", "disable", pName); err != nil {
			c.Fatalf("Could not disable plugin: %v %s", err, out)
		}
		if out, err := s.d.Cmd("plugin", "remove", pName); err != nil {
			c.Fatalf("Could not remove plugin: %v %s", err, out)
		}
	}()

	if err := s.d.Interrupt(); err != nil {
		c.Fatalf("Could not kill daemon: %v", err)
	}

	cmd := exec.Command("pgrep", "-f", pluginProcessName)
	if out, ec, err := runCommandWithOutput(cmd); ec != 0 {
		c.Fatalf("Expected exit code '0', got %d err: %v output: %s ", ec, err, out)
	}
}

// TestDaemonShutdownWithPlugins shuts down running plugins.
func (s *DockerDaemonSuite) TestDaemonShutdownWithPlugins(c *check.C) {
	testRequires(c, IsAmd64, Network, SameHostDaemon)

	s.d.Start(c)
	if out, err := s.d.Cmd("plugin", "install", "--grant-all-permissions", pName); err != nil {
		c.Fatalf("Could not install plugin: %v %s", err, out)
	}

	defer func() {
		s.d.Restart(c)
		if out, err := s.d.Cmd("plugin", "disable", pName); err != nil {
			c.Fatalf("Could not disable plugin: %v %s", err, out)
		}
		if out, err := s.d.Cmd("plugin", "remove", pName); err != nil {
			c.Fatalf("Could not remove plugin: %v %s", err, out)
		}
	}()

	if err := s.d.Interrupt(); err != nil {
		c.Fatalf("Could not kill daemon: %v", err)
	}

	for {
		if err := syscall.Kill(s.d.Pid(), 0); err == syscall.ESRCH {
			break
		}
	}

	cmd := exec.Command("pgrep", "-f", pluginProcessName)
	if out, ec, err := runCommandWithOutput(cmd); ec != 1 {
		c.Fatalf("Expected exit code '1', got %d err: %v output: %s ", ec, err, out)
	}

	s.d.Start(c, "--live-restore")
	cmd = exec.Command("pgrep", "-f", pluginProcessName)
	out, _, err := runCommandWithOutput(cmd)
	c.Assert(err, checker.IsNil, check.Commentf(out))
}

// TestVolumePlugin tests volume creation using a plugin.
func (s *DockerDaemonSuite) TestVolumePlugin(c *check.C) {
	testRequires(c, IsAmd64, Network)

	volName := "plugin-volume"
	destDir := "/tmp/data/"
	destFile := "foo"

	s.d.Start(c)
	out, err := s.d.Cmd("plugin", "install", pName, "--grant-all-permissions")
	if err != nil {
		c.Fatalf("Could not install plugin: %v %s", err, out)
	}
	pluginID, err := s.d.Cmd("plugin", "inspect", "-f", "{{.Id}}", pName)
	pluginID = strings.TrimSpace(pluginID)
	if err != nil {
		c.Fatalf("Could not retrieve plugin ID: %v %s", err, pluginID)
	}
	mountpointPrefix := filepath.Join(s.d.RootDir(), "plugins", pluginID, "rootfs")
	defer func() {
		if out, err := s.d.Cmd("plugin", "disable", pName); err != nil {
			c.Fatalf("Could not disable plugin: %v %s", err, out)
		}

		if out, err := s.d.Cmd("plugin", "remove", pName); err != nil {
			c.Fatalf("Could not remove plugin: %v %s", err, out)
		}

		exists, err := existsMountpointWithPrefix(mountpointPrefix)
		c.Assert(err, checker.IsNil)
		c.Assert(exists, checker.Equals, false)

	}()

	out, err = s.d.Cmd("volume", "create", "-d", pName, volName)
	if err != nil {
		c.Fatalf("Could not create volume: %v %s", err, out)
	}
	defer func() {
		if out, err := s.d.Cmd("volume", "remove", volName); err != nil {
			c.Fatalf("Could not remove volume: %v %s", err, out)
		}
	}()

	out, err = s.d.Cmd("volume", "ls")
	if err != nil {
		c.Fatalf("Could not list volume: %v %s", err, out)
	}
	c.Assert(out, checker.Contains, volName)
	c.Assert(out, checker.Contains, pName)

	mountPoint, err := s.d.Cmd("volume", "inspect", volName, "--format", "{{.Mountpoint}}")
	if err != nil {
		c.Fatalf("Could not inspect volume: %v %s", err, mountPoint)
	}
	mountPoint = strings.TrimSpace(mountPoint)

	out, err = s.d.Cmd("run", "--rm", "-v", volName+":"+destDir, "busybox", "touch", destDir+destFile)
	c.Assert(err, checker.IsNil, check.Commentf(out))
	path := filepath.Join(s.d.RootDir(), "plugins", pluginID, "rootfs", mountPoint, destFile)
	_, err = os.Lstat(path)
	c.Assert(err, checker.IsNil)

	exists, err := existsMountpointWithPrefix(mountpointPrefix)
	c.Assert(err, checker.IsNil)
	c.Assert(exists, checker.Equals, true)
}

func existsMountpointWithPrefix(mountpointPrefix string) (bool, error) {
	mounts, err := mount.GetMounts()
	if err != nil {
		return false, err
	}
	for _, mnt := range mounts {
		if strings.HasPrefix(mnt.Mountpoint, mountpointPrefix) {
			return true, nil
		}
	}
	return false, nil
}
