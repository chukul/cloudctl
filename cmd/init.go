package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate shell integration code",
	Long:  `Generate shell integration code to simplify cloudctl usage. Add the output to your shell config file.`,
	Run: func(cmd *cobra.Command, args []string) {
		shell := detectShell()

		fmt.Printf("# CloudCtl Shell Integration for %s\n", shell)
		fmt.Println("# Add this to your shell config file:")
		fmt.Println("# - Bash: ~/.bashrc or ~/.bash_profile")
		fmt.Println("# - Zsh: ~/.zshrc")
		fmt.Println("# - Fish: ~/.config/fish/config.fish")
		fmt.Println()

		switch shell {
		case "bash", "zsh":
			printBashZshIntegration()
		case "fish":
			printFishIntegration()
		default:
			printBashZshIntegration()
		}
	},
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		if runtime.GOOS == "windows" {
			return "powershell"
		}
		return "bash"
	}

	// Extract shell name from path
	for i := len(shell) - 1; i >= 0; i-- {
		if shell[i] == '/' || shell[i] == '\\' {
			return shell[i+1:]
		}
	}
	return shell
}

func printBashZshIntegration() {
	fmt.Println(`# Set your CloudCtl encryption secret
export CLOUDCTL_SECRET="your-32-char-encryption-key"

# Quick switch function - usage: ccs <profile>
ccs() {
  if [ -z "$1" ]; then
    # Interactive mode
    eval $(cloudctl switch)
  else
    # Direct profile switch
    eval $(cloudctl switch "$1")
  fi
}

# Show current session in prompt (optional)
cloudctl_prompt() {
  cloudctl prompt 2>/dev/null
}

# Add to your PS1 (Bash) or PROMPT (Zsh):
# PS1='$(cloudctl_prompt) \u@\h:\w\$ '
# PROMPT='$(cloudctl_prompt) %n@%m:%~%# '

# Aliases for common commands
alias ccl='cloudctl login'
alias ccst='cloudctl status'
alias ccr='cloudctl refresh'
alias ccc='cloudctl console'
alias ccm='cloudctl mfa-login'`)
}

func printFishIntegration() {
	fmt.Println(`# Set your CloudCtl encryption secret
set -gx CLOUDCTL_SECRET "your-32-char-encryption-key"

# Quick switch function - usage: ccs <profile>
# Quick switch function - usage: ccs <profile>
function ccs
    if test (count $argv) -eq 0
        # Interactive mode
        eval (cloudctl switch)
    else
        # Direct profile switch
        eval (cloudctl switch $argv[1])
    end
end

# Show current session in prompt (optional)
function fish_prompt
    set_color green
    cloudctl prompt 2>/dev/null
    set_color normal
    echo -n ' '
    set_color blue
    echo -n (whoami)@(hostname):(prompt_pwd)
    set_color normal
    echo -n '> '
end

# Aliases for common commands
alias ccl='cloudctl login'
alias ccst='cloudctl status'
alias ccr='cloudctl refresh'
alias ccc='cloudctl console'
alias ccm='cloudctl mfa-login'`)
}

func init() {
	rootCmd.AddCommand(initCmd)
}
