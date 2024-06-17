package nsenter

/*
#include "nsexec.h"

void __attribute__((constructor)) init(void) {
	nsexec();
}
*/
import "C"

