package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Envvars(envs []string, envvars map[string]string, dir string) error {
	for _, e := range envs {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envvars[pair[0]] = pair[1]
		} else if strings.HasSuffix(e, "-") {
			delete(envvars, strings.TrimSuffix(e, "-"))
		} else {
			fmt.Println(envvars[e])
		}
	}

	envvarStr := ""

	for k, v := range envvars {
		// Actually set the variables as a sanity check before trying to
		// commit potentially bad names.
		err := os.Setenv(k, v)
		if err != nil {
			return err
		}
		envvarStr += k + "=" + v + "\n"
	}

	envvarFile := filepath.Join(dir, ".estragon", "envvars")
	return os.WriteFile(envvarFile, []byte(envvarStr), 0666)
}
