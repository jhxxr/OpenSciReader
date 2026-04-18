//go:build !windows

package translator

func configureWorkerProcess(cmd *exec.Cmd) {}
