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
	// 预留块（Blocks-Bfree-Bavail）普通用户不可写，df 的 Use% 也把其算作已用，
	// 因此这里返回「含预留的剩余」（Bfree 口径）：free = total - used。
	// 注意不能返回 st.Bavail*bsize —— 那是普通用户可用量，调用方按
	// used = total - free 反算，会把预留块算成已用，Use% 反而高于 df。
	used := (st.Blocks - st.Bfree) * bsize
	if used > total {
		used = total
	}
	free = total - used
	return total, free, true
}
