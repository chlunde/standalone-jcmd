# standalone-jcmd
Standalone jcmd

# Why?

```bash
sh-4.2$ ./jcmd
./jcmd: error while loading shared libraries: libjli.so: cannot open shared object file: No such file or directory
```
copying `libjli.so` does not seem to work, with a number of different combinations of `LD_LIBRARY_PATH` and `JAVA_HOME`:

```bash
sh-4.2$ LD_LIBRARY_PATH=.:.... JAVA_HOME=... ./jcmd
Error: could not find libjava.so
Error: Could not find Java SE Runtime Environment.
```

# Usage

```bash
make

oc cp -n $NAMESPACE standalone-jcmd $CONTAINER:/tmp

ps -ef

# assuming Java is PID 1:
cd /tmp; ./standalone-jcmd 1 help

./standalone-jcmd 1 GC.class_histogram
```
