package git

import (
	"fmt"
	"os"
	"os/exec"
)

func getGitCommandEnv(name, email string) []string {
	return append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		fmt.Sprintf("GIT_AUTHOR_NAME=%s", name),
		fmt.Sprintf("GIT_AUTHOR_EMAIL=%s", email),
		fmt.Sprintf("GIT_COMMITTER_NAME=%s", name),
		fmt.Sprintf("GIT_COMMITTER_EMAIL=%s", email),
	)
}

func runGitCommand(dir string, args []string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = getGitCommandEnv("test", "test@example.com")
	return cmd.Run()
}
