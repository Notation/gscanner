GSCANNER=gscanner

ifdef outbin
BIN=$(outbin)
else
BIN=bin
endif

gscanner:
	scripts/build.sh -o ${BIN}/gscanner ${GSCANNER}/cmd

clean:
	@rm -rf ${BIN}/gscanner
