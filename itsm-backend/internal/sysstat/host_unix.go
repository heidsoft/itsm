//go:build unix

package sysstat

import "syscall"

// processCPUTimeSecs 返回进程（含已回收子进程）累计的用户态+内核态 CPU 时间秒数。
func processCPUTimeSecs() (float64, bool) {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0, false
	}
	user := float64(ru.Utime.Sec) + float64(ru.Utime.Usec)/1e6
	system := float64(ru.Stime.Sec) + float64(ru.Stime.Usec)/1e6
	return user + system, true
}

// diskUsage 返回 path 所在文件系统的总字节数与可用字节数。
func diskUsage(path string) (total uint64, free uint64, ok bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, false
	}
	bsize := uint64(st.Bsize)
	if bsize == 0 {
		return 0, 0, false
	}
	total = st.Blocks * bsize
	// 预留块（Blocks-Bfree-Bavail）普通用户不可写，df 的 Use% 也把其算作已用。
	used := (st.Blocks - st.Bfree) * bsize
	free = st.Bavail * bsize
	if used > total {
		used = total
	}
	return total, total - used, true
}
