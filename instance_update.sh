#!/bin/bash
source <( curl "http://169.254.169.254/latest/user-data" 2>/dev/null ) 
 
if [ "${GO_GIT_HASH}" != "" ]; then
  pushd $PROJECT_DIR >/dev/null 2>/dev/null
  su node -c "vendor/github.com/rem7/servercontrol/git_update_to_hash.sh ${GO_GIT_HASH}" >>/tmp/instance-update.log 2>&1
  popd >/dev/null 2>/dev/null
  /usr/bin/sv restart $GO_PROJECT
fi

