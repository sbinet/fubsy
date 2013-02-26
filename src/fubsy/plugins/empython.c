// +build python

#include <Python.h>
#include <assert.h>
#include "empython.h"
#include "_cgo_export.h"

static PyObject*
call_builtin(void *gofunc, PyObject *self, PyObject *args) {
    PyObject *arg, *ret = NULL;
    Py_ssize_t numargs, i;
    char **sargs;
    struct callBuiltin_return result = {NULL, NULL};

    /* convert the Python argument tuple to an array of C strings */
    numargs = PyTuple_GET_SIZE(args);
    sargs = malloc(numargs * sizeof(char*));
    if (!sargs) {
        PyErr_SetNone(PyExc_MemoryError);
        goto done;
    }
    for (i = 0; i < numargs; i++) {
        arg = PyTuple_GET_ITEM(args, i);
        if (!arg) {
            goto done;
        }
        if (!PyString_Check(arg)) {
            PyErr_SetString(PyExc_TypeError, "all arguments must be strings");
            goto done;
        }
        sargs[i] = PyString_AS_STRING(arg);  /* not a copy! */
    }

    /* call the desired builtin via callBuiltin() (exported from Go) */
    result = callBuiltin(gofunc, numargs, sargs);

    /* turn error return (r1) into a Python exception; convert return
       value (r0) to Python string */
    if (result.r1) {
        PyErr_SetString(PyExc_RuntimeError, result.r1);
        goto done;
    }
    if (result.r0) {
        ret = PyString_FromString(result.r0);
        goto done;
    } else {
        Py_INCREF(Py_None);
        ret = Py_None;
    }

 done:
    free(result.r0);
    free(result.r1);
    free(sargs);
    return ret;
}

/* gofunc members are set by InstallBuiltins() in python.go */
static builtin_t builtins[] = {
    {"println", NULL},
    {"mkdir", NULL},
    {"remove", NULL},
    {"build", NULL},
    {"FileNode", NULL},
    {"ActionNode", NULL},
};

static PyObject*
py_println(PyObject *self, PyObject *args) {
    return call_builtin(builtins[0].gofunc, self, args);
}

static PyObject*
py_mkdir(PyObject *self, PyObject *args) {
    return call_builtin(builtins[1].gofunc, self, args);
}

static PyObject*
py_remove(PyObject *self, PyObject *args) {
    return call_builtin(builtins[2].gofunc, self, args);
}

static PyMethodDef methods[] = {
    {"println", py_println, METH_VARARGS, NULL},
    {"mkdir", py_mkdir, METH_VARARGS, NULL},
    {"remove", py_remove, METH_VARARGS, NULL},
    {NULL, NULL, 0, NULL},
};

void
setCallback(int idx, void *gofunc) {
    int max = sizeof(builtins) / sizeof(builtins[0]);
    assert(idx < max);
    //assert(strcmp(builtins[idx].name, name) == 0);
    builtins[idx].gofunc = gofunc;
}

int
installBuiltins() {
    PyObject *fubsy, *main;

    fubsy = Py_InitModule("fubsy", methods);
    if (!fubsy) {
        return -1;
    }

    /* "import fubsy" in __main__, so it's visible to inline plugins for free */
    main = PyImport_ImportModule("__main__");
    if (!main) {
        return -1;
    }
    Py_INCREF(fubsy);                         /* AddObject() steals a ref */
    if (PyModule_AddObject(main, "fubsy", fubsy) < 0) {
        return -1;
    }

    Py_DECREF(main);
    return 0;
}
