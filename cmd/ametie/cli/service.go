package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage service",
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start service",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting service...")
		var err error
		switch runtime.GOOS {
		case "linux":
			err = exec.Command("systemctl", "start", "ametie").Run()
		case "darwin":
			err = exec.Command("launchctl", "start", "com.ametie").Run()
		case "windows":
			err = exec.Command("sc", "start", "Ametie").Run()
			if err != nil {
				err = exec.Command("powershell", "-Command", "Start-Service", "Ametie").Run()
			}
		default:
			fmt.Println("Unsupported OS")
			return
		}
		if err != nil {
			fmt.Printf("Error starting service: %v\n", err)
		} else {
			fmt.Println("Service started")
		}
	},
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop service",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Stopping service...")
		var err error
		switch runtime.GOOS {
		case "linux":
			err = exec.Command("systemctl", "stop", "ametie").Run()
		case "darwin":
			err = exec.Command("launchctl", "stop", "com.ametie").Run()
		case "windows":
			err = exec.Command("sc", "stop", "Ametie").Run()
			if err != nil {
				err = exec.Command("powershell", "-Command", "Stop-Service", "Ametie").Run()
			}
		default:
			fmt.Println("Unsupported OS")
			return
		}
		if err != nil {
			fmt.Printf("Error stopping service: %v\n", err)
		} else {
			fmt.Println("Service stopped")
		}
	},
}

var serviceRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart service",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Restarting service...")
		var err error
		switch runtime.GOOS {
		case "linux":
			err = exec.Command("systemctl", "restart", "ametie").Run()
		case "darwin":
			exec.Command("launchctl", "stop", "com.ametie").Run()
			time.Sleep(1 * time.Second)
			err = exec.Command("launchctl", "start", "com.ametie").Run()
		case "windows":
			err = exec.Command("sc", "stop", "Ametie").Run()
			time.Sleep(1 * time.Second)
			err = exec.Command("sc", "start", "Ametie").Run()
			if err != nil {
				exec.Command("powershell", "-Command", "Restart-Service", "Ametie").Run()
			}
		default:
			fmt.Println("Unsupported OS")
			return
		}
		if err != nil {
			fmt.Printf("Error restarting service: %v\n", err)
		} else {
			fmt.Println("Service restarted")
		}
	},
}

var serviceLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View service logs",
	Run: func(cmd *cobra.Command, args []string) {
		tail, _ := cmd.Flags().GetInt("tail")
		var cmdExec *exec.Cmd

		switch runtime.GOOS {
		case "linux":
			if tail > 0 {
				cmdExec = exec.Command("journalctl", "-u", "ametie", "-n", fmt.Sprintf("%d", tail), "--no-pager")
			} else {
				cmdExec = exec.Command("journalctl", "-u", "ametie", "--no-pager")
			}
		case "darwin":
			if tail > 0 {
				cmdExec = exec.Command("tail", "-n", fmt.Sprintf("%d", tail), "/var/log/ametie.log")
			} else {
				cmdExec = exec.Command("cat", "/var/log/ametie.log")
			}
		case "windows":
			if tail > 0 {
				cmdExec = exec.Command("powershell", "-Command", fmt.Sprintf("Get-EventLog -LogName Application -Source Ametie -Newest %d | Format-Table -AutoSize", tail))
			} else {
				cmdExec = exec.Command("powershell", "-Command", "Get-EventLog -LogName Application -Source Ametie | Format-Table -AutoSize")
			}
		default:
			fmt.Println("Unsupported OS")
			return
		}

		cmdExec.Stdout = os.Stdout
		cmdExec.Stderr = os.Stderr
		if err := cmdExec.Run(); err != nil {
			fmt.Printf("Error viewing logs: %v\n", err)
			fmt.Println("Note: Logs may not be available if service is not running or not configured")
		}
	},
}

func init() {
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceRestartCmd)
	serviceCmd.AddCommand(serviceLogsCmd)
	serviceLogsCmd.Flags().Int("tail", 0, "Number of lines to show")
}
