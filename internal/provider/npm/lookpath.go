package npm

import "os/exec"

// lookPath wraps exec.LookPath so it can be overridden in tests.
var lookPath = exec.LookPath
