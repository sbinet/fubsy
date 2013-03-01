#ifndef EMPYTHON_H
#define EMPYTHON_H

typedef struct {
    char *name;
    void *gofunc;
} builtin_t;

typedef struct {
    int numitems;
    char **names;
    PyObject **values;
} valuelist_t;

typedef struct {
    void *data;
    uint32_t len;
    uint32_t cap;
} goslice_t;

typedef struct {
    uint8_t *bytes;
    uint32_t len;
    uint32_t refs;              /* not sure what this is */
} gostring_t;

void
set_callback(int idx, void *gofunc);

int
install_builtins();

int
run_python(char *content, char **error, valuelist_t **exported);

char *
call_python(PyObject *callable, void *args, char **error);

void
free_valuelist(valuelist_t *list);

#endif
