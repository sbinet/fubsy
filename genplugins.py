#!/usr/bin/python

"""Generate code in src/fubsy/plugins based on information parsed from
other Fubsy source files. Most notably, generate C code that wraps Fubsy
builtin functions."""

import sys
import os
import re

BUILTIN_RE = re.compile(r'func fn_(\w+)\(\w+ types\.ArgSource\)')
SUBST_RE = re.compile(r'@([A-Za-z_]+)@\s*$')

def readbuiltins(filename):
    with open(filename) as file:
        for line in file:
            match = BUILTIN_RE.match(line)
            if match:
                yield match.group(1)

class Generator(object):
    def __init__(self, outfn, builtins):
        self.outfn = outfn
        self.builtins = builtins

    def __enter__(self):
        self.infile = open(self.outfn + ".in")
        self.outfile = open(self.outfn, "w")
        return self

    def __exit__(self, etype, evalue, tb):
        self.infile.close()
        self.outfile.close()
        if etype:
            os.remove(self.outfn)

    def generate(self):
        lineno = 0
        outfile = self.outfile
        for line in self.infile:
            lineno += 1
            match = SUBST_RE.match(line)
            if match:
                try:
                    method = getattr(self, "gen_" + match.group(1))
                except AttributeError:
                    raise RuntimeError(
                        "%s, line %d: invalid substitution: @%s@"
                        % (self.infile.name, lineno, match.group(1)))
                method(outfile)
                outfile.write("#line %d \"%s\"\n" %
                              (lineno+1, self.infile.name))
            else:
                outfile.write(line)

    def gen_builtins(self, outfile):
        outfile.write("static builtin_t builtins[] = {\n")
        for name in self.builtins:
            outfile.write("    {\"%s\", NULL},\n" % name)
        outfile.write("};\n")

    def gen_pycfunctions(self, outfile):
        for (idx, name) in enumerate(self.builtins):
            outfile.write("""
static PyObject*
py_%s(PyObject *self, PyObject *args) {
    return call_builtin(builtins[%d].gofunc, self, args);
}
""" % (name, idx))

    def gen_pymethodtable(self, outfile):
        outfile.write("static PyMethodDef methods[] = {\n")
        for name in self.builtins:
            outfile.write("    {\"%s\", py_%s, METH_VARARGS, NULL},\n" %
                          (name, name))
        outfile.write("    {NULL, NULL, 0, NULL}\n")
        outfile.write("};\n")

def main():
    args = sys.argv[1:]
    if len(args) < 2:
        sys.exit("usage: %s builtins_file outfile...\n\n"
                 "error: wrong number of arguments" % sys.argv[0])

    try:
        builtins = list(readbuiltins(args[0]))
        for outfn in args[1:]:
            with Generator(outfn, builtins) as g:
                g.generate()
    except (IOError, RuntimeError) as err:
        sys.exit("error: " + str(err))

main()
