#define _GNU_SOURCE  // 启用 GNU 扩展
#include "nsexec.h"
#include "stdio.h"
#include "stdarg.h"
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <sched.h>
#include "fcntl.h"

#define DEBUG 0
#define INFO 1
#define WARN 2
#define CONTAINERIDENV "my_container_id"
#define CONTAINERCMDENV "my_container_env"

void logging(int logType, const char *format, ...)
{
    va_list args;
    va_start(args, format);

    // 获取格式化后的字符串长度
    int len = vsnprintf(NULL, 0, format, args);
    va_end(args);

    // 分配足够的内存空间
    char *msg = (char *)malloc(len + 1);
    if (msg == NULL)
    {
        fprintf(stderr, "Memory allocation failed\n");
        return;
    }

    va_start(args, format);
    vsnprintf(msg, len + 1, format, args); // +1 用于空字符结尾
    va_end(args);

    fprintf(stdout, "INFO: %s\n", msg);
    fflush(stdout);

    free(msg);
}

void nsexec()
{
    char *container_pid = getenv(CONTAINERIDENV);
    if (!container_pid)
    {
        return;
    }
    char *exce_cmd = getenv(CONTAINERCMDENV);
    if (!exce_cmd)
    {
        return;
    }
    logging(DEBUG, "pid %s cmd %s", container_pid, exce_cmd);
    // 要进入的五种Namespace
    char *namespaces[] = {"ipc", "uts", "net", "pid", "mnt"};
    char nspath[1024] = {0};
    for (size_t i = 0; i < 5; i++)
    {
        sprintf(nspath, "/proc/%s/ns/%s", container_pid, namespaces[i]);
        int fd = open(nspath, O_RDONLY);
        if (fd == -1)
        {
            logging(WARN, "nspath %s open error %s", nspath, strerror(errno));
            continue;
        }
        if (setns(fd, 0) == -1)
        {
            logging(WARN, "setns %s error %s", nspath, strerror(errno));
        }
    }
    int res = system(exce_cmd);
    exit(0);
    return;
}
