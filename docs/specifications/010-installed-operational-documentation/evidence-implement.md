# Implement Evidence 010 — P1 checkpoint (T02 macOS arm64 partial)

**Status: BLOCKED at G1, 2026-10-20.** Base head `088eb89`. Only T02 on macOS arm64
was executed; everything else is NOT_RUN. Not product Evidence for AC-01–AC-08.

## Task status

| Task | Status | Reason |
| --- | --- | --- |
| T01 | NOT_RUN | Wire/version/quota values are D1/D3 human-review material; no value invented |
| T02 | PARTIAL — macOS arm64 only (exploratory, run before T01 freeze) | Linux amd64/arm64 hosts unavailable here (no Docker/VM/Linux host) |
| T03–T16 | NOT_RUN | Blocked by G1 (T04) |

## T02 macOS arm64 result (E02, partial)

Host: Darwin 27.0.0 arm64, APFS, Apple clang. Harness: disposable C program below,
run from a path containing spaces, via a symlink alias, with a file-gate pause.

| Case | Observed |
| --- | --- |
| A running, pathname atomically replaced by B (`mv`), A resumed | `proc_pidinfo(PROC_PIDREGIONPATHINFO)` text-region vnode dev/ino == A's start identity; `proc_pidpath` still returns the old pathname; `stat(pathname)` now yields B's inode |
| Fresh B | region identity == B |
| A unlinked, replaced by copy B at same pathname | region identity still A; pathname inode differs |
| Pin-at-start `open()`+`fstat` | equals A's identity (but opened after exec: TOCTOU window not measured) |

Conclusions (macOS arm64 only): pathname-based proof (hash of current path, or
`proc_pidpath`) fails the oracle as required; the text-region vnode identity of the
live process is a viable independent OS proof candidate. **Not tested:** hard link /
unsafe alias refusal, switched PATH, relocation/rebind, changed ancestry, inode reuse,
reader pinning between inventory validation and content open, non-APFS filesystems,
Linux amd64, Linux arm64.

## Blocker and smallest human decision

D2 cannot be proven on all three host rows from this session. Per the AXM-37 stop
rule, dependent work (T03–T16) is stopped and no fallback is adopted. Needed: access
to Linux amd64 and Linux arm64 hosts (or CI native runners authorized for T02) —
Linux candidate to falsify: `open("/proc/self/exe")` / `readlink` after replacement.
Also required: human review of T01 contract values.

## Harness source (`img.c`)

```c
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <fcntl.h>
#include <string.h>
#include <sys/stat.h>
#include <libproc.h>
#include <mach-o/dyld.h>
#ifndef TAG
#define TAG "A"
#endif
static int region(pid_t pid, unsigned long long *dev, unsigned long long *ino, char *path){
  uint64_t addr=0;
  for(int i=0;i<64;i++){
    struct proc_regionwithpathinfo r; memset(&r,0,sizeof r);
    int n=proc_pidinfo(pid,PROC_PIDREGIONPATHINFO,addr,&r,sizeof r);
    if(n<(int)sizeof r) return -1;
    if(r.prp_vip.vip_path[0]){ *dev=r.prp_vip.vip_vi.vi_stat.vst_dev; *ino=r.prp_vip.vip_vi.vi_stat.vst_ino; strcpy(path,r.prp_vip.vip_path); return 0;}
    addr=r.prp_prinfo.pri_address+r.prp_prinfo.pri_size;
  }
  return -1;
}
int main(int argc,char**argv){
  char *exe=argv[1]; char *gate=argv[2];
  int pin=open(exe,O_RDONLY); struct stat ps; fstat(pin,&ps);
  unsigned long long d0,i0; char p0[1024]; int r0=region(getpid(),&d0,&i0,p0);
  while(access(gate,F_OK)!=0) usleep(20000);
  unsigned long long d,i; char p[1024]; int r=region(getpid(),&d,&i,p);
  char pp[1024]={0}; proc_pidpath(getpid(),pp,sizeof pp);
  struct stat cur; int sc=stat(exe,&cur);
  printf("%s start: region rc=%d dev=%llu ino=%llu | pinfd dev=%llu ino=%llu\n",TAG,r0,d0,i0,(unsigned long long)ps.st_dev,(unsigned long long)ps.st_ino);
  printf("%s resume: region rc=%d dev=%llu ino=%llu path=%s | proc_pidpath=%s\n",TAG,r,d,i,p,pp);
  printf("%s pathname-now: rc=%d dev=%llu ino=%llu\n",TAG,sc,(unsigned long long)cur.st_dev,(unsigned long long)cur.st_ino);
  printf("%s verdict: region==start:%d pin==start:%d pathnow==start:%d\n",TAG,(d==d0&&i==i0),(ps.st_ino==i0),(cur.st_ino==i0));
  return 0;}
```

## Later checkpoint

2026-10-20 resumption: see [Resumption 1 Evidence](evidence-resume-1.md) for the
Proposed D1/D3 contract gate, native Linux probes and current limitations.
The measurements and blockers above remain the historical first-run snapshot.
