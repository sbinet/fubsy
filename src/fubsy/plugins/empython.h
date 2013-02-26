#ifndef EMPYTHON_H
#define EMPYTHON_H

typedef struct {
    char *name;
    void *gofunc;
} builtin_t;

void
setCallback(int idx, void *gofunc);

int
installBuiltins();

#endif
