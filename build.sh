#!/bin/sh

#Set fonts for Help.
NORM=`tput sgr0`
BOLD=`tput bold`
REV=`tput smso`
FG_GREEN="$(tput setaf 2)"
FG_RED="$(tput setaf 1)"

# get script's location path
SCRIPT=`basename ${BASH_SOURCE[0]}`
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
pushd $DIR > /dev/null

function HELP () {
  read -d '' help <<- EOF
Builds sensors utility application
EOF

  echo \\n"$help"\\n
}

docker-compose -f compile.yml --log-level ERROR up
RET=$?
if [[ $RET -eq 0 ]]; then
  echo "Compiling ... ${FG_GREEN}done${NORM}"
else
  echo "Compiling ... ${FG_RED}error:$RET${NORM}"
  docker-compose -f compile.yml --log-level ERROR rm -vf
  popd > /dev/null
  exit 1
fi

popd > /dev/null